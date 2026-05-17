// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

func TestCSMThreatsAgentRuleCreateResourceWithPolicy(t *testing.T) {
	generator := &CSMThreatsAgentRuleGenerator{}
	resource, err := generator.createResource("rule-123", "policy-abc")
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "rule-123" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "rule-123")
	}
	if resource.InstanceInfo.Type != "datadog_csm_threats_agent_rule" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_csm_threats_agent_rule")
	}
	if v := resource.InstanceState.Attributes["policy_id"]; v != "policy-abc" {
		t.Fatalf("policy_id = %q, want %q", v, "policy-abc")
	}
}

func TestCSMThreatsAgentRuleCreateResourceWithoutPolicy(t *testing.T) {
	generator := &CSMThreatsAgentRuleGenerator{}
	resource, err := generator.createResource("rule-123", "")
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "rule-123" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "rule-123")
	}
	if _, ok := resource.InstanceState.Attributes["policy_id"]; ok {
		t.Fatal("policy_id should not be set for unscoped rules")
	}
}

func TestCSMThreatsAgentRuleCreateResourceMissingID(t *testing.T) {
	generator := &CSMThreatsAgentRuleGenerator{}
	_, err := generator.createResource("", "policy-abc")
	if err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestParseCSMThreatsAgentRuleFilterID(t *testing.T) {
	tests := []struct {
		input      string
		wantPolicy string
		wantRule   string
	}{
		{"policy-id:rule-id", "policy-id", "rule-id"},
		{"a:b:c", "a", "b:c"},
		{"rule-id-only", "", "rule-id-only"},
	}
	for _, tt := range tests {
		policyID, ruleID := parseCSMThreatsAgentRuleFilterID(tt.input)
		if policyID != tt.wantPolicy || ruleID != tt.wantRule {
			t.Errorf("parseCSMThreatsAgentRuleFilterID(%q) = (%q, %q), want (%q, %q)",
				tt.input, policyID, ruleID, tt.wantPolicy, tt.wantRule)
		}
	}
}

func TestCSMThreatsPolicyCreateResource(t *testing.T) {
	data := datadogV2.NewCloudWorkloadSecurityAgentPolicyDataWithDefaults()
	data.SetId("policy-xyz-456")

	generator := &CSMThreatsPolicyGenerator{}
	resource, err := generator.createResource(*data)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "policy-xyz-456" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "policy-xyz-456")
	}
	if resource.InstanceInfo.Type != "datadog_csm_threats_policy" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_csm_threats_policy")
	}
}

func TestCSMThreatsPolicyCreateResourceMissingID(t *testing.T) {
	generator := &CSMThreatsPolicyGenerator{}
	_, err := generator.createResource(datadogV2.CloudWorkloadSecurityAgentPolicyData{})
	if err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestCSMThreatsPolicyCreateResources(t *testing.T) {
	first := datadogV2.NewCloudWorkloadSecurityAgentPolicyDataWithDefaults()
	first.SetId("policy-1")
	second := datadogV2.NewCloudWorkloadSecurityAgentPolicyDataWithDefaults()
	second.SetId("policy-2")

	generator := &CSMThreatsPolicyGenerator{}
	resources, err := generator.createResources([]datadogV2.CloudWorkloadSecurityAgentPolicyData{*first, *second})
	if err != nil {
		t.Fatalf("createResources returned error: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("resource count = %d, want %d", len(resources), 2)
	}
}
