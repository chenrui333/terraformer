// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	oktasdk "github.com/okta/okta-sdk-golang/v6/okta"
)

func TestOktaProviderRegistration(t *testing.T) {
	provider := &OktaProvider{}
	t.Setenv("OKTA_ORG_NAME", "dev-123")
	t.Setenv("OKTA_BASE_URL", "okta.com")
	t.Setenv("OKTA_API_TOKEN", "api-token")

	if err := provider.Init(nil); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if got := provider.GetName(); got != "okta" {
		t.Fatalf("GetName() = %q, want okta", got)
	}
	if provider.GetConfig().IsNull() {
		t.Fatal("GetConfig() returned null config")
	}

	wantServices := map[string]reflect.Type{
		"okta_idp_oidc":        reflect.TypeOf(&IdpOIDCGenerator{}),
		"okta_idp_saml":        reflect.TypeOf(&IdpSAMLGenerator{}),
		"okta_idp_social":      reflect.TypeOf(&IdpSocialGenerator{}),
		"okta_policy_password": reflect.TypeOf(&PasswordPolicyGenerator{}),
		"okta_policy_mfa":      reflect.TypeOf(&MFAPolicyGenerator{}),
		"okta_policy_signon":   reflect.TypeOf(&SignOnPolicyGenerator{}),
	}
	services := provider.GetSupportedService()
	for name, wantType := range wantServices {
		service, ok := services[name]
		if !ok {
			t.Fatalf("supported services missing %q", name)
		}
		if gotType := reflect.TypeOf(service); gotType != wantType {
			t.Fatalf("service %q type = %v, want %v", name, gotType, wantType)
		}
	}

	if err := provider.InitService("okta_idp_oidc", true); err != nil {
		t.Fatalf("InitService(okta_idp_oidc) returned error: %v", err)
	}
	if got := provider.GetService().GetArgs()["api_token"]; got != "api-token" {
		t.Fatalf("selected service api token arg = %#v, want api-token", got)
	}
	if err := provider.InitService("missing", false); err == nil {
		t.Fatal("InitService(missing) returned nil error")
	}
}

func TestOktaIDPCreateResources(t *testing.T) {
	tests := []struct {
		name     string
		resource terraformutils.Resource
		wantName string
		wantType string
	}{
		{
			name:     "oidc",
			resource: IdpOIDCGenerator{}.createResources([]oktasdk.IdentityProvider{testIdentityProvider("idp-oidc", "OIDC", "Example IDP")})[0],
			wantName: "idp_" + normalizeResourceName("OIDC_Example IDP"),
			wantType: "okta_idp_oidc",
		},
		{
			name:     "saml",
			resource: IdpSAMLGenerator{}.createResources([]oktasdk.IdentityProvider{testIdentityProvider("idp-saml", "SAML2", "Example IDP")})[0],
			wantName: "idp_" + normalizeResourceName("SAML2_Example IDP"),
			wantType: "okta_idp_saml",
		},
		{
			name:     "social",
			resource: IdpSocialGenerator{}.createResources([]oktasdk.IdentityProvider{testIdentityProvider("idp-social", "SOCIAL", "Example IDP")})[0],
			wantName: "idp_" + normalizeResourceName("SOCIAL_Example IDP"),
			wantType: "okta_idp_social",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertOktaResource(t, tt.resource, tt.resource.InstanceState.ID, tt.wantName, tt.wantType, map[string]string{})
		})
	}
}

