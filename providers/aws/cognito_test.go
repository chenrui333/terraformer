// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
	identitytypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentity/types"
	idptypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestCognitoResourceIDs(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "user group", got: cognitoUserGroupResourceID("us-east-1_abc", "admins"), want: "us-east-1_abc/admins"},
		{name: "identity provider", got: cognitoIdentityProviderResourceID("us-east-1_abc", "Google"), want: "us-east-1_abc:Google"},
		{name: "resource server", got: cognitoResourceServerResourceID("us-east-1_abc", "https://example.com"), want: "us-east-1_abc|https://example.com"},
		{name: "resource name skips empty parts", got: cognitoResourceName("pool", "", "client", "web"), want: "pool:client:web"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestCognitoIdentityPoolRolesAttachmentConfigured(t *testing.T) {
	tests := []struct {
		name                string
		output              *cognitoidentity.GetIdentityPoolRolesOutput
		wantConfigured      bool
		wantNeedsEmptyRoles bool
	}{
		{name: "nil", output: nil},
		{name: "empty", output: &cognitoidentity.GetIdentityPoolRolesOutput{}},
		{
			name: "roles",
			output: &cognitoidentity.GetIdentityPoolRolesOutput{
				Roles: map[string]string{"authenticated": "arn:aws:iam::123456789012:role/auth"},
			},
			wantConfigured: true,
		},
		{
			name: "role mappings only",
			output: &cognitoidentity.GetIdentityPoolRolesOutput{
				RoleMappings: map[string]identitytypes.RoleMapping{"accounts.google.com": {}},
			},
			wantConfigured:      true,
			wantNeedsEmptyRoles: true,
		},
		{
			name: "roles and role mappings",
			output: &cognitoidentity.GetIdentityPoolRolesOutput{
				Roles:        map[string]string{"authenticated": "arn:aws:iam::123456789012:role/auth"},
				RoleMappings: map[string]identitytypes.RoleMapping{"accounts.google.com": {}},
			},
			wantConfigured: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cognitoIdentityPoolRolesAttachmentConfigured(tt.output); got != tt.wantConfigured {
				t.Fatalf("cognitoIdentityPoolRolesAttachmentConfigured() = %t, want %t", got, tt.wantConfigured)
			}
			if got := cognitoIdentityPoolRolesAttachmentNeedsEmptyRoles(tt.output); got != tt.wantNeedsEmptyRoles {
				t.Fatalf("cognitoIdentityPoolRolesAttachmentNeedsEmptyRoles() = %t, want %t", got, tt.wantNeedsEmptyRoles)
			}
		})
	}
}

func TestCognitoIdentityPoolRolesAttachmentPreservesMappingOnlyRoles(t *testing.T) {
	identityPool := cognitoIdentityPoolRef{
		id:   "us-east-1:11111111-1111-1111-1111-111111111111",
		name: "orders",
	}
	resource := newCognitoIdentityPoolRolesAttachmentResource(identityPool, true)
	roles, ok := resource.AdditionalFields["roles"].(map[string]interface{})
	if !ok {
		t.Fatalf("roles additional field = %#v, want empty map", resource.AdditionalFields["roles"])
	}
	if len(roles) != 0 {
		t.Fatalf("roles additional field len = %d, want 0", len(roles))
	}

	resource = newCognitoIdentityPoolRolesAttachmentResource(identityPool, false)
	if _, ok := resource.AdditionalFields["roles"]; ok {
		t.Fatal("roles additional field exists for non-mapping-only attachment")
	}
}

