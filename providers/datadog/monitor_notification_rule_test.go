// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

func TestMonitorNotificationRuleCreateResource(t *testing.T) {
	monitorNotificationRule := datadogV2.NewMonitorNotificationRuleDataWithDefaults()
	monitorNotificationRule.SetId("rule-id")

	generator := &MonitorNotificationRuleGenerator{}
	resource, err := generator.createResource(*monitorNotificationRule)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "rule-id" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "rule-id")
	}
	if resource.ResourceName != "tfer--monitor_notification_rule_rule-id" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--monitor_notification_rule_rule-id")
	}
	if resource.InstanceInfo.Type != "datadog_monitor_notification_rule" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_monitor_notification_rule")
	}
}

func TestMonitorNotificationRuleCreateResourceMissingID(t *testing.T) {
	generator := &MonitorNotificationRuleGenerator{}
	_, err := generator.createResource(datadogV2.MonitorNotificationRuleData{})
	if err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestMonitorNotificationRuleCreateResources(t *testing.T) {
	firstRule := datadogV2.NewMonitorNotificationRuleDataWithDefaults()
	firstRule.SetId("rule-1")
	secondRule := datadogV2.NewMonitorNotificationRuleDataWithDefaults()
	secondRule.SetId("rule-2")

	generator := &MonitorNotificationRuleGenerator{}
	resources, err := generator.createResources([]datadogV2.MonitorNotificationRuleData{
		*firstRule,
		*secondRule,
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

func TestMonitorNotificationRulesFromRawData(t *testing.T) {
	rules := monitorNotificationRulesFromRawData([]interface{}{
		map[string]interface{}{
			"id":   "rule-1",
			"type": "monitor_notification_rule",
		},
		map[string]interface{}{
			"id":   "rule-2",
			"type": "monitor_notification_rule",
		},
		map[string]interface{}{
			"type": "monitor_notification_rule",
		},
	})

	if len(rules) != 2 {
		t.Fatalf("rule count = %d, want %d", len(rules), 2)
	}
	if rules[0].GetId() != "rule-1" {
		t.Fatalf("first rule ID = %q, want %q", rules[0].GetId(), "rule-1")
	}
	if rules[1].GetId() != "rule-2" {
		t.Fatalf("second rule ID = %q, want %q", rules[1].GetId(), "rule-2")
	}
}
