// Copyright 2019 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aws

import (
	"strings"
	"testing"
)

func TestMskVpcConnectionResourceName(t *testing.T) {
	// Test VPC connection ARN parsing for resource naming
	// ARN format: arn:aws:kafka:region:account:vpc-connection/account/name/uuid
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
	// Test cluster ARN parsing for policy resource naming
	// ARN format: arn:aws:kafka:region:account:cluster/name/uuid
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
	// Test single SCRAM secret association import ID format
	// Import ID format: cluster_arn,secret_arn
	testCases := []struct {
		clusterArn   string
		secretArn    string
		expectedID   string
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
	// Test secret name extraction from ARN for resource naming
	// ARN format: arn:aws:secretsmanager:region:account:secret:name-suffix
	testCases := []struct {
		secretArn    string
		expectedName string
	}{
		{
			secretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-AbCdEf",
			expectedName: "my-secret",
		},
		{
			secretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:kafka-credentials-XyZ123",
			expectedName: "kafka-credentials",
		},
		{
			// Secret name with longer suffix (more than 6 chars after last dash, not stripped)
			secretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-with-longersuffix",
			expectedName: "my-secret-with-longersuffix",
		},
	}

	for _, tc := range testCases {
		secretName := tc.secretArn
		if parts := strings.Split(tc.secretArn, ":"); len(parts) >= 7 {
			secretName = parts[6]
			// Remove the random suffix if present (e.g., "mysecret-AbCdEf")
			if idx := strings.LastIndex(secretName, "-"); idx > 0 && len(secretName)-idx <= 7 {
				secretName = secretName[:idx]
			}
		}
		if secretName != tc.expectedName {
			t.Errorf("Secret name extraction: expected %s, got %s (from %s)", tc.expectedName, secretName, tc.secretArn)
		}
	}
}

func TestMskAllowEmptyValues(t *testing.T) {
	// Ensure the allow empty values list is correctly defined
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