func TestCognitoResourceMissing(t *testing.T) {
	tests := []struct {
		name string
		fn   func(error) bool
		err  error
		want bool
	}{
		{name: "identity nil", fn: cognitoIdentityResourceMissing, err: nil},
		{name: "identity missing", fn: cognitoIdentityResourceMissing, err: &identitytypes.ResourceNotFoundException{}, want: true},
		{name: "identity wrapped missing", fn: cognitoIdentityResourceMissing, err: errors.Join(errors.New("wrapper"), &identitytypes.ResourceNotFoundException{}), want: true},
		{name: "identity generic", fn: cognitoIdentityResourceMissing, err: errors.New("boom")},
		{name: "idp nil", fn: cognitoIDPResourceMissing, err: nil},
		{name: "idp missing", fn: cognitoIDPResourceMissing, err: &idptypes.ResourceNotFoundException{}, want: true},
		{name: "idp wrapped missing", fn: cognitoIDPResourceMissing, err: errors.Join(errors.New("wrapper"), &idptypes.ResourceNotFoundException{}), want: true},
		{name: "idp generic", fn: cognitoIDPResourceMissing, err: errors.New("boom")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(tt.err); got != tt.want {
				t.Fatalf("missing helper = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestCognitoUserPoolDomainMetadataError(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name    string
		filters []terraformutils.ResourceFilter
		err     error
		wantErr bool
	}{
		{
			name: "broad discovery logs domain metadata error",
			err:  boom,
		},
		{
			name: "typed domain filter returns domain metadata error",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: cognitoUserPoolDomainResourceType, FieldPath: "id", AcceptableValues: []string{"auth.example.com"}},
			},
			err:     boom,
			wantErr: true,
		},
		{
			name: "typed domain filter ignores missing user pool",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: cognitoUserPoolDomainResourceType, FieldPath: "id", AcceptableValues: []string{"auth.example.com"}},
			},
			err: &idptypes.ResourceNotFoundException{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := CognitoGenerator{}
			g.Filter = tt.filters
			err := g.handleUserPoolDomainMetadataError("us-east-1_abc", tt.err)
			if tt.wantErr {
				if !errors.Is(err, boom) {
					t.Fatalf("handleUserPoolDomainMetadataError() error = %v, want %v", err, boom)
				}
				return
			}
			if err != nil {
				t.Fatalf("handleUserPoolDomainMetadataError() error = %v, want nil", err)
			}
		})
	}
}

func TestCognitoOptionalResourceLoaderErrors(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name         string
		filters      []terraformutils.ResourceFilter
		serviceNames []string
		required     bool
		wantErr      bool
	}{
		{
			name:         "optional loader error is logged for broad discovery",
			serviceNames: []string{cognitoUserGroupResourceType},
		},
		{
			name: "typed child filter returns loader error",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: cognitoUserGroupResourceType, FieldPath: "id", AcceptableValues: []string{"us-east-1_abc/admins"}},
			},
			serviceNames: []string{cognitoUserGroupResourceType},
			wantErr:      true,
		},
		{
			name:         "required loader returns error without typed filter",
			serviceNames: []string{cognitoUserPoolClientResourceType},
			required:     true,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := CognitoGenerator{}
			g.Filter = tt.filters
			called := false
			err := g.loadOptionalResources([]cognitoOptionalResourceLoader{
				{
					name:         "user groups",
					serviceNames: tt.serviceNames,
					required:     tt.required,
					load: func() error {
						called = true
						return boom
					},
				},
			})
			if !called {
				t.Fatal("loader was not called")
			}
			if tt.wantErr {
				if !errors.Is(err, boom) {
					t.Fatalf("loadOptionalResources() error = %v, want %v", err, boom)
				}
				return
			}
			if err != nil {
				t.Fatalf("loadOptionalResources() error = %v, want nil", err)
			}
		})
	}
}

func TestCognitoInitialCleanupPreservesUserPoolClientImportIDFilter(t *testing.T) {
	userPoolID := "us-east-1_abc"
	clientID := "client123"
	resource := newCognitoUserPoolClientResource(userPoolID, clientID, "web")

	tests := []struct {
		name        string
		filterValue string
		wantCount   int
	}{
		{
			name:        "full import ID filter keeps client resource",
			filterValue: cognitoUserPoolClientImportID(userPoolID, clientID),
			wantCount:   1,
		},
		{
			name:        "client ID filter keeps client resource",
			filterValue: clientID,
			wantCount:   1,
		},
		{
			name:        "nonmatching full import ID filter removes client resource",
			filterValue: cognitoUserPoolClientImportID(userPoolID, "other"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := CognitoGenerator{}
			g.Resources = []terraformutils.Resource{resource}
			g.Filter = []terraformutils.ResourceFilter{
				{ServiceName: cognitoUserPoolClientResourceType, FieldPath: "id", AcceptableValues: []string{tt.filterValue}},
			}
			g.InitialCleanup()
			if got := len(g.Resources); got != tt.wantCount {
				t.Fatalf("len(Resources) = %d, want %d", got, tt.wantCount)
			}
		})
	}
}

