// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

func TestLogsAccountPolicyTypes(t *testing.T) {
	want := []types.PolicyType{
		types.PolicyTypeDataProtectionPolicy,
		types.PolicyTypeSubscriptionFilterPolicy,
		types.PolicyTypeFieldIndexPolicy,
		types.PolicyTypeTransformerPolicy,
	}
	if len(logsAccountPolicyTypes) != len(want) {
		t.Fatalf("logsAccountPolicyTypes length = %d, want %d", len(logsAccountPolicyTypes), len(want))
	}
	for i, policyType := range logsAccountPolicyTypes {
		if policyType != want[i] {
			t.Fatalf("logsAccountPolicyTypes[%d] = %q, want %q", i, policyType, want[i])
		}
	}
}

func TestLogsResourceName(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "joins parts", parts: []string{"group", "filter"}, want: "group_filter"},
		{name: "omits empty parts", parts: []string{"", "group", "", "filter"}, want: "group_filter"},
		{name: "empty", parts: []string{"", ""}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := logsResourceName(tt.parts...); got != tt.want {
				t.Fatalf("logsResourceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLogsResourceNotFound(t *testing.T) {
	if !logsResourceNotFound(&types.ResourceNotFoundException{}) {
		t.Fatal("logsResourceNotFound() = false for ResourceNotFoundException, want true")
	}
	if logsResourceNotFound(errors.New("boom")) {
		t.Fatal("logsResourceNotFound() = true for generic error, want false")
	}
	if logsResourceNotFound(nil) {
		t.Fatal("logsResourceNotFound() = true for nil, want false")
	}
}

func TestLogsQueryDefinitionARN(t *testing.T) {
	tests := []struct {
		name              string
		region            string
		account           string
		queryDefinitionID string
		want              string
	}{
		{
			name:              "standard partition",
			region:            "us-east-1",
			account:           "123456789012",
			queryDefinitionID: "269951d7-6f75-496d-9d7b-6b7a5486bdbd",
			want:              "arn:aws:logs:us-east-1:123456789012:query-definition:269951d7-6f75-496d-9d7b-6b7a5486bdbd",
		},
		{
			name:              "china partition",
			region:            "cn-north-1",
			account:           "123456789012",
			queryDefinitionID: "query-id",
			want:              "arn:aws-cn:logs:cn-north-1:123456789012:query-definition:query-id",
		},
		{
			name:              "gov partition",
			region:            "us-gov-west-1",
			account:           "123456789012",
			queryDefinitionID: "query-id",
			want:              "arn:aws-us-gov:logs:us-gov-west-1:123456789012:query-definition:query-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := logsQueryDefinitionARN(tt.region, tt.account, tt.queryDefinitionID); got != tt.want {
				t.Fatalf("logsQueryDefinitionARN() = %q, want %q", got, tt.want)
			}
		})
	}
}
