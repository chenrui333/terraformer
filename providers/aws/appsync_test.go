// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	appsynctypes "github.com/aws/aws-sdk-go-v2/service/appsync/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestAppSyncResourceIDs(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "api key", got: appSyncAPIKeyResourceID("api123", "key123"), want: "api123:key123"},
		{name: "data source", got: appSyncDataSourceResourceID("api123", "orders"), want: "api123-orders"},
		{name: "function", got: appSyncFunctionResourceID("api123", "func123"), want: "api123-func123"},
		{name: "resolver", got: appSyncResolverResourceID("api123", "Query", "getOrder"), want: "api123-Query-getOrder"},
		{name: "source api association", got: appSyncSourceAPIAssociationResourceID("api123", "assoc123"), want: "api123,assoc123"},
		{name: "SDL type", got: appSyncTypeResourceID("api123", "SDL", "Order"), want: "api123:SDL:Order"},
		{name: "JSON type", got: appSyncTypeResourceID("api123", "JSON", "Order"), want: "api123:JSON:Order"},
		{name: "resource name skips empty parts", got: appSyncResourceName("api123", "", "resolver", "Query"), want: "api123:resolver:Query"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestAppSyncResourceMissing(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "not found", err: &appsynctypes.NotFoundException{}, want: true},
		{name: "wrapped not found", err: errors.Join(errors.New("wrapper"), &appsynctypes.NotFoundException{}), want: true},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appSyncResourceMissing(tt.err); got != tt.want {
				t.Fatalf("appSyncResourceMissing(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func TestAppSyncFilterGatesGraphQLAPIAndChildDiscovery(t *testing.T) {
	apiID := "api123"
	otherAPIID := "api456"
	api := newAppSyncGraphQLAPIResource(apiID, "orders")
	otherAPI := newAppSyncGraphQLAPIResource(otherAPIID, "other")
	apiKey := newAppSyncAPIKeyResource(apiID, "key123")
	resolver := newAppSyncResolverResource(apiID, "Query", "getOrder")

	tests := []struct {
		name              string
		filters           []terraformutils.ResourceFilter
		loadAPIs          bool
		appendAPI         bool
		appendOtherAPI    bool
		loadChildren      bool
		loadOtherChildren bool
		loadAPIKeys       bool
		loadOtherAPIKeys  bool
		appendAPIKey      bool
		appendResolver    bool
	}{
		{
			name:              "no filters imports APIs and children",
			loadAPIs:          true,
			appendAPI:         true,
			appendOtherAPI:    true,
			loadChildren:      true,
			loadOtherChildren: true,
			loadAPIKeys:       true,
			loadOtherAPIKeys:  true,
			appendAPIKey:      true,
			appendResolver:    true,
		},
		{
			name: "typed API id filter limits APIs and children",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appSyncGraphQLAPIResourceType, FieldPath: "id", AcceptableValues: []string{apiID}},
			},
			loadAPIs:       true,
			appendAPI:      true,
			loadChildren:   true,
			loadAPIKeys:    true,
			appendAPIKey:   true,
			appendResolver: true,
		},
		{
			name: "typed child id filter does not import parent APIs",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appSyncAPIKeyResourceType, FieldPath: "id", AcceptableValues: []string{appSyncAPIKeyResourceID(apiID, "key123")}},
			},
			loadAPIs:     true,
			loadChildren: true,
			loadAPIKeys:  true,
			appendAPIKey: true,
		},
		{
			name: "typed parent and child id filters load matching child outside parent filter",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appSyncGraphQLAPIResourceType, FieldPath: "id", AcceptableValues: []string{otherAPIID}},
				{ServiceName: appSyncAPIKeyResourceType, FieldPath: "id", AcceptableValues: []string{appSyncAPIKeyResourceID(apiID, "key123")}},
			},
			loadAPIs:       true,
			appendOtherAPI: true,
			loadChildren:   true,
			loadAPIKeys:    true,
			appendAPIKey:   true,
		},
		{
			name: "typed parent id filter scopes typed child non-id discovery",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appSyncGraphQLAPIResourceType, FieldPath: "id", AcceptableValues: []string{apiID}},
				{ServiceName: appSyncAPIKeyResourceType, FieldPath: "description", AcceptableValues: []string{"orders"}},
			},
			loadAPIs:     true,
			appendAPI:    true,
			loadChildren: true,
			loadAPIKeys:  true,
			appendAPIKey: true,
		},
		{
			name: "typed domain id filter skips APIs and API children",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appSyncDomainNameResourceType, FieldPath: "id", AcceptableValues: []string{"api.example.com"}},
			},
		},
		{
			name: "global id filter constrains typed child discovery",
			filters: []terraformutils.ResourceFilter{
				{FieldPath: "id", AcceptableValues: []string{otherAPIID}},
				{ServiceName: appSyncAPIKeyResourceType, FieldPath: "id", AcceptableValues: []string{appSyncAPIKeyResourceID(apiID, "key123")}},
			},
			loadAPIs:       true,
			appendOtherAPI: true,
		},
		{
			name: "untyped child id filter reaches matching child without parent API",
			filters: []terraformutils.ResourceFilter{
				{FieldPath: "id", AcceptableValues: []string{appSyncAPIKeyResourceID(apiID, "key123")}},
			},
			loadAPIs:     true,
			loadChildren: true,
			loadAPIKeys:  true,
			appendAPIKey: true,
		},
		{
			name: "typed non-id API filter does not pre-load children",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appSyncGraphQLAPIResourceType, FieldPath: "tags.env", AcceptableValues: []string{"prod"}},
			},
			loadAPIs:       true,
			appendAPI:      true,
			appendOtherAPI: true,
			appendAPIKey:   true,
			appendResolver: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := AppSyncGenerator{}
			g.Filter = tt.filters
			if got := g.shouldLoadGraphQLAPIs(); got != tt.loadAPIs {
				t.Fatalf("shouldLoadGraphQLAPIs() = %t, want %t", got, tt.loadAPIs)
			}
			if got := g.shouldAppendGraphQLAPIResource(api); got != tt.appendAPI {
				t.Fatalf("shouldAppendGraphQLAPIResource(api) = %t, want %t", got, tt.appendAPI)
			}
			if got := g.shouldAppendGraphQLAPIResource(otherAPI); got != tt.appendOtherAPI {
				t.Fatalf("shouldAppendGraphQLAPIResource(otherAPI) = %t, want %t", got, tt.appendOtherAPI)
			}
			if got := g.shouldLoadGraphQLAPIChildren(api); got != tt.loadChildren {
				t.Fatalf("shouldLoadGraphQLAPIChildren(api) = %t, want %t", got, tt.loadChildren)
			}
			if got := g.shouldLoadGraphQLAPIChildren(otherAPI); got != tt.loadOtherChildren {
				t.Fatalf("shouldLoadGraphQLAPIChildren(otherAPI) = %t, want %t", got, tt.loadOtherChildren)
			}
			if got := g.shouldLoadAPIChildResourceType(appSyncAPIKeyResourceType, apiID); got != tt.loadAPIKeys {
				t.Fatalf("shouldLoadAPIChildResourceType(api key, api) = %t, want %t", got, tt.loadAPIKeys)
			}
			if got := g.shouldLoadAPIChildResourceType(appSyncAPIKeyResourceType, otherAPIID); got != tt.loadOtherAPIKeys {
				t.Fatalf("shouldLoadAPIChildResourceType(api key, other api) = %t, want %t", got, tt.loadOtherAPIKeys)
			}
			if got := g.shouldAppendAppSyncChildResource(appSyncAPIKeyResourceType, apiKey); got != tt.appendAPIKey {
				t.Fatalf("shouldAppendAppSyncChildResource(api key) = %t, want %t", got, tt.appendAPIKey)
			}
			if got := g.shouldAppendAppSyncChildResource(appSyncResolverResourceType, resolver); got != tt.appendResolver {
				t.Fatalf("shouldAppendAppSyncChildResource(resolver) = %t, want %t", got, tt.appendResolver)
			}
		})
	}
}

