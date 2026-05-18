// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	cf "github.com/cloudflare/cloudflare-go"
)

func TestCloudflareProviderIncludesZeroTrustDeviceDLPService(t *testing.T) {
	services := (&CloudflareProvider{}).GetSupportedService()
	if _, ok := services["zero_trust_device_dlp"]; !ok {
		t.Fatal("zero_trust_device_dlp service is not registered")
	}
}

func TestZeroTrustDeviceDLPAccountResourceUsesCompositeImportID(t *testing.T) {
	resource, ok := zeroTrustDeviceDLPAccountResource(
		"account-123",
		"resource-456",
		"cloudflare_zero_trust_dex_rule",
		"dex_rule",
		"HTTP test",
	)
	if !ok {
		t.Fatal("expected account resource")
	}
	if resource.InstanceState.ID != "resource-456" {
		t.Fatalf("resource ID = %q, want resource-456", resource.InstanceState.ID)
	}
	if got := resource.InstanceState.Attributes["account_id"]; got != "account-123" {
		t.Fatalf("account_id = %q, want account-123", got)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "account-123/resource-456" {
		t.Fatalf("import_id = %q, want account-123/resource-456", got)
	}
}

func TestZeroTrustDeviceDLPSingletonResourceUsesAccountImportID(t *testing.T) {
	resource, ok := zeroTrustDeviceDLPSingletonResource(
		"account-123",
		"cloudflare_zero_trust_dlp_settings",
		"dlp_settings",
	)
	if !ok {
		t.Fatal("expected singleton resource")
	}
	if resource.InstanceState.ID != "account-123" {
		t.Fatalf("resource ID = %q, want account-123", resource.InstanceState.ID)
	}
	if got := resource.InstanceState.Attributes["account_id"]; got != "account-123" {
		t.Fatalf("account_id = %q, want account-123", got)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "account-123" {
		t.Fatalf("import_id = %q, want account-123", got)
	}
}

func TestListZeroTrustDeviceDLPResourcesPaginates(t *testing.T) {
	api := newZeroTrustDeviceDLPTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/account-123/devices/ip-profiles" {
			t.Fatalf("path = %q, want /accounts/account-123/devices/ip-profiles", r.URL.Path)
		}
		switch r.URL.Query().Get("page") {
		case "1":
			writeZeroTrustDeviceDLPTestResponse(t, w, []map[string]string{{"id": "ip-profile-1"}}, map[string]int{
				"page":        1,
				"per_page":    cloudflarePageSize,
				"total_pages": 2,
			})
		case "2":
			writeZeroTrustDeviceDLPTestResponse(t, w, []map[string]string{{"id": "ip-profile-2"}}, map[string]int{
				"page":        2,
				"per_page":    cloudflarePageSize,
				"total_pages": 2,
			})
		default:
			t.Fatalf("page query = %q, want 1 or 2", r.URL.Query().Get("page"))
		}
	}))

	resources, err := listZeroTrustDeviceDLPResources(context.Background(), api, "/accounts/account-123/devices/ip-profiles")
	if err != nil {
		t.Fatalf("listZeroTrustDeviceDLPResources() error = %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("resource count = %d, want 2", len(resources))
	}
	if got := zeroTrustDeviceDLPString(resources[0], "id"); got != "ip-profile-1" {
		t.Fatalf("first id = %q, want ip-profile-1", got)
	}
	if got := zeroTrustDeviceDLPString(resources[1], "id"); got != "ip-profile-2" {
		t.Fatalf("second id = %q, want ip-profile-2", got)
	}
}

func TestListZeroTrustDeviceDLPResourcesHandlesObjectWrappedDEXTests(t *testing.T) {
	api := newZeroTrustDeviceDLPTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/account-123/dex/devices/dex_tests" {
			t.Fatalf("path = %q, want /accounts/account-123/dex/devices/dex_tests", r.URL.Path)
		}
		writeZeroTrustDeviceDLPTestResponse(t, w, map[string]interface{}{
			"dex_tests": []map[string]string{{"test_id": "dex-test-1"}},
		}, nil)
	}))

	resources, err := listZeroTrustDeviceDLPResources(context.Background(), api, "/accounts/account-123/dex/devices/dex_tests")
	if err != nil {
		t.Fatalf("listZeroTrustDeviceDLPResources() error = %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("resource count = %d, want 1", len(resources))
	}
	if got := zeroTrustDeviceDLPString(resources[0], "test_id"); got != "dex-test-1" {
		t.Fatalf("test_id = %q, want dex-test-1", got)
	}
}

