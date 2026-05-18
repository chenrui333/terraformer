// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"net/http"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestCloudflareTunnelRouteResource(t *testing.T) {
	resource, ok := cloudflareTunnelRouteResource("account-123", cloudflareTunnelRoute{
		ID:               "route-456",
		Network:          "10.0.0.0/16",
		TunnelID:         "tunnel-789",
		Comment:          "private network",
		TunType:          "cfd_tunnel",
		VirtualNetworkID: "vnet-123",
	})
	if !ok {
		t.Fatal("expected tunnel route resource")
	}
	if got := resource.InstanceInfo.Type; got != "cloudflare_zero_trust_tunnel_cloudflared_route" {
		t.Fatalf("resource type = %q, want cloudflare_zero_trust_tunnel_cloudflared_route", got)
	}
	if got := resource.InstanceState.ID; got != "route-456" {
		t.Fatalf("resource ID = %q, want route-456", got)
	}
	if got := resource.InstanceState.Attributes["account_id"]; got != "account-123" {
		t.Fatalf("account_id = %q, want account-123", got)
	}
	if got := resource.InstanceState.Attributes["network"]; got != "10.0.0.0/16" {
		t.Fatalf("network = %q, want 10.0.0.0/16", got)
	}
	if got := resource.InstanceState.Attributes["tunnel_id"]; got != "tunnel-789" {
		t.Fatalf("tunnel_id = %q, want tunnel-789", got)
	}
	if got := resource.InstanceState.Attributes["comment"]; got != "private network" {
		t.Fatalf("comment = %q, want private network", got)
	}
	if got := resource.InstanceState.Attributes["virtual_network_id"]; got != "vnet-123" {
		t.Fatalf("virtual_network_id = %q, want vnet-123", got)
	}
	if got, want := resource.ResourceName, terraformutils.TfSanitize(cloudflareResourceName("account-123", "tunnel_route", "10.0.0.0/16", "route-456")); got != want {
		t.Fatalf("resource name = %q, want %q", got, want)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "account-123/route-456" {
		t.Fatalf("import_id = %q, want account-123/route-456", got)
	}
}

func TestCloudflareTunnelRouteResourceSkipsUnsafeRoutes(t *testing.T) {
	deletedAt := "2026-01-02T03:04:05Z"
	tests := []struct {
		name    string
		account string
		route   cloudflareTunnelRoute
	}{
		{name: "missing account", account: "", route: cloudflareTunnelRoute{ID: "route-1", Network: "10.0.0.0/16", TunnelID: "tunnel-1"}},
		{name: "missing id", account: "account-123", route: cloudflareTunnelRoute{Network: "10.0.0.0/16", TunnelID: "tunnel-1"}},
		{name: "missing network", account: "account-123", route: cloudflareTunnelRoute{ID: "route-1", TunnelID: "tunnel-1"}},
		{name: "missing tunnel", account: "account-123", route: cloudflareTunnelRoute{ID: "route-1", Network: "10.0.0.0/16"}},
		{name: "deleted", account: "account-123", route: cloudflareTunnelRoute{ID: "route-1", Network: "10.0.0.0/16", TunnelID: "tunnel-1", DeletedAt: &deletedAt}},
		{name: "warp connector", account: "account-123", route: cloudflareTunnelRoute{ID: "route-1", Network: "10.0.0.0/16", TunnelID: "tunnel-1", TunType: "warp_connector"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := cloudflareTunnelRouteResource(tt.account, tt.route); ok {
				t.Fatal("expected unsafe tunnel route to be skipped")
			}
		})
	}
}

func TestAppendTunnelRouteResourcesPaginatesAndSkipsNonCloudflaredRoutes(t *testing.T) {
	api := newCloudflareNetworkEdgeTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/account-123/teamnet/routes" {
			t.Fatalf("path = %q, want /accounts/account-123/teamnet/routes", r.URL.Path)
		}
		if got := r.URL.Query().Get("is_deleted"); got != "false" {
			t.Fatalf("is_deleted query = %q, want false", got)
		}
		if got := r.URL.Query().Get("tun_types"); got != "cfd_tunnel" {
			t.Fatalf("tun_types query = %q, want cfd_tunnel", got)
		}
		switch r.URL.Query().Get("cursor") {
		case "":
			if got := r.URL.Query().Get("page"); got != "1" {
				t.Fatalf("page query = %q, want 1", got)
			}
			writeCloudflareNetworkEdgeTestResponse(t, w, []map[string]string{
				{"id": "route-1", "network": "10.0.0.0/16", "tunnel_id": "tunnel-1", "tun_type": "cfd_tunnel"},
				{"id": "route-warp", "network": "10.1.0.0/16", "tunnel_id": "tunnel-warp", "tun_type": "warp_connector"},
			}, map[string]interface{}{
				"cursors": map[string]string{"after": "cursor-2"},
			})
		case "cursor-2":
			writeCloudflareNetworkEdgeTestResponse(t, w, []map[string]string{
				{"id": "route-2", "network": "10.2.0.0/16", "tunnel_id": "tunnel-2", "tun_type": "cfd_tunnel"},
			}, map[string]interface{}{
				"cursors": map[string]string{},
			})
		default:
			t.Fatalf("cursor query = %q, want empty or cursor-2", r.URL.Query().Get("cursor"))
		}
	}))

	generator := &TunnelGenerator{}
	if err := generator.appendTunnelRouteResources(context.Background(), api, "account-123"); err != nil {
		t.Fatalf("appendTunnelRouteResources() error = %v", err)
	}
	if len(generator.Resources) != 2 {
		t.Fatalf("resource count = %d, want 2", len(generator.Resources))
	}
	if got := generator.Resources[0].InstanceState.Meta["import_id"]; got != "account-123/route-1" {
		t.Fatalf("first import_id = %q, want account-123/route-1", got)
	}
	if got := generator.Resources[1].InstanceState.Meta["import_id"]; got != "account-123/route-2" {
		t.Fatalf("second import_id = %q, want account-123/route-2", got)
	}
}