func TestAppSyncFilterGatesDomainNamesAndAssociations(t *testing.T) {
	domainName := "api.example.com"
	otherDomainName := "other.example.com"
	domain := newAppSyncDomainNameResource(domainName)
	otherDomain := newAppSyncDomainNameResource(otherDomainName)
	association := newAppSyncDomainNameAPIAssociationResource(domainName, "api123")

	tests := []struct {
		name                 string
		filters              []terraformutils.ResourceFilter
		loadDomains          bool
		requireDomains       bool
		appendDomain         bool
		appendOther          bool
		loadAssociation      bool
		loadOtherAssociation bool
		appendAssociation    bool
	}{
		{
			name:                 "no filters imports domain names and associations",
			loadDomains:          true,
			appendDomain:         true,
			appendOther:          true,
			loadAssociation:      true,
			loadOtherAssociation: true,
			appendAssociation:    true,
		},
		{
			name: "typed GraphQL API filter skips domain scan",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appSyncGraphQLAPIResourceType, FieldPath: "id", AcceptableValues: []string{"api123"}},
			},
		},
		{
			name: "untyped domain id filter keeps domain scan with typed API filter",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appSyncGraphQLAPIResourceType, FieldPath: "id", AcceptableValues: []string{"api123"}},
				{FieldPath: "id", AcceptableValues: []string{domainName}},
			},
			loadDomains:       true,
			appendDomain:      true,
			loadAssociation:   true,
			appendAssociation: true,
		},
		{
			name: "typed API child filter skips domain scan",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appSyncAPIKeyResourceType, FieldPath: "id", AcceptableValues: []string{"api123:key123"}},
			},
		},
		{
			name: "typed domain id filter loads matching domain and association",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appSyncDomainNameResourceType, FieldPath: "id", AcceptableValues: []string{domainName}},
			},
			loadDomains:       true,
			requireDomains:    true,
			appendDomain:      true,
			loadAssociation:   true,
			appendAssociation: true,
		},
		{
			name: "typed domain id filter scopes typed association non-id discovery",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appSyncDomainNameResourceType, FieldPath: "id", AcceptableValues: []string{domainName}},
				{ServiceName: appSyncDomainNameAPIAssociationResourceType, FieldPath: "api_id", AcceptableValues: []string{"api123"}},
			},
			loadDomains:       true,
			requireDomains:    true,
			appendDomain:      true,
			loadAssociation:   true,
			appendAssociation: true,
		},
		{
			name: "typed domain non-id filter avoids association pre-load",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appSyncDomainNameResourceType, FieldPath: "tags.env", AcceptableValues: []string{"prod"}},
			},
			loadDomains:       true,
			requireDomains:    true,
			appendDomain:      true,
			appendOther:       true,
			appendAssociation: true,
		},
		{
			name: "typed association filter skips parent domain resource",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appSyncDomainNameAPIAssociationResourceType, FieldPath: "id", AcceptableValues: []string{domainName}},
			},
			loadDomains:       true,
			requireDomains:    true,
			loadAssociation:   true,
			appendAssociation: true,
		},
		{
			name: "global id filter constrains typed association",
			filters: []terraformutils.ResourceFilter{
				{FieldPath: "id", AcceptableValues: []string{otherDomainName}},
				{ServiceName: appSyncDomainNameAPIAssociationResourceType, FieldPath: "id", AcceptableValues: []string{domainName}},
			},
			loadDomains:    true,
			requireDomains: true,
			appendOther:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := AppSyncGenerator{}
			g.Filter = tt.filters
			if got := g.shouldLoadDomainNames(); got != tt.loadDomains {
				t.Fatalf("shouldLoadDomainNames() = %t, want %t", got, tt.loadDomains)
			}
			if got := g.shouldRequireDomainNameLoad(); got != tt.requireDomains {
				t.Fatalf("shouldRequireDomainNameLoad() = %t, want %t", got, tt.requireDomains)
			}
			if got := g.shouldAppendDomainNameResource(domain); got != tt.appendDomain {
				t.Fatalf("shouldAppendDomainNameResource(domain) = %t, want %t", got, tt.appendDomain)
			}
			if got := g.shouldAppendDomainNameResource(otherDomain); got != tt.appendOther {
				t.Fatalf("shouldAppendDomainNameResource(otherDomain) = %t, want %t", got, tt.appendOther)
			}
			if got := g.shouldLoadDomainNameAPIAssociation(domainName); got != tt.loadAssociation {
				t.Fatalf("shouldLoadDomainNameAPIAssociation() = %t, want %t", got, tt.loadAssociation)
			}
			if got := g.shouldLoadDomainNameAPIAssociation(otherDomainName); got != tt.loadOtherAssociation {
				t.Fatalf("shouldLoadDomainNameAPIAssociation(other) = %t, want %t", got, tt.loadOtherAssociation)
			}
			if got := g.shouldAppendAppSyncChildResource(appSyncDomainNameAPIAssociationResourceType, association); got != tt.appendAssociation {
				t.Fatalf("shouldAppendAppSyncChildResource(association) = %t, want %t", got, tt.appendAssociation)
			}
		})
	}
}