func TestCognitoFilterGatesUserPoolsAndChildren(t *testing.T) {
	userPoolID := "us-east-1_abc"
	otherUserPoolID := "us-east-1_def"
	userPool := newCognitoUserPoolResource(cognitoUserPoolRef{id: userPoolID, name: "orders"})
	otherUserPool := newCognitoUserPoolResource(cognitoUserPoolRef{id: otherUserPoolID, name: "other"})
	client := newCognitoUserPoolClientResource(userPoolID, "client123", "web")
	group := newCognitoUserGroupResource(userPoolID, "admins")

	tests := []struct {
		name             string
		filters          []terraformutils.ResourceFilter
		loadUserPools    bool
		appendUserPool   bool
		appendOtherPool  bool
		loadClients      bool
		loadOtherClients bool
		appendClient     bool
		appendGroup      bool
	}{
		{
			name:             "no filters imports user pools and children",
			loadUserPools:    true,
			appendUserPool:   true,
			appendOtherPool:  true,
			loadClients:      true,
			loadOtherClients: true,
			appendClient:     true,
			appendGroup:      true,
		},
		{
			name: "typed user pool id filter limits parent output",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: cognitoUserPoolResourceType, FieldPath: "id", AcceptableValues: []string{userPoolID}},
			},
			loadUserPools:  true,
			appendUserPool: true,
			loadClients:    true,
		},
		{
			name: "typed child id filter skips parent output",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: cognitoUserPoolClientResourceType, FieldPath: "id", AcceptableValues: []string{cognitoUserPoolClientImportID(userPoolID, "client123")}},
			},
			loadUserPools: true,
			loadClients:   true,
			appendClient:  true,
		},
		{
			name: "typed parent id filter scopes typed child non-id discovery",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: cognitoUserPoolResourceType, FieldPath: "id", AcceptableValues: []string{userPoolID}},
				{ServiceName: cognitoUserPoolClientResourceType, FieldPath: "name", AcceptableValues: []string{"web"}},
			},
			loadUserPools:  true,
			appendUserPool: true,
			loadClients:    true,
			appendClient:   true,
		},
		{
			name: "typed child id filter can load outside parent filter",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: cognitoUserPoolResourceType, FieldPath: "id", AcceptableValues: []string{otherUserPoolID}},
				{ServiceName: cognitoUserPoolClientResourceType, FieldPath: "id", AcceptableValues: []string{cognitoUserPoolClientImportID(userPoolID, "client123")}},
			},
			loadUserPools:   true,
			appendOtherPool: true,
			loadClients:     true,
			appendClient:    true,
		},
		{
			name: "typed user pool non-id filter avoids child pre-load",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: cognitoUserPoolResourceType, FieldPath: "tags.env", AcceptableValues: []string{"prod"}},
			},
			loadUserPools:   true,
			appendUserPool:  true,
			appendOtherPool: true,
		},
		{
			name: "typed user pool non-id filter does not block typed child non-id discovery",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: cognitoUserPoolResourceType, FieldPath: "tags.env", AcceptableValues: []string{"prod"}},
				{ServiceName: cognitoUserPoolClientResourceType, FieldPath: "name", AcceptableValues: []string{"web"}},
			},
			loadUserPools:    true,
			appendUserPool:   true,
			appendOtherPool:  true,
			loadClients:      true,
			loadOtherClients: true,
			appendClient:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := CognitoGenerator{}
			g.Filter = tt.filters
			if got := g.shouldLoadUserPools(); got != tt.loadUserPools {
				t.Fatalf("shouldLoadUserPools() = %t, want %t", got, tt.loadUserPools)
			}
			if got := g.shouldAppendCognitoResource(cognitoUserPoolResourceType, userPool); got != tt.appendUserPool {
				t.Fatalf("shouldAppendCognitoResource(user pool) = %t, want %t", got, tt.appendUserPool)
			}
			if got := g.shouldAppendCognitoResource(cognitoUserPoolResourceType, otherUserPool); got != tt.appendOtherPool {
				t.Fatalf("shouldAppendCognitoResource(other user pool) = %t, want %t", got, tt.appendOtherPool)
			}
			if got := g.shouldLoadUserPoolChildResourceType(cognitoUserPoolClientResourceType, userPoolID); got != tt.loadClients {
				t.Fatalf("shouldLoadUserPoolChildResourceType(client) = %t, want %t", got, tt.loadClients)
			}
			if got := g.shouldLoadUserPoolChildResourceType(cognitoUserPoolClientResourceType, otherUserPoolID); got != tt.loadOtherClients {
				t.Fatalf("shouldLoadUserPoolChildResourceType(other client) = %t, want %t", got, tt.loadOtherClients)
			}
			if got := g.shouldAppendCognitoResource(cognitoUserPoolClientResourceType, client); got != tt.appendClient {
				t.Fatalf("shouldAppendCognitoResource(client) = %t, want %t", got, tt.appendClient)
			}
			if got := g.shouldAppendCognitoResource(cognitoUserGroupResourceType, group); got != tt.appendGroup {
				t.Fatalf("shouldAppendCognitoResource(group) = %t, want %t", got, tt.appendGroup)
			}
		})
	}
}

