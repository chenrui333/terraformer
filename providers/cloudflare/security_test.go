// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	cf "github.com/chenrui333/terraformer/providers/cloudflare/internal/cloudflarev7"
)

func TestCloudflareZoneSecurityResourceUsesCompositeImportID(t *testing.T) {
	resource, ok := cloudflareZoneSecurityResource(
		cf.Zone{ID: "zone-123", Name: "example.com"},
		"operation-456",
		"cloudflare_api_shield_operation",
		"api_shield_operation",
		"api.example.com",
		"GET",
	)
	if !ok {
		t.Fatal("expected zone security resource")
	}
	if resource.InstanceInfo.Type != "cloudflare_api_shield_operation" {
		t.Fatalf("resource type = %q, want cloudflare_api_shield_operation", resource.InstanceInfo.Type)
	}
	if resource.InstanceState.ID != "operation-456" {
		t.Fatalf("resource ID = %q, want operation-456", resource.InstanceState.ID)
	}
	if got := resource.InstanceState.Attributes["zone_id"]; got != "zone-123" {
		t.Fatalf("zone_id = %q, want zone-123", got)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "zone-123/operation-456" {
		t.Fatalf("import_id = %q, want zone-123/operation-456", got)
	}
}

func TestCloudflareZoneSecuritySingletonResourceUsesZoneImportID(t *testing.T) {
	resource, ok := cloudflareZoneSecuritySingletonResource(
		cf.Zone{ID: "zone-123", Name: "example.com"},
		"cloudflare_api_shield",
		"api_shield",
	)
	if !ok {
		t.Fatal("expected zone security singleton resource")
	}
	if resource.InstanceState.ID != "zone-123" {
		t.Fatalf("resource ID = %q, want zone-123", resource.InstanceState.ID)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "zone-123" {
		t.Fatalf("import_id = %q, want zone-123", got)
	}
}

func TestCloudflareAccountSecurityResourceUsesCompositeImportID(t *testing.T) {
	resource, ok := cloudflareAccountSecurityResource(
		"account-123",
		"target-456",
		"cloudflare_vulnerability_scanner_target_environment",
		"vulnerability_scanner_target_environment",
		"production",
	)
	if !ok {
		t.Fatal("expected account security resource")
	}
	if resource.InstanceInfo.Type != "cloudflare_vulnerability_scanner_target_environment" {
		t.Fatalf("resource type = %q, want cloudflare_vulnerability_scanner_target_environment", resource.InstanceInfo.Type)
	}
	if got := resource.InstanceState.Attributes["account_id"]; got != "account-123" {
		t.Fatalf("account_id = %q, want account-123", got)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "account-123/target-456" {
		t.Fatalf("import_id = %q, want account-123/target-456", got)
	}
}

func TestCloudflareSecurityIDStringHandlesStringAndNumericIDs(t *testing.T) {
	tests := []struct {
		name     string
		resource cloudflareSecurityRawResource
		keys     []string
		want     string
	}{
		{
			name:     "string ID",
			resource: cloudflareSecurityRawResource{"id": "rule-123"},
			keys:     []string{"id"},
			want:     "rule-123",
		},
		{
			name:     "numeric ID",
			resource: cloudflareSecurityRawResource{"id": 12345.0},
			keys:     []string{"id"},
			want:     "12345",
		},
		{
			name:     "fallback key",
			resource: cloudflareSecurityRawResource{"name": "asset-123"},
			keys:     []string{"id", "name"},
			want:     "asset-123",
		},
		{
			name:     "fractional numeric ID skipped",
			resource: cloudflareSecurityRawResource{"id": 1.5},
			keys:     []string{"id"},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cloudflareSecurityIDString(tt.resource, tt.keys...); got != tt.want {
				t.Fatalf("cloudflareSecurityIDString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCloudflareScopedSecurityResourceUsesScopedImportID(t *testing.T) {
	resource, ok := cloudflareScopedSecurityResource(
		"zones",
		"zone-123",
		"waf_block",
		"cloudflare_custom_pages",
		"custom_page",
		"example.com",
	)
	if !ok {
		t.Fatal("expected scoped security resource")
	}
	if got := resource.InstanceState.Attributes["zone_id"]; got != "zone-123" {
		t.Fatalf("zone_id = %q, want zone-123", got)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "zones/zone-123/waf_block" {
		t.Fatalf("import_id = %q, want zones/zone-123/waf_block", got)
	}

	accountResource, ok := cloudflareScopedSecurityResource(
		"accounts",
		"account-123",
		"asset-123",
		"cloudflare_custom_page_asset",
		"custom_page_asset",
	)
	if !ok {
		t.Fatal("expected account-scoped security resource")
	}
	if got := accountResource.InstanceState.Attributes["account_id"]; got != "account-123" {
		t.Fatalf("account_id = %q, want account-123", got)
	}
	if got := accountResource.InstanceState.Meta["import_id"]; got != "accounts/account-123/asset-123" {
		t.Fatalf("import_id = %q, want accounts/account-123/asset-123", got)
	}
}

func TestCloudflareSecurityResourceConstructorsSkipMissingIDs(t *testing.T) {
	if _, ok := cloudflareZoneSecurityResource(cf.Zone{ID: "zone-123"}, "", "cloudflare_api_shield_operation", "api_shield_operation"); ok {
		t.Fatal("expected zone resource without resource ID to be skipped")
	}
	if _, ok := cloudflareZoneSecurityResource(cf.Zone{}, "operation-123", "cloudflare_api_shield_operation", "api_shield_operation"); ok {
		t.Fatal("expected zone resource without zone ID to be skipped")
	}
	if _, ok := cloudflareAccountSecurityResource("", "credential-set-123", "cloudflare_vulnerability_scanner_credential_set", "credential_set"); ok {
		t.Fatal("expected account resource without account ID to be skipped")
	}
	if _, ok := cloudflareAccountSecurityResource("account-123", "", "cloudflare_vulnerability_scanner_credential_set", "credential_set"); ok {
		t.Fatal("expected account resource without resource ID to be skipped")
	}
}

func TestListCloudflareSecurityResourcesPaginates(t *testing.T) {
	api := newCloudflareSecurityTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/zones/zone-123/api_gateway/operations" {
			t.Fatalf("path = %q, want /zones/zone-123/api_gateway/operations", r.URL.Path)
		}
		switch r.URL.Query().Get("page") {
		case "1":
			writeCloudflareSecurityTestResponse(t, w, []map[string]string{{"operation_id": "operation-1"}}, map[string]int{
				"page":        1,
				"per_page":    cloudflarePageSize,
				"total_pages": 2,
			})
		case "2":
			writeCloudflareSecurityTestResponse(t, w, []map[string]string{{"operation_id": "operation-2"}}, map[string]int{
				"page":        2,
				"per_page":    cloudflarePageSize,
				"total_pages": 2,
			})
		default:
			t.Fatalf("page query = %q, want 1 or 2", r.URL.Query().Get("page"))
		}
	}))

	resources, err := listCloudflareSecurityResources(context.Background(), api, "/zones/zone-123/api_gateway/operations")
	if err != nil {
		t.Fatalf("listCloudflareSecurityResources() error = %v", err)
	}
	got := []string{
		cloudflareSecurityString(resources[0], "operation_id"),
		cloudflareSecurityString(resources[1], "operation_id"),
	}
	if want := []string{"operation-1", "operation-2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("operation IDs = %#v, want %#v", got, want)
	}
}

func TestListCloudflareSecurityResourcesHandlesEmptyResponse(t *testing.T) {
	api := newCloudflareSecurityTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeCloudflareSecurityTestResponse(t, w, []map[string]string{}, map[string]int{
			"page":        1,
			"per_page":    cloudflarePageSize,
			"total_pages": 1,
		})
	}))

	resources, err := listCloudflareSecurityResources(context.Background(), api, "/zones/zone-123/page_shield/policies")
	if err != nil {
		t.Fatalf("listCloudflareSecurityResources() error = %v", err)
	}
	if len(resources) != 0 {
		t.Fatalf("resources = %#v, want empty", resources)
	}
}

func TestAPIShieldConfigImportable(t *testing.T) {
	tests := []struct {
		name   string
		config cloudflareSecurityRawResource
		want   bool
	}{
		{name: "configured", config: cloudflareSecurityRawResource{"auth_id_characteristics": []interface{}{map[string]interface{}{"name": "authorization", "type": "header"}}}, want: true},
		{name: "default empty", config: cloudflareSecurityRawResource{"auth_id_characteristics": []interface{}{}}, want: false},
		{name: "missing characteristics", config: cloudflareSecurityRawResource{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apiShieldConfigImportable(tt.config); got != tt.want {
				t.Fatalf("apiShieldConfigImportable() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestAppendAPIShieldResourceSkipsDefaultConfig(t *testing.T) {
	api := newCloudflareSecurityTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/zones/zone-123/api_gateway/configuration" {
			t.Fatalf("path = %q, want /zones/zone-123/api_gateway/configuration", r.URL.Path)
		}
		writeCloudflareSecurityTestResponse(t, w, map[string]interface{}{
			"auth_id_characteristics": []interface{}{},
		}, nil)
	}))
	g := &SecurityGenerator{}

	if err := g.appendAPIShieldResource(context.Background(), api, cf.Zone{ID: "zone-123", Name: "example.com"}); err != nil {
		t.Fatalf("appendAPIShieldResource() error = %v", err)
	}
	if len(g.Resources) != 0 {
		t.Fatalf("resources = %#v, want empty", g.Resources)
	}
}

func TestAppendAPIShieldResourceCreatesConfiguredSingleton(t *testing.T) {
	api := newCloudflareSecurityTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeCloudflareSecurityTestResponse(t, w, map[string]interface{}{
			"auth_id_characteristics": []map[string]string{{"name": "authorization", "type": "header"}},
		}, nil)
	}))
	g := &SecurityGenerator{}

	if err := g.appendAPIShieldResource(context.Background(), api, cf.Zone{ID: "zone-123", Name: "example.com"}); err != nil {
		t.Fatalf("appendAPIShieldResource() error = %v", err)
	}
	if len(g.Resources) != 1 {
		t.Fatalf("resource count = %d, want 1", len(g.Resources))
	}
	resource := g.Resources[0]
	if resource.InstanceInfo.Type != "cloudflare_api_shield" {
		t.Fatalf("resource type = %q, want cloudflare_api_shield", resource.InstanceInfo.Type)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "zone-123" {
		t.Fatalf("import_id = %q, want zone-123", got)
	}
}