func TestOktaPolicyCreateResources(t *testing.T) {
	passwordDefault := newTestPasswordPolicy("pol-password-default", "Default Policy")
	passwordCustom := newTestPasswordPolicy("pol-password-custom", "Custom Password")
	mfaDefault := newTestAuthenticatorEnrollmentPolicy("pol-mfa-default", "Default Policy")
	signOn := newTestSignOnPolicy("pol-signon", "Sign On")

	tests := []struct {
		name      string
		resources []terraformutils.Resource
		wantIDs   []string
		wantNames []string
		wantTypes []string
	}{
		{
			name: "password policies",
			resources: PasswordPolicyGenerator{}.createResources([]oktasdk.ListPolicies200ResponseInner{
				oktasdk.PasswordPolicyAsListPolicies200ResponseInner(passwordDefault),
				oktasdk.PasswordPolicyAsListPolicies200ResponseInner(passwordCustom),
			}),
			wantIDs:   []string{"pol-password-default", "pol-password-custom"},
			wantNames: []string{"policy_password_" + normalizeResourceName("Default Policy"), "policy_password_" + normalizeResourceName("Custom Password")},
			wantTypes: []string{"okta_policy_password_default", "okta_policy_password"},
		},
		{
			name: "mfa default policy",
			resources: MFAPolicyGenerator{}.createResources([]oktasdk.ListPolicies200ResponseInner{
				oktasdk.AuthenticatorEnrollmentPolicyAsListPolicies200ResponseInner(mfaDefault),
			}),
			wantIDs:   []string{"pol-mfa-default"},
			wantNames: []string{"policy_mfa_" + normalizeResourceName("Default Policy")},
			wantTypes: []string{"okta_policy_mfa_default"},
		},
		{
			name: "signon policy",
			resources: SignOnPolicyGenerator{}.createResources([]oktasdk.ListPolicies200ResponseInner{
				oktasdk.OktaSignOnPolicyAsListPolicies200ResponseInner(signOn),
			}),
			wantIDs:   []string{"pol-signon"},
			wantNames: []string{"policy_signon_" + normalizeResourceName("Sign On")},
			wantTypes: []string{"okta_policy_signon"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.resources) != len(tt.wantIDs) {
				t.Fatalf("resources length = %d, want %d", len(tt.resources), len(tt.wantIDs))
			}
			for i, resource := range tt.resources {
				assertOktaResource(t, resource, tt.wantIDs[i], tt.wantNames[i], tt.wantTypes[i], map[string]string{})
			}
		})
	}
}

func TestGetAuthorizationServerPoliciesPaginates(t *testing.T) {
	var requests []string
	client := newTestOktaClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.String())
		if r.URL.Path != "/api/v1/authorizationServers/authz-1/policies" {
			t.Errorf("request path = %q, want /api/v1/authorizationServers/authz-1/policies", r.URL.Path)
			writeOktaError(t, w, http.StatusNotFound, "unexpected path")
			return
		}

		switch r.URL.Query().Get("after") {
		case "":
			w.Header().Set("Link", fmt.Sprintf("<%s/api/v1/authorizationServers/authz-1/policies?after=second>; rel=%q", testOktaRequestBaseURL(r), "next"))
			writeOktaJSON(t, w, []map[string]string{{"id": "policy-1", "type": "RESOURCE_ACCESS", "name": "First Policy"}})
		case "second":
			writeOktaJSON(t, w, []map[string]string{{"id": "policy-2", "type": "RESOURCE_ACCESS", "name": "Second Policy"}})
		default:
			t.Errorf("after query = %q, want empty or second", r.URL.Query().Get("after"))
			writeOktaError(t, w, http.StatusBadRequest, "unexpected page")
		}
	})

	policies, err := getAuthorizationServerPolicies(context.Background(), client, "authz-1")
	if err != nil {
		t.Fatalf("getAuthorizationServerPolicies() returned error: %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("getAuthorizationServerPolicies() returned %d policies, want 2; requests=%v", len(policies), requests)
	}

	resources := AuthorizationServerPolicyGenerator{}.createResources(policies, "authz-1", "Authorization Server")
	if len(resources) != 2 {
		t.Fatalf("resources length = %d, want 2", len(resources))
	}
	assertOktaResource(t, resources[0], "policy-1", normalizeResourceName("auth_server_Authorization Server_policy_First Policy"), "okta_auth_server_policy", map[string]string{"auth_server_id": "authz-1"})
	assertOktaResource(t, resources[1], "policy-2", normalizeResourceName("auth_server_Authorization Server_policy_Second Policy"), "okta_auth_server_policy", map[string]string{"auth_server_id": "authz-1"})
}

