// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"
)

func TestMskVpcConnectionName(t *testing.T) {
	testCases := []struct {
		arn          string
		expectedName string
	}{
		{
			arn:          "arn:aws:kafka:us-east-1:123456789012:vpc-connection/123456789012/my-connection/abc123-def456",
			expectedName: "my-connection-abc123-def456",
		},
		{
			arn:          "arn:aws:kafka:eu-west-1:987654321098:vpc-connection/987654321098/test-vpc-conn/xyz789",
			expectedName: "test-vpc-conn-xyz789",
		},
		{
			arn:          "no-slashes",
			expectedName: "no-slashes",
		},
	}

	for _, tc := range testCases {
		if got := mskVpcConnectionName(tc.arn); got != tc.expectedName {
			t.Errorf("mskVpcConnectionName(%q) = %q, want %q", tc.arn, got, tc.expectedName)
		}
	}
}

func TestMskClusterPolicyName(t *testing.T) {
	testCases := []struct {
		arn          string
		expectedName string
	}{
		{
			arn:          "arn:aws:kafka:us-east-1:123456789012:cluster/my-cluster/abc123-def456",
			expectedName: "my-cluster-policy",
		},
		{
			arn:          "arn:aws:kafka:eu-west-1:987654321098:cluster/production-kafka/xyz789",
			expectedName: "production-kafka-policy",
		},
		{
			arn:          "no-slashes",
			expectedName: "no-slashes",
		},
	}

	for _, tc := range testCases {
		if got := mskClusterPolicyName(tc.arn); got != tc.expectedName {
			t.Errorf("mskClusterPolicyName(%q) = %q, want %q", tc.arn, got, tc.expectedName)
		}
	}
}

func TestMskSecretName(t *testing.T) {
	testCases := []struct {
		secretArn    string
		expectedName string
	}{
		{
			secretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-AbCdEf",
			expectedName: "my-secret-AbCdEf",
		},
		{
			secretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:kafka-credentials-XyZ123",
			expectedName: "kafka-credentials-XyZ123",
		},
		{
			secretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-prod",
			expectedName: "my-secret-prod",
		},
		{
			secretArn:    "short:arn",
			expectedName: "short:arn",
		},
	}

	for _, tc := range testCases {
		if got := mskSecretName(tc.secretArn); got != tc.expectedName {
			t.Errorf("mskSecretName(%q) = %q, want %q", tc.secretArn, got, tc.expectedName)
		}
	}
}
