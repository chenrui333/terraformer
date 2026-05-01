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

func TestLogsResourcePolicyResource(t *testing.T) {
	policyName := "account-policy"
	policyDocument := "{}"
	resourceArn := "arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/example"

	tests := []struct {
		name           string
		policy         types.ResourcePolicy
		wantID         string
		wantName       string
		wantAttributes map[string]string
	}{
		{
			name: "account scoped policy uses policy name",
			policy: types.ResourcePolicy{
				PolicyDocument: &policyDocument,
				PolicyName:     &policyName,
				PolicyScope:    types.PolicyScopeAccount,
			},
			wantID:   policyName,
			wantName: policyName,
			wantAttributes: map[string]string{
				"policy_name": policyName,
			},
		},
		{
			name: "resource scoped policy uses resource ARN",
			policy: types.ResourcePolicy{
				PolicyDocument: &policyDocument,
				PolicyName:     &policyName,
				PolicyScope:    types.PolicyScopeResource,
				ResourceArn:    &resourceArn,
			},
			wantID:   resourceArn,
			wantName: logsResourceName(policyName, resourceArn),
			wantAttributes: map[string]string{
				"policy_scope": string(types.PolicyScopeResource),
				"resource_arn": resourceArn,
			},
		},
		{
			name: "resource scoped policy without ARN is skipped",
			policy: types.ResourcePolicy{
				PolicyDocument: &policyDocument,
				PolicyName:     &policyName,
				PolicyScope:    types.PolicyScopeResource,
			},
		},
		{
			name: "account scoped policy without name is skipped",
			policy: types.ResourcePolicy{
				PolicyDocument: &policyDocument,
				PolicyScope:    types.PolicyScopeAccount,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotName, gotAttributes := logsResourcePolicyResource(tt.policy)
			if gotID != tt.wantID {
				t.Fatalf("logsResourcePolicyResource() id = %q, want %q", gotID, tt.wantID)
			}
			if gotName != tt.wantName {
				t.Fatalf("logsResourcePolicyResource() name = %q, want %q", gotName, tt.wantName)
			}
			if len(gotAttributes) != len(tt.wantAttributes) {
				t.Fatalf("logsResourcePolicyResource() attributes = %#v, want %#v", gotAttributes, tt.wantAttributes)
			}
			for key, want := range tt.wantAttributes {
				if gotAttributes[key] != want {
					t.Fatalf("logsResourcePolicyResource() attributes[%q] = %q, want %q", key, gotAttributes[key], want)
				}
			}
		})
	}
}
