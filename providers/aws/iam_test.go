// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/smithy-go"
)

func TestIamOpenIDConnectProviderName(t *testing.T) {
	tests := []struct {
		name        string
		providerARN string
		want        string
	}{
		{
			name:        "provider host",
			providerARN: "arn:aws:iam::123456789012:oidc-provider/token.actions.githubusercontent.com",
			want:        "token.actions.githubusercontent.com",
		},
		{
			name:        "provider path preserved",
			providerARN: "arn:aws:iam::123456789012:oidc-provider/example.com/team/prod",
			want:        "example.com/team/prod",
		},
		{
			name:        "fallback",
			providerARN: "arn:aws:iam::123456789012:saml-provider/example",
			want:        "example",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := iamOpenIDConnectProviderName(tt.providerARN)
			if got != tt.want {
				t.Fatalf("iamOpenIDConnectProviderName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIamResourceName(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "filters empty parts", parts: []string{"", "oidc", "", "example.com"}, want: "oidc/example.com"},
		{name: "preserves segment boundaries", parts: []string{"oidc", "example.com/team", "prod"}, want: "oidc/example.com/team/prod"},
		{name: "fallback", parts: nil, want: "iam_resource"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := iamResourceName(tt.parts...)
			if got != tt.want {
				t.Fatalf("iamResourceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIamServiceLinkedRoleResourceName(t *testing.T) {
	tests := []struct {
		roleName string
		want     string
	}{
		{roleName: "AWSServiceRoleForECS", want: "slr/AWSServiceRoleForECS"},
		{roleName: "AWSServiceRoleForElasticLoadBalancing", want: "slr/AWSServiceRoleForElasticLoadBalancing"},
	}
	for _, tt := range tests {
		got := iamResourceName("slr", tt.roleName)
		if got != tt.want {
			t.Fatalf("iamResourceName(slr, %q) = %q, want %q", tt.roleName, got, tt.want)
		}
	}
}

func TestIamServerCertificateResourceName(t *testing.T) {
	got := iamResourceName("cert", "my-server-cert")
	want := "cert/my-server-cert"
	if got != want {
		t.Fatalf("iamResourceName(cert, my-server-cert) = %q, want %q", got, want)
	}
}

func TestIamResourceMissing(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "typed no such entity", err: &types.NoSuchEntityException{}, want: true},
		{name: "wrapped no such entity", err: fmt.Errorf("wrapped: %w", &types.NoSuchEntityException{}), want: true},
		{name: "api no such entity code", err: &smithy.GenericAPIError{Code: "NoSuchEntityException"}, want: true},
		{name: "other", err: errors.New("boom"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := iamResourceMissing(tt.err)
			if got != tt.want {
				t.Fatalf("iamResourceMissing() = %v, want %v", got, tt.want)
			}
		})
	}
}
