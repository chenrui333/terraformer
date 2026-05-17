// SPDX-License-Identifier: Apache-2.0

package cloudflare

import "testing"

func TestZeroTrustGatewayAccountResourceUsesCompositeImportID(t *testing.T) {
	resource := zeroTrustGatewayAccountResource(
		"account-123",
		"resource-456",
		"example",
		"cloudflare_zero_trust_gateway_policy",
	)

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

func TestZeroTrustGatewaySingletonResourceUsesAccountImportID(t *testing.T) {
	resource := zeroTrustGatewaySingletonResource(
		"account-123",
		"example",
		"cloudflare_zero_trust_gateway_settings",
	)

	if resource.InstanceState.ID != "account-123" {
		t.Fatalf("resource ID = %q, want account-123", resource.InstanceState.ID)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "account-123" {
		t.Fatalf("import_id = %q, want account-123", got)
	}
}

func TestZeroTrustGatewayPolicyImportable(t *testing.T) {
	tests := []struct {
		name   string
		policy zeroTrustGatewayRawResource
		want   bool
	}{
		{name: "active local policy", policy: zeroTrustGatewayRawResource{"id": "policy-1"}, want: true},
		{name: "missing id", policy: zeroTrustGatewayRawResource{"name": "missing"}, want: false},
		{name: "deleted policy", policy: zeroTrustGatewayRawResource{"id": "policy-1", "deleted_at": "2026-05-17T00:00:00Z"}, want: false},
		{name: "read only tenant policy", policy: zeroTrustGatewayRawResource{"id": "policy-1", "read_only": true}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := zeroTrustGatewayPolicyImportable(tt.policy); got != tt.want {
				t.Fatalf("zeroTrustGatewayPolicyImportable() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestZeroTrustNetworkHostnameRouteImportable(t *testing.T) {
	tests := []struct {
		name  string
		route zeroTrustGatewayRawResource
		want  bool
	}{
		{name: "active route", route: zeroTrustGatewayRawResource{"id": "route-1"}, want: true},
		{name: "missing id", route: zeroTrustGatewayRawResource{"hostname": "app.example.com"}, want: false},
		{name: "deleted route", route: zeroTrustGatewayRawResource{"id": "route-1", "deleted_at": "2026-05-17T00:00:00Z"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := zeroTrustNetworkHostnameRouteImportable(tt.route); got != tt.want {
				t.Fatalf("zeroTrustNetworkHostnameRouteImportable() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestCloudflareProviderIncludesZeroTrustGatewayService(t *testing.T) {
	services := (&CloudflareProvider{}).GetSupportedService()
	if _, ok := services["zero_trust_gateway"]; !ok {
		t.Fatal("zero_trust_gateway service is not registered")
	}
}
