// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	accessanalyzertypes "github.com/aws/aws-sdk-go-v2/service/accessanalyzer/types"
)

func TestAccessAnalyzerArchiveRuleResourceID(t *testing.T) {
	got := accessAnalyzerArchiveRuleResourceID("account-analyzer", "archive-public")
	want := "account-analyzer/archive-public"
	if got != want {
		t.Fatalf("archive rule resource ID = %q, want %q", got, want)
	}
}

func TestAccessAnalyzerArchiveRuleResource(t *testing.T) {
	resource := newAccessAnalyzerArchiveRuleResource("account-analyzer", "archive-public")

	if got, want := resource.InstanceState.ID, "account-analyzer/archive-public"; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceInfo.Type, accessAnalyzerArchiveRuleResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["analyzer_name"], "account-analyzer"; got != want {
		t.Fatalf("analyzer_name = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["rule_name"], "archive-public"; got != want {
		t.Fatalf("rule_name = %q, want %q", got, want)
	}
}

func TestAccessAnalyzerResourceNamesDoNotCollapseJoinedParts(t *testing.T) {
	left := newAccessAnalyzerArchiveRuleResource("a_b", "c")
	right := newAccessAnalyzerArchiveRuleResource("a", "b_c")
	if left.ResourceName == right.ResourceName {
		t.Fatalf("archive rule resource names collide: %q", left.ResourceName)
	}
}

func TestAccessAnalyzerResourceNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "resource not found", err: &accessanalyzertypes.ResourceNotFoundException{}, want: true},
		{name: "wrapped resource not found", err: errors.Join(errors.New("lookup failed"), &accessanalyzertypes.ResourceNotFoundException{}), want: true},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := accessAnalyzerResourceNotFound(tt.err); got != tt.want {
				t.Fatalf("accessAnalyzerResourceNotFound(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}
