// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestVerifiedAccessResourceConstructors(t *testing.T) {
	tests := []struct {
		name       string
		resource   terraformResourceResult
		wantID     string
		wantType   string
		wantAttr   map[string]string
		wantExists bool
	}{
		{
			name: "instance",
			resource: newTerraformResourceResult(newVerifiedAccessInstanceResource(ec2types.VerifiedAccessInstance{
				VerifiedAccessInstanceId: aws.String("vai-123"),
			})),
			wantID:     "vai-123",
			wantType:   verifiedAccessInstanceResourceType,
			wantExists: true,
		},
		{
			name:       "instance empty ID",
			resource:   newTerraformResourceResult(newVerifiedAccessInstanceResource(ec2types.VerifiedAccessInstance{})),
			wantExists: false,
		},
		{
			name: "group",
			resource: newTerraformResourceResult(newVerifiedAccessGroupResource(ec2types.VerifiedAccessGroup{
				VerifiedAccessGroupId:    aws.String("vag-123"),
				VerifiedAccessInstanceId: aws.String("vai-123"),
			})),
			wantID:     "vag-123",
			wantType:   verifiedAccessGroupResourceType,
			wantAttr:   map[string]string{"verifiedaccess_instance_id": "vai-123"},
			wantExists: true,
		},
		{
			name: "group deleting",
			resource: newTerraformResourceResult(newVerifiedAccessGroupResource(ec2types.VerifiedAccessGroup{
				VerifiedAccessGroupId:    aws.String("vag-123"),
				VerifiedAccessInstanceId: aws.String("vai-123"),
				DeletionTime:             aws.String("2026-05-11T00:00:00Z"),
			})),
			wantExists: false,
		},
		{
			name: "endpoint active",
			resource: newTerraformResourceResult(newVerifiedAccessEndpointResource(ec2types.VerifiedAccessEndpoint{
				VerifiedAccessEndpointId: aws.String("vae-123"),
				VerifiedAccessGroupId:    aws.String("vag-123"),
				Status: &ec2types.VerifiedAccessEndpointStatus{
					Code: ec2types.VerifiedAccessEndpointStatusCodeActive,
				},
			})),
			wantID:     "vae-123",
			wantType:   verifiedAccessEndpointResourceType,
			wantAttr:   map[string]string{"verifiedaccess_group_id": "vag-123"},
			wantExists: true,
		},
		{
			name: "endpoint deleting",
			resource: newTerraformResourceResult(newVerifiedAccessEndpointResource(ec2types.VerifiedAccessEndpoint{
				VerifiedAccessEndpointId: aws.String("vae-123"),
				VerifiedAccessGroupId:    aws.String("vag-123"),
				Status: &ec2types.VerifiedAccessEndpointStatus{
					Code: ec2types.VerifiedAccessEndpointStatusCodeDeleting,
				},
			})),
			wantExists: false,
		},
		{
			name: "trust provider without secret",
			resource: newTerraformResourceResult(newVerifiedAccessTrustProviderResource(ec2types.VerifiedAccessTrustProvider{
				VerifiedAccessTrustProviderId: aws.String("vatp-123"),
				PolicyReferenceName:           aws.String("idc"),
				UserTrustProviderType:         ec2types.UserTrustProviderTypeIamIdentityCenter,
			})),
			wantID:     "vatp-123",
			wantType:   verifiedAccessTrustProviderResourceType,
			wantExists: true,
		},
		{
			name: "trust provider oidc skipped",
			resource: newTerraformResourceResult(newVerifiedAccessTrustProviderResource(ec2types.VerifiedAccessTrustProvider{
				VerifiedAccessTrustProviderId: aws.String("vatp-123"),
				OidcOptions:                   &ec2types.OidcOptions{},
			})),
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.resource.ok != tt.wantExists {
				t.Fatalf("resource exists = %t, want %t", tt.resource.ok, tt.wantExists)
			}
			if !tt.wantExists {
				return
			}
			resource := tt.resource.resource
			if resource.InstanceState.ID != tt.wantID {
				t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, tt.wantID)
			}
			if resource.InstanceInfo.Type != tt.wantType {
				t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, tt.wantType)
			}
			for key, want := range tt.wantAttr {
				if got := resource.InstanceState.Attributes[key]; got != want {
					t.Fatalf("attribute %s = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestVerifiedAccessEndpointStatusImportable(t *testing.T) {
	tests := []struct {
		name   string
		status *ec2types.VerifiedAccessEndpointStatus
		want   bool
	}{
		{name: "active", status: &ec2types.VerifiedAccessEndpointStatus{Code: ec2types.VerifiedAccessEndpointStatusCodeActive}, want: true},
		{name: "updating", status: &ec2types.VerifiedAccessEndpointStatus{Code: ec2types.VerifiedAccessEndpointStatusCodeUpdating}, want: false},
		{name: "nil", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := verifiedAccessEndpointStatusImportable(tt.status); got != tt.want {
				t.Fatalf("verifiedAccessEndpointStatusImportable() = %t, want %t", got, tt.want)
			}
		})
	}
}
