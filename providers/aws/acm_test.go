// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	acmtypes "github.com/aws/aws-sdk-go-v2/service/acm/types"
)

func TestACMCertificateStatusImportable(t *testing.T) {
	tests := []struct {
		name   string
		status acmtypes.CertificateStatus
		want   bool
	}{
		{name: "validation timed out", status: acmtypes.CertificateStatusValidationTimedOut},
		{name: "failed", status: acmtypes.CertificateStatusFailed, want: true},
		{name: "expired", status: acmtypes.CertificateStatusExpired, want: true},
		{name: "revoked", status: acmtypes.CertificateStatusRevoked, want: true},
		{name: "pending validation", status: acmtypes.CertificateStatusPendingValidation, want: true},
		{name: "issued", status: acmtypes.CertificateStatusIssued, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := acmCertificateStatusImportable(tt.status); got != tt.want {
				t.Fatalf("acmCertificateStatusImportable(%q) = %t, want %t", tt.status, got, tt.want)
			}
		})
	}
}

func TestNewACMCertificateResourceUsesSummaryStatus(t *testing.T) {
	resource, ok := newACMCertificateResource(acmtypes.CertificateSummary{
		CertificateArn: aws.String("arn:aws:acm:us-east-1:123456789012:certificate/cert-1"),
		DomainName:     aws.String("example.com."),
		Status:         acmtypes.CertificateStatusIssued,
	})
	if !ok {
		t.Fatal("newACMCertificateResource() ok = false, want true")
	}
	if got := resource.InstanceState.ID; got != "arn:aws:acm:us-east-1:123456789012:certificate/cert-1" {
		t.Fatalf("InstanceState.ID = %q, want certificate ARN", got)
	}
	if got := resource.ResourceName; got != "tfer--cert-1_example-002E-com" {
		t.Fatalf("ResourceName = %q, want tfer--cert-1_example-002E-com", got)
	}
	if got := resource.InstanceState.Attributes["domain_name"]; got != "example.com" {
		t.Fatalf("domain_name = %q, want example.com", got)
	}

	_, ok = newACMCertificateResource(acmtypes.CertificateSummary{
		CertificateArn: aws.String("arn:aws:acm:us-east-1:123456789012:certificate/cert-2"),
		DomainName:     aws.String("timed-out.example.com"),
		Status:         acmtypes.CertificateStatusValidationTimedOut,
	})
	if ok {
		t.Fatal("newACMCertificateResource() ok = true for VALIDATION_TIMED_OUT, want false")
	}
}