func TestCognitoUserPoolDomainMetadataGating(t *testing.T) {
	userPoolID := "us-east-1_abc"
	otherUserPoolID := "us-east-1_def"
	tests := []struct {
		name    string
		filters []terraformutils.ResourceFilter
		want    bool
	}{
		{name: "no filters loads domain metadata", want: true},
		{
			name: "typed user pool filter does not load domains",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: cognitoUserPoolResourceType, FieldPath: "id", AcceptableValues: []string{userPoolID}},
			},
		},
		{
			name: "typed domain filter loads domain metadata",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: cognitoUserPoolDomainResourceType, FieldPath: "id", AcceptableValues: []string{"auth.example.com"}},
			},
			want: true,
		},
		{
			name: "typed client filter does not load domains",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: cognitoUserPoolClientResourceType, FieldPath: "id", AcceptableValues: []string{cognitoUserPoolClientImportID(userPoolID, "client123")}},
			},
		},
		{
			name: "typed parent filter scopes typed domain discovery",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: cognitoUserPoolResourceType, FieldPath: "id", AcceptableValues: []string{otherUserPoolID}},
				{ServiceName: cognitoUserPoolDomainResourceType, FieldPath: "domain", AcceptableValues: []string{"auth.example.com"}},
			},
		},
		{
			name: "untyped id filter keeps domain discovery possible",
			filters: []terraformutils.ResourceFilter{
				{FieldPath: "id", AcceptableValues: []string{"auth.example.com"}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := CognitoGenerator{}
			g.Filter = tt.filters
			if got := g.shouldLoadUserPoolDomainMetadata(userPoolID); got != tt.want {
				t.Fatalf("shouldLoadUserPoolDomainMetadata() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestCognitoFilterGatesIdentityPoolsAndChildren(t *testing.T) {
	identityPoolID := "us-east-1:11111111-1111-1111-1111-111111111111"
	otherIdentityPoolID := "us-east-1:22222222-2222-2222-2222-222222222222"
	identityPool := newCognitoIdentityPoolResource(cognitoIdentityPoolRef{id: identityPoolID, name: "orders"})
	otherIdentityPool := newCognitoIdentityPoolResource(cognitoIdentityPoolRef{id: otherIdentityPoolID, name: "other"})
	roles := newCognitoIdentityPoolRolesAttachmentResource(cognitoIdentityPoolRef{id: identityPoolID, name: "orders"}, false)

	tests := []struct {
		name           string
		filters        []terraformutils.ResourceFilter
		loadPools      bool
		appendPool     bool
		appendOther    bool
		loadRoles      bool
		loadOtherRoles bool
		appendRoles    bool
	}{
		{
			name:           "no filters imports identity pools and role attachments",
			loadPools:      true,
			appendPool:     true,
			appendOther:    true,
			loadRoles:      true,
			loadOtherRoles: true,
			appendRoles:    true,
		},
		{
			name: "typed role attachment id filter skips parent output",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: cognitoIdentityPoolRolesAttachmentResourceType, FieldPath: "id", AcceptableValues: []string{identityPoolID}},
			},
			loadPools:   true,
			loadRoles:   true,
			appendRoles: true,
		},
		{
			name: "typed identity pool id filter scopes role loading",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: cognitoIdentityPoolResourceType, FieldPath: "id", AcceptableValues: []string{identityPoolID}},
			},
			loadPools:  true,
			appendPool: true,
			loadRoles:  true,
		},
		{
			name: "typed identity pool non-id filter avoids child pre-load",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: cognitoIdentityPoolResourceType, FieldPath: "tags.env", AcceptableValues: []string{"prod"}},
			},
			loadPools:   true,
			appendPool:  true,
			appendOther: true,
		},
		{
			name: "typed identity pool non-id filter does not block typed child non-id discovery",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: cognitoIdentityPoolResourceType, FieldPath: "tags.env", AcceptableValues: []string{"prod"}},
				{ServiceName: cognitoIdentityPoolRolesAttachmentResourceType, FieldPath: "roles.authenticated", AcceptableValues: []string{"arn:aws:iam::123456789012:role/auth"}},
			},
			loadPools:      true,
			appendPool:     true,
			appendOther:    true,
			loadRoles:      true,
			loadOtherRoles: true,
			appendRoles:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := CognitoGenerator{}
			g.Filter = tt.filters
			if got := g.shouldLoadIdentityPools(); got != tt.loadPools {
				t.Fatalf("shouldLoadIdentityPools() = %t, want %t", got, tt.loadPools)
			}
			if got := g.shouldAppendCognitoResource(cognitoIdentityPoolResourceType, identityPool); got != tt.appendPool {
				t.Fatalf("shouldAppendCognitoResource(identity pool) = %t, want %t", got, tt.appendPool)
			}
			if got := g.shouldAppendCognitoResource(cognitoIdentityPoolResourceType, otherIdentityPool); got != tt.appendOther {
				t.Fatalf("shouldAppendCognitoResource(other identity pool) = %t, want %t", got, tt.appendOther)
			}
			if got := g.shouldLoadIdentityPoolChildResourceType(cognitoIdentityPoolRolesAttachmentResourceType, identityPoolID); got != tt.loadRoles {
				t.Fatalf("shouldLoadIdentityPoolChildResourceType(roles) = %t, want %t", got, tt.loadRoles)
			}
			if got := g.shouldLoadIdentityPoolChildResourceType(cognitoIdentityPoolRolesAttachmentResourceType, otherIdentityPoolID); got != tt.loadOtherRoles {
				t.Fatalf("shouldLoadIdentityPoolChildResourceType(other roles) = %t, want %t", got, tt.loadOtherRoles)
			}
			if got := g.shouldAppendCognitoResource(cognitoIdentityPoolRolesAttachmentResourceType, roles); got != tt.appendRoles {
				t.Fatalf("shouldAppendCognitoResource(roles) = %t, want %t", got, tt.appendRoles)
			}
		})
	}
}

func TestCognitoUserPoolDomainResource(t *testing.T) {
	resource := newCognitoUserPoolDomainResource("us-east-1_abc", "auth.example.com")
	if resource.InstanceState.ID != "auth.example.com" {
		t.Fatalf("ID = %q, want auth.example.com", resource.InstanceState.ID)
	}
	if got := resource.InstanceState.Attributes["user_pool_id"]; got != "us-east-1_abc" {
		t.Fatalf("user_pool_id = %q, want us-east-1_abc", got)
	}
	if got := resource.InstanceState.Attributes["domain"]; got != "auth.example.com" {
		t.Fatalf("domain = %q, want auth.example.com", got)
	}
}

func TestCognitoUserPoolClientResourceUsesProviderReadID(t *testing.T) {
	resource := newCognitoUserPoolClientResource("us-east-1_abc", "client123", "web")
	if resource.InstanceState.ID != "client123" {
		t.Fatalf("ID = %q, want client123", resource.InstanceState.ID)
	}
	if got := resource.InstanceState.Attributes["user_pool_id"]; got != "us-east-1_abc" {
		t.Fatalf("user_pool_id = %q, want us-east-1_abc", got)
	}
	if resource.ResourceName == "" {
		t.Fatal("ResourceName is empty")
	}
}
