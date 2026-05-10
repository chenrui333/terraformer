// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewSchedulerScheduleGroupResource(t *testing.T) {
	tests := []struct {
		name      string
		groupName string
		wantOK    bool
		wantID    string
	}{
		{name: "custom group", groupName: "orders", wantOK: true, wantID: "orders"},
		{name: "default group skipped", groupName: defaultSchedulerScheduleGroupName, wantOK: false},
		{name: "empty group skipped", groupName: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, ok := newSchedulerScheduleGroupResource(tt.groupName)
			if ok != tt.wantOK {
				t.Fatalf("newSchedulerScheduleGroupResource() ok = %t, want %t", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if resource.InstanceState.ID != tt.wantID {
				t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, tt.wantID)
			}
			if resource.InstanceInfo.Type != "aws_scheduler_schedule_group" {
				t.Fatalf("resource type = %q, want aws_scheduler_schedule_group", resource.InstanceInfo.Type)
			}
		})
	}
}

func TestNewSchedulerScheduleResource(t *testing.T) {
	tests := []struct {
		name         string
		groupName    string
		scheduleName string
		wantOK       bool
		wantID       string
	}{
		{name: "default group schedule", groupName: defaultSchedulerScheduleGroupName, scheduleName: "daily", wantOK: true, wantID: "default/daily"},
		{name: "custom group schedule", groupName: "orders", scheduleName: "daily", wantOK: true, wantID: "orders/daily"},
		{name: "empty group skipped", groupName: "", scheduleName: "daily", wantOK: false},
		{name: "empty schedule skipped", groupName: "orders", scheduleName: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, ok := newSchedulerScheduleResource(tt.groupName, tt.scheduleName)
			if ok != tt.wantOK {
				t.Fatalf("newSchedulerScheduleResource() ok = %t, want %t", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if resource.InstanceState.ID != tt.wantID {
				t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, tt.wantID)
			}
			if resource.InstanceInfo.Type != "aws_scheduler_schedule" {
				t.Fatalf("resource type = %q, want aws_scheduler_schedule", resource.InstanceInfo.Type)
			}
		})
	}
}

func TestSchedulerScheduleResourceNamesPreserveParentScope(t *testing.T) {
	resourceA, ok := newSchedulerScheduleResource("a_b", "c")
	if !ok {
		t.Fatal("newSchedulerScheduleResource() should create resourceA")
	}
	resourceB, ok := newSchedulerScheduleResource("a", "b_c")
	if !ok {
		t.Fatal("newSchedulerScheduleResource() should create resourceB")
	}
	if resourceA.ResourceName == resourceB.ResourceName {
		t.Fatalf("resource names collide: %q", resourceA.ResourceName)
	}
}

func TestSchedulerInitialCleanupHonorsTypedFilters(t *testing.T) {
	group, ok := newSchedulerScheduleGroupResource("orders")
	if !ok {
		t.Fatal("newSchedulerScheduleGroupResource() should create group")
	}
	schedule, ok := newSchedulerScheduleResource("orders", "daily")
	if !ok {
		t.Fatal("newSchedulerScheduleResource() should create schedule")
	}
	g := SchedulerGenerator{}
	g.Resources = []terraformutils.Resource{group, schedule}
	g.Filter = []terraformutils.ResourceFilter{{
		ServiceName:      schedulerScheduleResourceType,
		FieldPath:        "id",
		AcceptableValues: []string{"orders/daily"},
	}}

	g.InitialCleanup()

	if len(g.Resources) != 1 {
		t.Fatalf("InitialCleanup() resources len = %d, want 1", len(g.Resources))
	}
	if got := g.Resources[0].InstanceInfo.Type; got != "aws_scheduler_schedule" {
		t.Fatalf("InitialCleanup() kept resource type = %q, want aws_scheduler_schedule", got)
	}
}