func TestZeroTrustDLPCustomProfileImportable(t *testing.T) {
	tests := []struct {
		name    string
		profile zeroTrustDeviceDLPRawResource
		want    bool
	}{
		{
			name: "plain custom profile",
			profile: zeroTrustDeviceDLPRawResource{
				"id":   "profile-123",
				"type": "custom",
			},
			want: true,
		},
		{
			name: "missing id",
			profile: zeroTrustDeviceDLPRawResource{
				"type": "custom",
			},
			want: false,
		},
		{
			name: "predefined profile",
			profile: zeroTrustDeviceDLPRawResource{
				"id":   "profile-123",
				"type": "predefined",
			},
			want: false,
		},
		{
			name: "inline entries",
			profile: zeroTrustDeviceDLPRawResource{
				"id":      "profile-123",
				"type":    "custom",
				"entries": []interface{}{map[string]interface{}{"entry_id": "entry-123"}},
			},
			want: false,
		},
		{
			name: "shared entries",
			profile: zeroTrustDeviceDLPRawResource{
				"id":             "profile-123",
				"type":           "custom",
				"shared_entries": []interface{}{map[string]interface{}{"entry_id": "entry-123"}},
			},
			want: false,
		},
		{
			name: "enabled context awareness",
			profile: zeroTrustDeviceDLPRawResource{
				"id":   "profile-123",
				"type": "custom",
				"context_awareness": map[string]interface{}{
					"enabled": true,
				},
			},
			want: false,
		},
		{
			name: "inactive context awareness",
			profile: zeroTrustDeviceDLPRawResource{
				"id":   "profile-123",
				"type": "custom",
				"context_awareness": map[string]interface{}{
					"enabled": false,
					"skip": map[string]interface{}{
						"files": false,
					},
				},
			},
			want: true,
		},
		{
			name: "context awareness skip settings",
			profile: zeroTrustDeviceDLPRawResource{
				"id":   "profile-123",
				"type": "custom",
				"context_awareness": map[string]interface{}{
					"enabled": false,
					"skip": map[string]interface{}{
						"files": true,
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := zeroTrustDLPCustomProfileImportable(tt.profile); got != tt.want {
				t.Fatalf("zeroTrustDLPCustomProfileImportable() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestAppendZeroTrustDeviceDLPResourcesDiscoversSupportedResources(t *testing.T) {
	api := newZeroTrustDeviceDLPTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/accounts/account-123/devices/policy":
			writeZeroTrustDeviceDLPTestResponse(t, w, map[string]interface{}{
				"policy_id":        "default-policy",
				"name":             "Default",
				"fallback_domains": []map[string]string{{"suffix": "corp.example"}},
			}, nil)
		case "/accounts/account-123/devices/policies":
			writeZeroTrustDeviceDLPTestResponse(t, w, []map[string]interface{}{
				{
					"policy_id": "default-policy",
					"name":      "Default",
					"default":   true,
				},
				{
					"policy_id":        "custom-policy",
					"name":             "Engineering",
					"fallback_domains": []map[string]string{{"suffix": "eng.example"}},
				},
			}, nil)
		case "/accounts/account-123/devices/networks":
			writeZeroTrustDeviceDLPTestResponse(t, w, []map[string]string{{
				"network_id": "network-123",
				"name":       "Office",
			}}, nil)
		case "/accounts/account-123/devices/ip-profiles":
			writeZeroTrustDeviceDLPTestResponse(t, w, []map[string]string{{
				"id":   "ip-profile-123",
				"name": "Trusted devices",
			}}, nil)
		case "/accounts/account-123/dex/rules":
			writeZeroTrustDeviceDLPTestResponse(t, w, []map[string]string{{
				"id":   "dex-rule-123",
				"name": "HTTP tests",
			}}, nil)
		case "/accounts/account-123/dex/devices/dex_tests":
			writeZeroTrustDeviceDLPTestResponse(t, w, []map[string]string{{
				"test_id": "dex-test-123",
				"name":    "Homepage",
			}}, nil)
		case "/accounts/account-123/dlp/profiles":
			writeZeroTrustDeviceDLPTestResponse(t, w, []map[string]interface{}{
				{"id": "dlp-profile-123", "name": "Secrets", "type": "custom"},
				{
					"id":      "dlp-profile-with-entries",
					"name":    "Secrets with entries",
					"type":    "custom",
					"entries": []map[string]string{{"entry_id": "dlp-entry-123"}},
				},
				{
					"id":   "dlp-profile-with-context",
					"name": "Secrets with context",
					"type": "custom",
					"context_awareness": map[string]bool{
						"enabled": true,
					},
				},
				{"id": "predefined-profile", "name": "Credentials", "type": "predefined"},
			}, nil)
		case "/accounts/account-123/dlp/entries":
			writeZeroTrustDeviceDLPTestResponse(t, w, []map[string]string{
				{"id": "dlp-entry-123", "name": "Token", "type": "custom"},
				{"id": "predefined-entry", "name": "Credit cards", "type": "predefined"},
			}, nil)
		case "/accounts/account-123/dlp/settings":
			writeZeroTrustDeviceDLPTestResponse(t, w, map[string]bool{"ai_context_enabled": true}, nil)
		default:
			t.Fatalf("unexpected discovery path %q", r.URL.Path)
		}
	}))

	g := &ZeroTrustDeviceDLPGenerator{}
	if err := g.appendZeroTrustDeviceDLPResources(context.Background(), api, "account-123"); err != nil {
		t.Fatalf("appendZeroTrustDeviceDLPResources() error = %v", err)
	}
	if len(g.Resources) != 11 {
		t.Fatalf("resource count = %d, want 11", len(g.Resources))
	}

	got := map[string]string{}
	for _, resource := range g.Resources {
		got[resource.InstanceInfo.Type] = resource.InstanceState.Meta["import_id"].(string)
	}
	for resourceType, wantImportID := range map[string]string{
		"cloudflare_zero_trust_device_default_profile":                       "account-123",
		"cloudflare_zero_trust_device_default_profile_local_domain_fallback": "account-123",
		"cloudflare_zero_trust_device_custom_profile":                        "account-123/custom-policy",
		"cloudflare_zero_trust_device_custom_profile_local_domain_fallback":  "account-123/custom-policy",
		"cloudflare_zero_trust_device_managed_networks":                      "account-123/network-123",
		"cloudflare_zero_trust_device_ip_profile":                            "account-123/ip-profile-123",
		"cloudflare_zero_trust_dex_rule":                                     "account-123/dex-rule-123",
		"cloudflare_zero_trust_dex_test":                                     "account-123/dex-test-123",
		"cloudflare_zero_trust_dlp_custom_profile":                           "account-123/dlp-profile-123",
		"cloudflare_zero_trust_dlp_custom_entry":                             "account-123/dlp-entry-123",
		"cloudflare_zero_trust_dlp_settings":                                 "account-123",
	} {
		if got[resourceType] != wantImportID {
			t.Fatalf("%s import ID = %q, want %q", resourceType, got[resourceType], wantImportID)
		}
	}
}

func TestZeroTrustDeviceDLPUnsupportedResourcesMetadata(t *testing.T) {
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
		"cloudflare_zero_trust_device_default_profile_certificates",
		"cloudflare_zero_trust_device_posture_integration",
		"cloudflare_zero_trust_device_posture_rule",
		"cloudflare_zero_trust_device_settings",
		"cloudflare_zero_trust_device_subnet",
		"cloudflare_zero_trust_dlp_dataset",
		"cloudflare_zero_trust_dlp_entry",
		"cloudflare_zero_trust_dlp_integration_entry",
		"cloudflare_zero_trust_dlp_predefined_entry",
		"cloudflare_zero_trust_dlp_predefined_profile",
		"cloudflare_zero_trust_risk_behavior",
		"cloudflare_zero_trust_risk_scoring_integration",
	} {
		if !seen[resource] {
			t.Fatalf("unsupported metadata is missing %s", resource)
		}
	}
	for _, resource := range []string{
		"cloudflare_zero_trust_dlp_custom_entry",
		"cloudflare_zero_trust_dlp_custom_profile",
	} {
		if seen[resource] {
			t.Fatalf("supported resource %s should not remain in unsupported metadata", resource)
		}
	}
}

func newZeroTrustDeviceDLPTestAPI(t *testing.T, handler http.Handler) *cf.API {
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

func writeZeroTrustDeviceDLPTestResponse(t *testing.T, w http.ResponseWriter, result interface{}, resultInfo interface{}) {
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
		t.Fatalf("write response: %v", err)
	}
}