func TestAppendAPIShieldOperationResources(t *testing.T) {
	api := newCloudflareSecurityTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeCloudflareSecurityTestResponse(t, w, []map[string]string{
			{
				"endpoint":     "/api/users/{var1}",
				"host":         "api.example.com",
				"method":       "GET",
				"operation_id": "operation-123",
			},
		}, nil)
	}))
	g := &SecurityGenerator{}

	if err := g.appendAPIShieldOperationResources(context.Background(), api, cf.Zone{ID: "zone-123", Name: "example.com"}); err != nil {
		t.Fatalf("appendAPIShieldOperationResources() error = %v", err)
	}
	if len(g.Resources) != 1 {
		t.Fatalf("resource count = %d, want 1", len(g.Resources))
	}
	resource := g.Resources[0]
	if resource.InstanceInfo.Type != "cloudflare_api_shield_operation" {
		t.Fatalf("resource type = %q, want cloudflare_api_shield_operation", resource.InstanceInfo.Type)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "zone-123/operation-123" {
		t.Fatalf("import_id = %q, want zone-123/operation-123", got)
	}
}

func TestAppendSecurityZoneAdjunctResources(t *testing.T) {
	responses := map[string]interface{}{
		"/zones/zone-123/cloud_connector/rules": []map[string]string{
			{"id": "connector-rule-123"},
		},
		"/zones/zone-123/custom_pages": []map[string]string{
			{"identifier": "waf_block", "state": "customized", "description": "WAF block"},
			{"identifier": "basic_challenge", "state": "default", "description": "Default challenge"},
		},
		"/zones/zone-123/custom_pages/assets": []map[string]string{
			{"name": "error_asset", "description": "Error page"},
		},
		"/zones/zone-123/leaked-credential-checks/detections": []map[string]string{
			{"id": "detection-123", "username": "lookup_json_string(http.request.body.raw, \"user\")", "password": "lookup_json_string(http.request.body.raw, \"password\")"},
		},
		"/zones/zone-123/token_validation/config": []map[string]string{
			{"id": "config-123", "title": "JWT config"},
		},
		"/zones/zone-123/token_validation/rules": []map[string]string{
			{"id": "rule-123", "title": "JWT rule"},
		},
		"/zones/zone-123/firewall/ua_rules": []map[string]string{
			{"id": "ua-123", "description": "Legacy client"},
		},
	}
	api := newCloudflareSecurityTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result, ok := responses[r.URL.Path]
		if !ok {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		writeCloudflareSecurityTestResponse(t, w, result, nil)
	}))
	g := &SecurityGenerator{}
	zone := cf.Zone{ID: "zone-123", Name: "example.com"}

	for _, appendResources := range []func(context.Context, *cf.API, cf.Zone) error{
		g.appendCloudConnectorRulesResource,
		func(ctx context.Context, api *cf.API, zone cf.Zone) error {
			return g.appendCustomPageResources(ctx, api, "zones", zone.ID, zone.Name)
		},
		func(ctx context.Context, api *cf.API, zone cf.Zone) error {
			return g.appendCustomPageAssetResources(ctx, api, "zones", zone.ID, zone.Name)
		},
		g.appendLeakedCredentialCheckRuleResources,
		g.appendTokenValidationConfigResources,
		g.appendTokenValidationRuleResources,
		g.appendUserAgentBlockingRuleResources,
	} {
		if err := appendResources(context.Background(), api, zone); err != nil {
			t.Fatalf("appendResources() error = %v", err)
		}
	}

	got := map[string]string{}
	for _, resource := range g.Resources {
		got[resource.InstanceInfo.Type] = resource.InstanceState.Meta["import_id"].(string)
	}
	want := map[string]string{
		"cloudflare_cloud_connector_rules":        "zone-123",
		"cloudflare_custom_pages":                 "zones/zone-123/waf_block",
		"cloudflare_custom_page_asset":            "zones/zone-123/error_asset",
		"cloudflare_leaked_credential_check_rule": "zone-123/detection-123",
		"cloudflare_token_validation_config":      "zone-123/config-123",
		"cloudflare_token_validation_rules":       "zone-123/rule-123",
		"cloudflare_user_agent_blocking_rule":     "zone-123/ua-123",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resources = %#v, want %#v", got, want)
	}
}