func TestGetIdpOIDCPaginates(t *testing.T) {
	var requests []string
	client := newTestOktaClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.String())
		if r.URL.Path != "/api/v1/idps" {
			t.Errorf("request path = %q, want /api/v1/idps", r.URL.Path)
			writeOktaError(t, w, http.StatusNotFound, "unexpected path")
			return
		}
		if got := r.URL.Query().Get("type"); got != "OIDC" {
			t.Errorf("type query = %q, want OIDC", got)
		}
		if got := r.URL.Query().Get("limit"); got != "1" {
			t.Errorf("limit query = %q, want 1", got)
		}

		switch r.URL.Query().Get("after") {
		case "":
			w.Header().Set("Link", fmt.Sprintf("<%s/api/v1/idps?after=second>; rel=\"next\"", testOktaRequestBaseURL(r)))
			writeOktaJSON(t, w, []map[string]string{{"id": "idp-1", "type": "OIDC", "name": "First"}})
		case "second":
			writeOktaJSON(t, w, []map[string]string{{"id": "idp-2", "type": "OIDC", "name": "Second"}})
		default:
			t.Errorf("after query = %q, want empty or second", r.URL.Query().Get("after"))
			writeOktaError(t, w, http.StatusBadRequest, "unexpected page")
		}
	})

	idps, err := getIdpOIDC(context.Background(), client)
	if err != nil {
		t.Fatalf("getIdpOIDC() returned error: %v", err)
	}
	if len(idps) != 2 {
		t.Fatalf("getIdpOIDC() returned %d IDPs, want 2; requests=%v", len(idps), requests)
	}
	resources := IdpOIDCGenerator{}.createResources(idps)
	if len(resources) != 2 {
		t.Fatalf("resources length = %d, want 2", len(resources))
	}
	assertOktaResource(t, resources[0], "idp-1", "idp_"+normalizeResourceName("OIDC_First"), "okta_idp_oidc", map[string]string{})
	assertOktaResource(t, resources[1], "idp-2", "idp_"+normalizeResourceName("OIDC_Second"), "okta_idp_oidc", map[string]string{})
}

func TestGetIdpOIDCEmptyResponse(t *testing.T) {
	client := newTestOktaClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/idps" {
			t.Errorf("request path = %q, want /api/v1/idps", r.URL.Path)
			writeOktaError(t, w, http.StatusNotFound, "unexpected path")
			return
		}
		writeOktaJSON(t, w, []map[string]string{})
	})

	idps, err := getIdpOIDC(context.Background(), client)
	if err != nil {
		t.Fatalf("getIdpOIDC() returned error: %v", err)
	}
	if len(idps) != 0 {
		t.Fatalf("getIdpOIDC() returned %d IDPs, want 0", len(idps))
	}
}

func TestGetIdpOIDCPropagatesNextPageError(t *testing.T) {
	sawSecondPage := false
	client := newTestOktaClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("after") {
		case "":
			w.Header().Set("Link", fmt.Sprintf("<%s/api/v1/idps?after=second>; rel=\"next\"", testOktaRequestBaseURL(r)))
			writeOktaJSON(t, w, []map[string]string{{"id": "idp-1", "type": "OIDC", "name": "First"}})
		case "second":
			sawSecondPage = true
			writeOktaError(t, w, http.StatusInternalServerError, "second page failed")
		default:
			t.Errorf("after query = %q, want empty or second", r.URL.Query().Get("after"))
			writeOktaError(t, w, http.StatusBadRequest, "unexpected page")
		}
	})

	_, err := getIdpOIDC(context.Background(), client)
	if err == nil {
		t.Fatal("getIdpOIDC() returned nil error")
	}
	if !sawSecondPage {
		t.Fatal("getIdpOIDC() did not request the second page")
	}
	if !strings.Contains(err.Error(), "500") && !strings.Contains(err.Error(), "unmarshal") {
		t.Fatalf("getIdpOIDC() error = %q, want propagated second-page failure", err)
	}
}

