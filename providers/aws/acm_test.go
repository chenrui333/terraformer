// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

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