func TestAppendSecurityEmailResources(t *testing.T) {
	responses := map[string]interface{}{
		"/accounts/account-123/email-security/settings/block_senders": []map[string]interface{}{
			{"id": 101.0, "pattern": "bad.example.com", "pattern_type": "domain"},
		},
		"/accounts/account-123/email-security/settings/impersonation_registry": []map[string]interface{}{
			{"id": 202.0, "name": "Finance", "email": "finance@example.com"},
		},
	}
	api := newCloudflareSecurityTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result, ok := responses[r.URL.Path]
		if !ok {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		writeCloudflareSecurityTestResponse(t, w, result, nil)
	}))
	g := &SecurityGenerator{}

	for _, appendResources := range []func(context.Context, *cf.API, string) error{
		g.appendEmailSecurityBlockSenderResources,
		g.appendEmailSecurityImpersonationRegistryResources,
	} {
		if err := appendResources(context.Background(), api, "account-123"); err != nil {
			t.Fatalf("appendResources() error = %v", err)
		}
	}

	got := map[string]string{}
	for _, resource := range g.Resources {
		got[resource.InstanceInfo.Type] = resource.InstanceState.Meta["import_id"].(string)
	}
	want := map[string]string{
		"cloudflare_email_security_block_sender":           "account-123/101",
		"cloudflare_email_security_impersonation_registry": "account-123/202",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resources = %#v, want %#v", got, want)
	}
}

