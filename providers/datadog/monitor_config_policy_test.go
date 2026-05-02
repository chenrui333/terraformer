// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

func TestMonitorConfigPolicyCreateResource(t *testing.T) {
	monitorConfigPolicy := datadogV2.NewMonitorConfigPolicyResponseDataWithDefaults()
	monitorConfigPolicy.SetId("policy-id")

	generator := &MonitorConfigPolicyGenerator{}
	resource, err := generator.createResource(*monitorConfigPolicy)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "policy-id" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "policy-id")
	}
	if resource.ResourceName != "tfer--monitor_config_policy_policy-id" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--monitor_config_policy_policy-id")
	}
	if resource.InstanceInfo.Type != "datadog_monitor_config_policy" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_monitor_config_policy")
	}
}

func TestMonitorConfigPolicyCreateResourceMissingID(t *testing.T) {
	generator := &MonitorConfigPolicyGenerator{}
	_, err := generator.createResource(datadogV2.MonitorConfigPolicyResponseData{})
	if err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestMonitorConfigPolicyCreateResources(t *testing.T) {
	firstPolicy := datadogV2.NewMonitorConfigPolicyResponseDataWithDefaults()
	firstPolicy.SetId("policy-1")
	secondPolicy := datadogV2.NewMonitorConfigPolicyResponseDataWithDefaults()
	secondPolicy.SetId("policy-2")

	generator := &MonitorConfigPolicyGenerator{}
	resources, err := generator.createResources([]datadogV2.MonitorConfigPolicyResponseData{
		*firstPolicy,
		*secondPolicy,
	})
	if err != nil {
		t.Fatalf("createResources returned error: %v", err)
	}

	if len(resources) != 2 {
		t.Fatalf("resource count = %d, want %d", len(resources), 2)
	}
	if resources[0].ResourceName == resources[1].ResourceName {
		t.Fatalf("resource names should be unique, got %q", resources[0].ResourceName)
	}
}

func TestMonitorConfigPoliciesFromRawData(t *testing.T) {
	policies := monitorConfigPoliciesFromRawData([]interface{}{
		map[string]interface{}{
			"id":   "policy-1",
			"type": "monitor_config_policy",
		},
		map[string]interface{}{
			"id":   "policy-2",
			"type": "monitor_config_policy",
		},
		map[string]interface{}{
			"type": "monitor_config_policy",
		},
	})

	if len(policies) != 2 {
		t.Fatalf("policy count = %d, want %d", len(policies), 2)
	}
	if policies[0].GetId() != "policy-1" {
		t.Fatalf("first policy ID = %q, want %q", policies[0].GetId(), "policy-1")
	}
	if policies[1].GetId() != "policy-2" {
		t.Fatalf("second policy ID = %q, want %q", policies[1].GetId(), "policy-2")
	}
}
