// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"
)

func TestMskVpcConnectionName(t *testing.T) {
	testCases := []struct {
		name         string
		arn          string
		expectedName string
	}{
		{
			name:         "standard vpc connection arn",
			arn:          "arn:aws:kafka:us-east-1:123456789012:vpc-connection/123456789012/my-connection/abc123-def456",
			expectedName: "my-connection-abc123-def456",
		},
		{
			name:         "different region and account",
			arn:          "arn:aws:kafka:eu-west-1:987654321098:vpc-connection/987654321098/test-vpc-conn/xyz789",
			expectedName: "test-vpc-conn-xyz789",
		},
		{
			name:         "no slashes fallback",
			arn:          "no-slashes",
			expectedName: "no-slashes",
		},
		{
			name:         "empty string",
			arn:          "",
			expectedName: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mskVpcConnectionName(tc.arn); got != tc.expectedName {
				t.Errorf("mskVpcConnectionName(%q) = %q, want %q", tc.arn, got, tc.expectedName)
			}
		})
	}
}

func TestMskClusterPolicyName(t *testing.T) {
	testCases := []struct {
		name         string
		arn          string
		expectedName string
	}{
		{
			name:         "standard cluster arn",
			arn:          "arn:aws:kafka:us-east-1:123456789012:cluster/my-cluster/abc123-def456",
			expectedName: "my-cluster-policy",
		},
		{
			name:         "different region and account",
			arn:          "arn:aws:kafka:eu-west-1:987654321098:cluster/production-kafka/xyz789",
			expectedName: "production-kafka-policy",
		},
		{
			name:         "no slashes fallback",
			arn:          "no-slashes",
			expectedName: "no-slashes",
		},
		{
			name:         "empty string",
			arn:          "",
			expectedName: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mskClusterPolicyName(tc.arn); got != tc.expectedName {
				t.Errorf("mskClusterPolicyName(%q) = %q, want %q", tc.arn, got, tc.expectedName)
			}
		})
	}
}

func TestMskSecretName(t *testing.T) {
	testCases := []struct {
		name         string
		secretArn    string
		expectedName string
	}{
		{
			name:         "standard secret arn",
			secretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-AbCdEf",
			expectedName: "my-secret-AbCdEf",
		},
		{
			name:         "secret with different suffix",
			secretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:kafka-credentials-XyZ123",
			expectedName: "kafka-credentials-XyZ123",
		},
		{
			name:         "secret name ending in -prod",
			secretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-prod",
			expectedName: "my-secret-prod",
		},
		{
			name:         "secret name with colons",
			secretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:path/to:name:with:colons",
			expectedName: "path/to:name:with:colons",
		},
		{
			name:         "short arn fallback",
			secretArn:    "short:arn",
			expectedName: "short:arn",
		},
		{
			name:         "empty string",
			secretArn:    "",
			expectedName: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mskSecretName(tc.secretArn); got != tc.expectedName {
				t.Errorf("mskSecretName(%q) = %q, want %q", tc.secretArn, got, tc.expectedName)
			}
		})
	}
}
