// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestCloudWatchEventRuleImportID(t *testing.T) {
	tests := []struct {
		name         string
		eventBusName string
		ruleName     string
		want         string
	}{
		{name: "default bus", eventBusName: defaultEventBusName, ruleName: "daily", want: "daily"},
		{name: "empty bus", eventBusName: "", ruleName: "daily", want: "daily"},
		{name: "custom bus", eventBusName: "orders", ruleName: "daily", want: "orders/daily"},
		{name: "partner bus", eventBusName: "aws.partner/example.com/source", ruleName: "daily", want: "aws.partner/example.com/source/daily"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cloudwatchEventRuleImportID(tt.eventBusName, tt.ruleName); got != tt.want {
				t.Fatalf("cloudwatchEventRuleImportID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCloudWatchEventTargetImportID(t *testing.T) {
	tests := []struct {
		name         string
		eventBusName string
		ruleName     string
		targetID     string
		want         string
	}{
		{name: "default bus", eventBusName: defaultEventBusName, ruleName: "daily", targetID: "target", want: "daily/target"},
		{name: "empty bus", eventBusName: "", ruleName: "daily", targetID: "target", want: "daily/target"},
		{name: "custom bus", eventBusName: "orders", ruleName: "daily", targetID: "target", want: "orders/daily/target"},
		{name: "partner bus", eventBusName: "aws.partner/example.com/source", ruleName: "daily", targetID: "target", want: "aws.partner/example.com/source/daily/target"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cloudwatchEventTargetImportID(tt.eventBusName, tt.ruleName, tt.targetID); got != tt.want {
				t.Fatalf("cloudwatchEventTargetImportID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCloudWatchEventResourceName(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "default bus omitted", parts: []string{defaultEventBusName, "daily"}, want: "daily"},
		{name: "default rule name preserved", parts: []string{defaultEventBusName, defaultEventBusName}, want: "default"},
		{name: "default target ID preserved", parts: []string{defaultEventBusName, "daily", defaultEventBusName}, want: "daily_default"},
		{name: "custom bus included", parts: []string{"orders", "daily"}, want: "orders_daily"},
		{name: "empty parts omitted", parts: []string{"", "orders", "", "daily"}, want: "orders_daily"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cloudwatchEventResourceName(tt.parts...); got != tt.want {
				t.Fatalf("cloudwatchEventResourceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCloudWatchCompositeAlarmResourceType(t *testing.T) {
	alarmName := "high-level-alarm"
	resource := terraformutils.NewSimpleResource(
		alarmName, alarmName, "aws_cloudwatch_composite_alarm", "aws", cloudwatchAllowEmptyValues)

	if got := resource.InstanceState.ID; got != alarmName {
		t.Fatalf("resource ID = %q, want %q", got, alarmName)
	}
	if got := resource.InstanceInfo.Type; got != "aws_cloudwatch_composite_alarm" {
		t.Fatalf("resource type = %q, want %q", got, "aws_cloudwatch_composite_alarm")
	}
}