func newTestOktaClient(t *testing.T, handler http.HandlerFunc) *oktasdk.APIClient {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}

	config, err := oktasdk.NewConfiguration(
		oktasdk.WithOrgUrl("https://example.okta.com"),
		oktasdk.WithToken("test-token"),
	)
	if err != nil {
		t.Fatalf("okta.NewConfiguration() returned error: %v", err)
	}
	config.Servers = oktasdk.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()
	config.Host = serverURL.Host
	config.Scheme = serverURL.Scheme
	config.Okta.Client.OrgUrl = server.URL
	config.Okta.Client.AuthorizationMode = "SSWS"
	config.Okta.Client.Token = "test-token"
	config.Okta.Client.Proxy.Host = ""
	config.Okta.Client.Proxy.Port = 0
	config.Okta.Client.Proxy.Username = ""
	config.Okta.Client.Proxy.Password = ""
	return oktasdk.NewAPIClient(config)
}

func writeOktaJSON(t *testing.T, w http.ResponseWriter, value interface{}) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Errorf("write JSON response: %v", err)
	}
}

func writeOktaError(t *testing.T, w http.ResponseWriter, status int, message string) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"errorSummary": message,
	}); err != nil {
		t.Errorf("write error response: %v", err)
	}
}

func testOktaRequestBaseURL(r *http.Request) string {
	return "http://" + r.Host
}

func testIdentityProvider(id, idpType, name string) oktasdk.IdentityProvider {
	idp := oktasdk.IdentityProvider{}
	idp.SetId(id)
	idp.SetType(idpType)
	idp.SetName(name)
	return idp
}

func newTestPasswordPolicy(id, name string) *oktasdk.PasswordPolicy {
	policy := oktasdk.NewPasswordPolicy(name, "PASSWORD")
	policy.Id = stringPtr(id)
	return policy
}

func newTestAuthenticatorEnrollmentPolicy(id, name string) *oktasdk.AuthenticatorEnrollmentPolicy {
	policy := oktasdk.NewAuthenticatorEnrollmentPolicy(name, "MFA_ENROLL")
	policy.Id = stringPtr(id)
	return policy
}

func newTestSignOnPolicy(id, name string) *oktasdk.OktaSignOnPolicy {
	policy := oktasdk.NewOktaSignOnPolicy(name, "OKTA_SIGN_ON")
	policy.Id = stringPtr(id)
	return policy
}

func assertOktaResource(t *testing.T, resource terraformutils.Resource, wantID, wantName, wantType string, wantAttrs map[string]string) {
	t.Helper()

	if resource.InstanceState == nil {
		t.Fatal("resource InstanceState is nil")
	}
	if resource.InstanceInfo == nil {
		t.Fatal("resource InstanceInfo is nil")
	}
	if got := resource.InstanceState.ID; got != wantID {
		t.Fatalf("resource ID = %q, want %q", got, wantID)
	}
	wantResourceName := terraformutils.TfSanitize(wantName)
	if got := resource.ResourceName; got != wantResourceName {
		t.Fatalf("resource name = %q, want %q", got, wantResourceName)
	}
	if got := resource.InstanceInfo.Type; got != wantType {
		t.Fatalf("resource type = %q, want %q", got, wantType)
	}
	if !reflect.DeepEqual(resource.InstanceState.Attributes, wantAttrs) {
		t.Fatalf("resource attrs = %#v, want %#v", resource.InstanceState.Attributes, wantAttrs)
	}
	if got := resource.Provider; got != "okta" {
		t.Fatalf("resource provider = %q, want okta", got)
	}
}
