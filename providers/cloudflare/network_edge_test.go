// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
	cf "github.com/cloudflare/cloudflare-go"
)

func TestCloudflareZoneNetworkEdgeResourceUsesCompositeImportID(t *testing.T) {
	resource, ok := cloudflareZoneNetworkEdgeResource(
		cf.Zone{ID: "zone-123", Name: "example.com"},
		"app-456",
		"cloudflare_spectrum_application",
		"spectrum_application",
		"ssh.example.com",
		"tcp/22",
	)
	if !ok {
		t.Fatal("expected zone network edge resource")
	}
	if resource.InstanceInfo.Type != "cloudflare_spectrum_application" {
		t.Fatalf("resource type = %q, want cloudflare_spectrum_application", resource.InstanceInfo.Type)
	}
	if resource.InstanceState.ID != "app-456" {
		t.Fatalf("resource ID = %q, want app-456", resource.InstanceState.ID)
	}
	if got := resource.InstanceState.Attributes["zone_id"]; got != "zone-123" {
		t.Fatalf("zone_id = %q, want zone-123", got)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "zone-123/app-456" {
		t.Fatalf("import_id = %q, want zone-123/app-456", got)
	}
}

func TestCloudflareRegionalHostnameResourcePreservesID(t *testing.T) {
	resource, ok := cloudflareRegionalHostnameResource(
		cf.Zone{ID: "zone-123", Name: "example.com"},
		"eu.example.com",
	)
	if !ok {
		t.Fatal("expected regional hostname resource")
	}
	if resource.InstanceInfo.Type != "cloudflare_regional_hostname" {
		t.Fatalf("resource type = %q, want cloudflare_regional_hostname", resource.InstanceInfo.Type)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "zone-123/eu.example.com" {
		t.Fatalf("import_id = %q, want zone-123/eu.example.com", got)
	}
	preserveID, ok := resource.InstanceState.Meta[tfcompat.MetaKeyPreserveIDAfterRefresh].(bool)
	if !ok || !preserveID {
		t.Fatalf("preserve ID metadata = %#v, want true", resource.InstanceState.Meta[tfcompat.MetaKeyPreserveIDAfterRefresh])
	}
}

func TestCloudflareAccountNetworkEdgeResourceUsesCompositeImportID(t *testing.T) {
	resource, ok := cloudflareAccountNetworkEdgeResource(
		"account-123",
		"map-456",
		"cloudflare_address_map",
		"address_map",
		"production",
	)
	if !ok {
		t.Fatal("expected account network edge resource")
	}
	if resource.InstanceInfo.Type != "cloudflare_address_map" {
		t.Fatalf("resource type = %q, want cloudflare_address_map", resource.InstanceInfo.Type)
	}
	if got := resource.InstanceState.Attributes["account_id"]; got != "account-123" {
		t.Fatalf("account_id = %q, want account-123", got)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "account-123/map-456" {
		t.Fatalf("import_id = %q, want account-123/map-456", got)
	}
}

func TestCloudflareMagicTransitSiteChildResourceUsesCompositeImportID(t *testing.T) {
	resource, ok := cloudflareMagicTransitSiteChildResource(
		"account-123",
		"site-456",
		"lan-789",
		"cloudflare_magic_transit_site_lan",
		"magic_transit_site_lan",
		"office",
	)
	if !ok {
		t.Fatal("expected Magic Transit site child resource")
	}
	if got := resource.InstanceState.Attributes["account_id"]; got != "account-123" {
		t.Fatalf("account_id = %q, want account-123", got)
	}
	if got := resource.InstanceState.Attributes["site_id"]; got != "site-456" {
		t.Fatalf("site_id = %q, want site-456", got)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "account-123/site-456/lan-789" {
		t.Fatalf("import_id = %q, want account-123/site-456/lan-789", got)
	}
}

func TestCloudflareNetworkEdgeResourceConstructorsSkipMissingIDs(t *testing.T) {
	if _, ok := cloudflareZoneNetworkEdgeResource(cf.Zone{ID: "zone-123"}, "", "cloudflare_spectrum_application", "spectrum_application"); ok {
		t.Fatal("expected zone resource without resource ID to be skipped")
	}
	if _, ok := cloudflareRegionalHostnameResource(cf.Zone{ID: "zone-123"}, ""); ok {
		t.Fatal("expected regional hostname without hostname to be skipped")
	}
	if _, ok := cloudflareAccountNetworkEdgeResource("", "map-123", "cloudflare_address_map", "address_map"); ok {
		t.Fatal("expected account resource without account ID to be skipped")
	}
	if _, ok := cloudflareMagicTransitSiteChildResource("account-123", "", "lan-123", "cloudflare_magic_transit_site_lan", "lan"); ok {
		t.Fatal("expected site child resource without site ID to be skipped")
	}
}

func TestCloudflareAddressMapImportableSkipsManagedMaps(t *testing.T) {
	tests := []struct {
		name       string
		addressMap cloudflareNetworkEdgeRawResource
		want       bool
	}{
		{name: "modifiable", addressMap: cloudflareNetworkEdgeRawResource{"can_delete": true, "can_modify_ips": true}, want: true},
		{name: "cannot delete", addressMap: cloudflareNetworkEdgeRawResource{"can_delete": false, "can_modify_ips": true}, want: false},
		{name: "cannot modify IPs", addressMap: cloudflareNetworkEdgeRawResource{"can_delete": true, "can_modify_ips": false}, want: false},
		{name: "missing flags", addressMap: cloudflareNetworkEdgeRawResource{}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cloudflareAddressMapImportable(tt.addressMap); got != tt.want {
				t.Fatalf("cloudflareAddressMapImportable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunCloudflareNetworkEdgeDiscoveriesRollsBackFailedDiscovery(t *testing.T) {
	resources := []terraformutils.Resource{}
	err := runCloudflareNetworkEdgeDiscoveries([]cloudflareNetworkEdgeDiscovery{
		{
			name:      "fails",
			scope:     "account-123",
			resources: &resources,
			discover: func() error {
				resources = append(resources, terraformutils.NewResource("partial", "partial", "cloudflare_address_map", "cloudflare", nil, nil, nil))
				return errors.New("unexpected response")
			},
		},
	})
	if err == nil {
		t.Fatal("expected non-optional discovery error")
	}
	if len(resources) != 0 {
		t.Fatalf("resources after failed discovery = %#v, want none", resources)
	}
}

func TestRunCloudflareNetworkEdgeDiscoveriesContinuesAfterOptionalError(t *testing.T) {
	resources := []terraformutils.Resource{}
	err := runCloudflareNetworkEdgeDiscoveries([]cloudflareNetworkEdgeDiscovery{
		{
			name:      "permission gated",
			scope:     "account-123",
			resources: &resources,
			discover: func() error {
				resources = append(resources, terraformutils.NewResource("partial", "partial", "cloudflare_address_map", "cloudflare", nil, nil, nil))
				requestErr := cf.NewRequestError(&cf.Error{ErrorMessages: []string{"access denied"}})
				return &requestErr
			},
		},
		{
			name:      "succeeds",
			scope:     "account-123",
			resources: &resources,
			discover: func() error {
				resources = append(resources, terraformutils.NewResource("complete", "complete", "cloudflare_address_map", "cloudflare", nil, nil, nil))
				return nil
			},
		},
	})
	if err != nil {
		t.Fatalf("runCloudflareNetworkEdgeDiscoveries() error = %v, want nil", err)
	}
	if len(resources) != 1 || resources[0].InstanceState.ID != "complete" {
		t.Fatalf("resources after optional discovery rollback = %#v, want only complete", resources)
	}
}

func TestCloudflareNetworkEdgeOptionalErrorMessageDoesNotHideGenericNotFound(t *testing.T) {
	if cloudflareNetworkEdgeOptionalErrorMessage("not found", nil) {
		t.Fatal("generic not found should not be treated as optional")
	}
	if !cloudflareNetworkEdgeOptionalErrorMessage("", []string{"feature is not available on this plan"}) {
		t.Fatal("feature-gated errors should be treated as optional")
	}
}

func TestCloudflareNetworkEdgeOptionalDiscoveryErrorHandlesAuthErrors(t *testing.T) {
	authenticationErr := cf.NewAuthenticationError(&cf.Error{ErrorMessages: []string{"not authorized"}})
	if !cloudflareNetworkEdgeOptionalDiscoveryError(&authenticationErr) {
		t.Fatal("authentication errors should be treated as optional when they match optional markers")
	}

	authorizationErr := cf.NewAuthorizationError(&cf.Error{ErrorMessages: []string{"missing permission"}})
	if !cloudflareNetworkEdgeOptionalDiscoveryError(&authorizationErr) {
		t.Fatal("authorization errors should be treated as optional when they match optional markers")
	}
}

func TestListCloudflareNetworkEdgeResourcesPaginates(t *testing.T) {
	api := newCloudflareNetworkEdgeTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/account-123/addressing/address_maps" {
			t.Fatalf("path = %q, want /accounts/account-123/addressing/address_maps", r.URL.Path)
		}
		switch r.URL.Query().Get("cursor") {
		case "":
			if got := r.URL.Query().Get("page"); got != "1" {
				t.Fatalf("page query = %q, want 1", got)
			}
			writeCloudflareNetworkEdgeTestResponse(t, w, []map[string]string{{"id": "map-1"}}, map[string]interface{}{
				"cursors": map[string]string{"after": "cursor-2"},
			})
		case "cursor-2":
			writeCloudflareNetworkEdgeTestResponse(t, w, []map[string]string{{"id": "map-2"}}, map[string]interface{}{
				"cursors": map[string]string{},
			})
		default:
			t.Fatalf("cursor query = %q, want empty or cursor-2", r.URL.Query().Get("cursor"))
		}
	}))

	resources, err := listCloudflareNetworkEdgeResources(context.Background(), api, "/accounts/account-123/addressing/address_maps")
	if err != nil {
		t.Fatalf("listCloudflareNetworkEdgeResources() error = %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("resource count = %d, want 2", len(resources))
	}
	if got := cloudflareNetworkEdgeString(resources[0], "id"); got != "map-1" {
		t.Fatalf("first id = %q, want map-1", got)
	}
	if got := cloudflareNetworkEdgeString(resources[1], "id"); got != "map-2" {
		t.Fatalf("second id = %q, want map-2", got)
	}
}

func TestNetworkEdgeUnsupportedResourcesMetadata(t *testing.T) {
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
		"cloudflare_byo_ip_prefix",
		"cloudflare_magic_network_monitoring_configuration",
		"cloudflare_magic_transit_connector",
	} {
		if !seen[resource] {
			t.Fatalf("unsupported metadata is missing %s", resource)
		}
	}
}

func newCloudflareNetworkEdgeTestAPI(t *testing.T, handler http.Handler) *cf.API {
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

func writeCloudflareNetworkEdgeTestResponse(t *testing.T, w http.ResponseWriter, result interface{}, resultInfo interface{}) {
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
