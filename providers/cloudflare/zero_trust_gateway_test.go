// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"testing"

	cf "github.com/cloudflare/cloudflare-go"
)

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

func TestZeroTrustGatewayOptionalUnavailableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "generic not found propagates",
			err:  testCloudflareNotFoundError("not found"),
			want: false,
		},
		{
			name: "missing endpoint propagates",
			err:  testCloudflareNotFoundError("The requested endpoint was not found"),
			want: false,
		},
		{
			name: "zero trust account not configured is optional",
			err:  testCloudflareNotFoundError("Zero Trust account is not configured"),
			want: true,
		},
		{
			name: "zero trust feature unavailable request is optional",
			err:  testCloudflareRequestError("feature is not available for this Zero Trust account"),
			want: true,
		},
		{
			name: "generic request not found propagates",
			err:  testCloudflareRequestError("not found"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := zeroTrustGatewayOptionalUnavailableError(tt.err); got != tt.want {
				t.Fatalf("zeroTrustGatewayOptionalUnavailableError() = %t, want %t", got, tt.want)
			}
		})
	}
}

func testCloudflareNotFoundError(messages ...string) error {
	err := cf.NewNotFoundError(testCloudflareError(messages...))
	return &err
}

func testCloudflareRequestError(messages ...string) error {
	err := cf.NewRequestError(testCloudflareError(messages...))
	return &err
}

func testCloudflareError(messages ...string) *cf.Error {
	responseInfo := make([]cf.ResponseInfo, 0, len(messages))
	for _, message := range messages {
		responseInfo = append(responseInfo, cf.ResponseInfo{Message: message})
	}
	return &cf.Error{
		Errors:        responseInfo,
		ErrorMessages: messages,
	}
}

func TestCloudflareProviderIncludesZeroTrustGatewayService(t *testing.T) {
	services := (&CloudflareProvider{}).GetSupportedService()
	if _, ok := services["zero_trust_gateway"]; !ok {
		t.Fatal("zero_trust_gateway service is not registered")
	}
}
