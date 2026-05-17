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

	cf "github.com/cloudflare/cloudflare-go"
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
		"cloudflare_schema_validation_operation_settings",
		"cloudflare_schema_validation_settings",
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
		"cloudflare_page_shield_policy",
		"cloudflare_schema_validation_schemas",
		"cloudflare_vulnerability_scanner_credential_set",
		"cloudflare_vulnerability_scanner_target_environment",
	}
	fmt.Println(strings.Join(resources, ", "))
	// Output: cloudflare_api_shield, cloudflare_api_shield_operation, cloudflare_page_shield_policy, cloudflare_schema_validation_schemas, cloudflare_vulnerability_scanner_credential_set, cloudflare_vulnerability_scanner_target_environment
}
