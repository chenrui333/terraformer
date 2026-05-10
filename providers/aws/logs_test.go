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

func TestNewLogsDestinationPolicyResource(t *testing.T) {
	destinationName := "central-logs"
	accessPolicy := "{}"

	tests := []struct {
		name        string
		destination types.Destination
		wantOK      bool
	}{
		{
			name: "destination with access policy",
			destination: types.Destination{
				AccessPolicy:    &accessPolicy,
				DestinationName: &destinationName,
			},
			wantOK: true,
		},
		{
			name: "destination without access policy is skipped",
			destination: types.Destination{
				DestinationName: &destinationName,
			},
		},
		{
			name: "destination without name is skipped",
			destination: types.Destination{
				AccessPolicy: &accessPolicy,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, ok := newLogsDestinationPolicyResource(tt.destination)
			if ok != tt.wantOK {
				t.Fatalf("newLogsDestinationPolicyResource() ok = %t, want %t", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got := resource.InstanceState.ID; got != destinationName {
				t.Fatalf("resource ID = %q, want %q", got, destinationName)
			}
			if got := resource.InstanceInfo.Type; got != logsDestinationPolicyResourceType {
				t.Fatalf("resource type = %q, want %q", got, logsDestinationPolicyResourceType)
			}
			if got := resource.InstanceState.Attributes["destination_name"]; got != destinationName {
				t.Fatalf("destination_name = %q, want %q", got, destinationName)
			}
		})
	}
}

func TestNewLogsIndexPolicyResource(t *testing.T) {
	logGroupName := "/aws/lambda/example"
	policyDocument := "{}"

	tests := []struct {
		name         string
		logGroupName string
		policy       types.IndexPolicy
		wantOK       bool
	}{
		{
			name:         "log group policy",
			logGroupName: logGroupName,
			policy: types.IndexPolicy{
				PolicyDocument: &policyDocument,
				Source:         types.IndexSourceLogGroup,
			},
			wantOK: true,
		},
		{
			name:         "policy without source is accepted",
			logGroupName: logGroupName,
			policy: types.IndexPolicy{
				PolicyDocument: &policyDocument,
			},
			wantOK: true,
		},
		{
			name:         "account policy is skipped",
			logGroupName: logGroupName,
			policy: types.IndexPolicy{
				PolicyDocument: &policyDocument,
				Source:         types.IndexSourceAccount,
			},
		},
		{
			name: "empty log group is skipped",
			policy: types.IndexPolicy{
				PolicyDocument: &policyDocument,
				Source:         types.IndexSourceLogGroup,
			},
		},
		{
			name:         "policy without document is skipped",
			logGroupName: logGroupName,
			policy: types.IndexPolicy{
				Source: types.IndexSourceLogGroup,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, ok := newLogsIndexPolicyResource(tt.logGroupName, tt.policy)
			if ok != tt.wantOK {
				t.Fatalf("newLogsIndexPolicyResource() ok = %t, want %t", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got := resource.InstanceState.ID; got != tt.logGroupName {
				t.Fatalf("resource ID = %q, want %q", got, tt.logGroupName)
			}
			if got := resource.InstanceInfo.Type; got != logsIndexPolicyResourceType {
				t.Fatalf("resource type = %q, want %q", got, logsIndexPolicyResourceType)
			}
			if got := resource.InstanceState.Attributes["log_group_name"]; got != tt.logGroupName {
				t.Fatalf("log_group_name = %q, want %q", got, tt.logGroupName)
			}
		})
	}
}
