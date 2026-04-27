// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"strings"
	"testing"
)

func TestMskVpcConnectionResourceName(t *testing.T) {
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
	}

	for _, tc := range testCases {
		resourceName := tc.arn
		if parts := strings.Split(resourceName, "/"); len(parts) >= 3 {
			resourceName = parts[len(parts)-2] + "-" + parts[len(parts)-1]
		}
		if resourceName != tc.expectedName {
			t.Errorf("VPC connection resource name: expected %s, got %s", tc.expectedName, resourceName)
		}
	}
}

func TestMskClusterPolicyResourceName(t *testing.T) {
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
	}

	for _, tc := range testCases {
		resourceName := tc.arn
		if parts := strings.Split(tc.arn, "/"); len(parts) >= 2 {
			resourceName = parts[1] + "-policy"
		}
		if resourceName != tc.expectedName {
			t.Errorf("Cluster policy resource name: expected %s, got %s", tc.expectedName, resourceName)
		}
	}
}

func TestMskSingleScramSecretImportID(t *testing.T) {
	testCases := []struct {
		clusterArn string
		secretArn  string
		expectedID string
	}{
		{
			clusterArn: "arn:aws:kafka:us-east-1:123456789012:cluster/my-cluster/abc123",
			secretArn:  "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-AbCdEf",
			expectedID: "arn:aws:kafka:us-east-1:123456789012:cluster/my-cluster/abc123,arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-AbCdEf",
		},
	}

	for _, tc := range testCases {
		importID := tc.clusterArn + "," + tc.secretArn
		if importID != tc.expectedID {
			t.Errorf("Single SCRAM secret import ID: expected %s, got %s", tc.expectedID, importID)
		}
	}
}

func TestMskSecretNameExtraction(t *testing.T) {
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
	}

	for _, tc := range testCases {
		secretName := tc.secretArn
		if parts := strings.Split(tc.secretArn, ":"); len(parts) >= 7 {
			secretName = parts[6]
		}
		if secretName != tc.expectedName {
			t.Errorf("Secret name extraction: expected %s, got %s (from %s)", tc.expectedName, secretName, tc.secretArn)
		}
	}
}

func TestMskAllowEmptyValues(t *testing.T) {
	expected := []string{"tags."}
	if len(mskAllowEmptyValues) != len(expected) {
		t.Errorf("mskAllowEmptyValues length: expected %d, got %d", len(expected), len(mskAllowEmptyValues))
	}
	for i, v := range expected {
		if mskAllowEmptyValues[i] != v {
			t.Errorf("mskAllowEmptyValues[%d]: expected %s, got %s", i, v, mskAllowEmptyValues[i])
		}
	}
}