func TestRunCloudflareSecurityDiscoveriesSkipsOptionalErrors(t *testing.T) {
	calls := []string{}
	err := runCloudflareSecurityDiscoveries([]cloudflareSecurityDiscovery{
		{
			name:  "Page Shield policies",
			scope: "zone-123",
			discover: func() error {
				calls = append(calls, "optional")
				return testCloudflareRequestError("access denied: missing permission Page Shield Read")
			},
		},
		{
			name:  "API Shield operations",
			scope: "zone-123",
			discover: func() error {
				calls = append(calls, "next")
				return nil
			},
		},
	})
	if err != nil {
		t.Fatalf("runCloudflareSecurityDiscoveries() error = %v, want nil", err)
	}
	if want := []string{"optional", "next"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
}

func TestRunCloudflareSecurityDiscoveriesPropagatesUnexpectedErrors(t *testing.T) {
	expectedErr := errors.New("json decode failed")
	err := runCloudflareSecurityDiscoveries([]cloudflareSecurityDiscovery{
		{
			name:  "schema validation schemas",
			scope: "zone-123",
			discover: func() error {
				return expectedErr
			},
		},
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("runCloudflareSecurityDiscoveries() error = %v, want wrapped %v", err, expectedErr)
	}
	for _, expected := range []string{"schema validation schemas", "zone-123"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("runCloudflareSecurityDiscoveries() error = %q, want to contain %q", err, expected)
		}
	}
}

func TestCloudflareSecurityOptionalDiscoveryError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "generic not found propagates", err: testCloudflareNotFoundError("not found"), want: false},
		{name: "feature unavailable is optional", err: testCloudflareNotFoundError("feature is not available for this zone"), want: true},
		{name: "access denied is optional", err: testCloudflareRequestError("access denied: missing permission Domain Page Shield Read"), want: true},
		{name: "forbidden is optional", err: testCloudflareRequestError("forbidden: API Gateway is not enabled on this zone"), want: true},
		{name: "server error propagates", err: testCloudflareRequestError("internal server error"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cloudflareSecurityOptionalDiscoveryError(tt.err); got != tt.want {
				t.Fatalf("cloudflareSecurityOptionalDiscoveryError() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestCloudflareProviderIncludesSecurityService(t *testing.T) {
	services := (&CloudflareProvider{}).GetSupportedService()
	if _, ok := services["security"]; !ok {
		t.Fatal("security service is not registered")
	}
}

func TestCloudflareSecurityUnsupportedMetadataCoversDeferredResources(t *testing.T) {
	data, err := os.ReadFile("unsupported_resources.json")
	if err != nil {
		t.Fatalf("read unsupported resources: %v", err)
	}
	var metadata cloudflareUnsupportedResourcesFile
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("decode unsupported resources: %v", err)
	}

	seen := map[string]bool{}
	for _, resource := range metadata.Resources {
		seen[resource.Resource] = true
	}
	for _, resource := range []string{
		"cloudflare_api_shield_discovery_operation",
		"cloudflare_api_shield_operation_schema_validation_settings",
		"cloudflare_api_shield_schema",
		"cloudflare_api_shield_schema_validation_settings",
		"cloudflare_bot_management",
		"cloudflare_content_scanning",
		"cloudflare_content_scanning_expression",
		"cloudflare_email_security_trusted_domains",
		"cloudflare_observatory_scheduled_test",
		"cloudflare_schema_validation_operation_settings",
		"cloudflare_schema_validation_settings",
		"cloudflare_zero_trust_access_ai_controls_mcp_portal",
		"cloudflare_zero_trust_access_key_configuration",
	} {
		if !seen[resource] {
			t.Fatalf("unsupported metadata is missing %s", resource)
		}
	}
}

func newCloudflareSecurityTestAPI(t *testing.T, handler http.Handler) *cf.API {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	api, err := cf.NewWithAPIToken(
		"test-token",
		cf.BaseURL(server.URL),
		cf.UsingRateLimit(100000),
		cf.UsingRetryPolicy(0, 0, 0),
	)
	if err != nil {
		t.Fatalf("create Cloudflare test API: %v", err)
	}
	return api
}

func writeCloudflareSecurityTestResponse(t *testing.T, w http.ResponseWriter, result interface{}, resultInfo interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")

	payload := map[string]interface{}{
		"success": true,
		"result":  result,
	}
	if resultInfo != nil {
		payload["result_info"] = resultInfo
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode test response: %v", err)
	}
}

func ExampleSecurityGenerator_resources() {
	resources := []string{
		"cloudflare_api_shield",
		"cloudflare_api_shield_operation",
		"cloudflare_cloud_connector_rules",
		"cloudflare_custom_page_asset",
		"cloudflare_custom_pages",
		"cloudflare_email_security_block_sender",
		"cloudflare_email_security_impersonation_registry",
		"cloudflare_leaked_credential_check_rule",
		"cloudflare_page_shield_policy",
		"cloudflare_schema_validation_schemas",
		"cloudflare_token_validation_config",
		"cloudflare_token_validation_rules",
		"cloudflare_user_agent_blocking_rule",
		"cloudflare_vulnerability_scanner_credential_set",
		"cloudflare_vulnerability_scanner_target_environment",
	}
	fmt.Println(strings.Join(resources, ", "))
	// Output: cloudflare_api_shield, cloudflare_api_shield_operation, cloudflare_cloud_connector_rules, cloudflare_custom_page_asset, cloudflare_custom_pages, cloudflare_email_security_block_sender, cloudflare_email_security_impersonation_registry, cloudflare_leaked_credential_check_rule, cloudflare_page_shield_policy, cloudflare_schema_validation_schemas, cloudflare_token_validation_config, cloudflare_token_validation_rules, cloudflare_user_agent_blocking_rule, cloudflare_vulnerability_scanner_credential_set, cloudflare_vulnerability_scanner_target_environment
}
