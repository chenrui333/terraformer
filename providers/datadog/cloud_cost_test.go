// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

func TestAwsCURConfigCreateResource(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		acctID   string
		wantID   string
		wantName string
		wantType string
	}{
		{
			name:     "uses account ID in resource name",
			id:       "12345",
			acctID:   "123456789012",
			wantID:   "12345",
			wantName: "tfer--aws_cur_config_123456789012",
			wantType: "datadog_aws_cur_config",
		},
		{
			name:     "falls back to id in resource name",
			id:       "67890",
			acctID:   "",
			wantID:   "67890",
			wantName: "tfer--aws_cur_config_67890",
			wantType: "datadog_aws_cur_config",
		},
	}

	generator := AwsCURConfigGenerator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := datadogV2.NewAwsCURConfigWithDefaults()
			config.SetId(tt.id)
			attrs := datadogV2.NewAwsCURConfigAttributesWithDefaults()
			if tt.acctID != "" {
				attrs.SetAccountId(tt.acctID)
			}
			config.SetAttributes(*attrs)

			resource := generator.createResource(*config)
			if resource.InstanceState.ID != tt.wantID {
				t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, tt.wantID)
			}
			if resource.ResourceName != tt.wantName {
				t.Fatalf("resource name = %q, want %q", resource.ResourceName, tt.wantName)
			}
			if resource.InstanceInfo.Type != tt.wantType {
				t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, tt.wantType)
			}
		})
	}
}

func TestAzureUCConfigCreateResource(t *testing.T) {
	generator := AzureUCConfigGenerator{}

	config := datadogV2.NewAzureUCConfigPairWithDefaults()
	config.SetId("abc-123")

	resource := generator.createResource(*config)
	if resource.InstanceState.ID != "abc-123" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "abc-123")
	}
	if resource.ResourceName != "tfer--azure_uc_config_abc-123" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--azure_uc_config_abc-123")
	}
	if resource.InstanceInfo.Type != "datadog_azure_uc_config" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_azure_uc_config")
	}
}

func TestGCPUCConfigCreateResource(t *testing.T) {
	generator := GCPUCConfigGenerator{}

	config := datadogV2.NewGCPUsageCostConfigWithDefaults()
	config.SetId("gcp-456")

	resource := generator.createResource(*config)
	if resource.InstanceState.ID != "gcp-456" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "gcp-456")
	}
	if resource.ResourceName != "tfer--gcp_uc_config_gcp-456" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--gcp_uc_config_gcp-456")
	}
	if resource.InstanceInfo.Type != "datadog_gcp_uc_config" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_gcp_uc_config")
	}
}

func TestCostBudgetCreateResource(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		budgetName string
		wantID     string
		wantName   string
	}{
		{
			name:       "uses budget name",
			id:         "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			budgetName: "Production-Budget",
			wantID:     "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			wantName:   "tfer--cost_budget_Production-Budget",
		},
		{
			name:       "falls back to id",
			id:         "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			budgetName: "",
			wantID:     "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			wantName:   "tfer--cost_budget_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		},
	}

	generator := CostBudgetGenerator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budget := datadogV2.NewBudget("budget")
			budget.SetId(tt.id)
			attrs := datadogV2.NewBudgetAttributesWithDefaults()
			if tt.budgetName != "" {
				attrs.SetName(tt.budgetName)
			}
			budget.SetAttributes(*attrs)

			resource := generator.createResource(*budget)
			if resource.InstanceState.ID != tt.wantID {
				t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, tt.wantID)
			}
			if resource.ResourceName != tt.wantName {
				t.Fatalf("resource name = %q, want %q", resource.ResourceName, tt.wantName)
			}
			if resource.InstanceInfo.Type != "datadog_cost_budget" {
				t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_cost_budget")
			}
		})
	}
}

func TestCustomAllocationRuleCreateResource(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		ruleName string
		wantName string
	}{
		{
			name:     "uses rule name",
			id:       "99",
			ruleName: "my-allocation-rule",
			wantName: "tfer--custom_allocation_rule_my-allocation-rule",
		},
		{
			name:     "falls back to id",
			id:       "99",
			ruleName: "",
			wantName: "tfer--custom_allocation_rule_99",
		},
	}

	generator := CustomAllocationRuleGenerator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := datadogV2.NewArbitraryRuleResponseDataWithDefaults()
			rule.SetId(tt.id)
			attrs := datadogV2.NewArbitraryRuleResponseDataAttributesWithDefaults()
			if tt.ruleName != "" {
				attrs.SetRuleName(tt.ruleName)
			}
			rule.SetAttributes(*attrs)

			resource := generator.createResource(*rule)
			if resource.InstanceState.ID != tt.id {
				t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, tt.id)
			}
			if resource.ResourceName != tt.wantName {
				t.Fatalf("resource name = %q, want %q", resource.ResourceName, tt.wantName)
			}
			if resource.InstanceInfo.Type != "datadog_custom_allocation_rule" {
				t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_custom_allocation_rule")
			}
		})
	}
}

func TestTagPipelineRulesetCreateResource(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		rulesetName string
		wantName    string
	}{
		{
			name:        "uses ruleset name",
			id:          "rs-abc123",
			rulesetName: "standardize-tags",
			wantName:    "tfer--tag_pipeline_ruleset_standardize-tags",
		},
		{
			name:        "falls back to id",
			id:          "rs-abc123",
			rulesetName: "",
			wantName:    "tfer--tag_pipeline_ruleset_rs-abc123",
		},
	}

	generator := TagPipelineRulesetGenerator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ruleset := datadogV2.NewRulesetRespDataWithDefaults()
			ruleset.SetId(tt.id)
			attrs := datadogV2.NewRulesetRespDataAttributesWithDefaults()
			if tt.rulesetName != "" {
				attrs.SetName(tt.rulesetName)
			}
			ruleset.SetAttributes(*attrs)

			resource := generator.createResource(*ruleset)
			if resource.InstanceState.ID != tt.id {
				t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, tt.id)
			}
			if resource.ResourceName != tt.wantName {
				t.Fatalf("resource name = %q, want %q", resource.ResourceName, tt.wantName)
			}
			if resource.InstanceInfo.Type != "datadog_tag_pipeline_ruleset" {
				t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_tag_pipeline_ruleset")
			}
		})
	}
}
