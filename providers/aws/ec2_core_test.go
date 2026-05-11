// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewEC2PlacementGroupResource(t *testing.T) {
	resource, ok := newEC2PlacementGroupResource(types.PlacementGroup{
		GroupName: aws.String("cluster-a"),
		State:     types.PlacementGroupStateAvailable,
	})
	if !ok {
		t.Fatal("newEC2PlacementGroupResource() ok = false, want true")
	}
	if got := resource.InstanceInfo.Type; got != ec2PlacementGroupResourceType {
		t.Fatalf("resource type = %q, want %q", got, ec2PlacementGroupResourceType)
	}
	if got := resource.InstanceState.ID; got != "cluster-a" {
		t.Fatalf("resource ID = %q, want cluster-a", got)
	}

	tests := []struct {
		name           string
		placementGroup types.PlacementGroup
	}{
		{name: "empty name", placementGroup: types.PlacementGroup{State: types.PlacementGroupStateAvailable}},
		{name: "deleted", placementGroup: types.PlacementGroup{GroupName: aws.String("cluster-a"), State: types.PlacementGroupStateDeleted}},
		{name: "aws managed", placementGroup: types.PlacementGroup{
			GroupName: aws.String("cluster-a"),
			State:     types.PlacementGroupStateAvailable,
			Operator:  &types.OperatorResponse{Managed: aws.Bool(true)},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := newEC2PlacementGroupResource(tt.placementGroup); ok {
				t.Fatal("newEC2PlacementGroupResource() ok = true, want false")
			}
		})
	}
}

func TestNewEC2InstanceConnectEndpointResource(t *testing.T) {
	resource, ok := newEC2InstanceConnectEndpointResource(types.Ec2InstanceConnectEndpoint{
		InstanceConnectEndpointId: aws.String("eice-123"),
		State:                     types.Ec2InstanceConnectEndpointStateCreateComplete,
		SubnetId:                  aws.String("subnet-123"),
	})
	if !ok {
		t.Fatal("newEC2InstanceConnectEndpointResource() ok = false, want true")
	}
	if got := resource.InstanceInfo.Type; got != ec2InstanceConnectEndpointResourceType {
		t.Fatalf("resource type = %q, want %q", got, ec2InstanceConnectEndpointResourceType)
	}
	if got := resource.InstanceState.ID; got != "eice-123" {
		t.Fatalf("resource ID = %q, want eice-123", got)
	}

	if _, ok := newEC2InstanceConnectEndpointResource(types.Ec2InstanceConnectEndpoint{
		InstanceConnectEndpointId: aws.String("eice-123"),
		State:                     types.Ec2InstanceConnectEndpointStateDeleteComplete,
	}); ok {
		t.Fatal("delete-complete instance connect endpoint should be skipped")
	}
	if _, ok := newEC2InstanceConnectEndpointResource(types.Ec2InstanceConnectEndpoint{
		State: types.Ec2InstanceConnectEndpointStateCreateComplete,
	}); ok {
		t.Fatal("endpoint with empty ID should be skipped")
	}
}

func TestNewEC2CapacityReservationResource(t *testing.T) {
	resource, ok := newEC2CapacityReservationResource(types.CapacityReservation{
		AvailabilityZone:      aws.String("us-east-1a"),
		CapacityReservationId: aws.String("cr-123"),
		State:                 types.CapacityReservationStateActive,
	})
	if !ok {
		t.Fatal("newEC2CapacityReservationResource() ok = false, want true")
	}
	if got := resource.InstanceInfo.Type; got != ec2CapacityReservationResourceType {
		t.Fatalf("resource type = %q, want %q", got, ec2CapacityReservationResourceType)
	}
	if got := resource.InstanceState.ID; got != "cr-123" {
		t.Fatalf("resource ID = %q, want cr-123", got)
	}

	if _, ok := newEC2CapacityReservationResource(types.CapacityReservation{
		CapacityReservationId: aws.String("cr-123"),
		State:                 types.CapacityReservationStateCancelled,
	}); ok {
		t.Fatal("cancelled capacity reservation should be skipped")
	}
	if _, ok := newEC2CapacityReservationResource(types.CapacityReservation{
		CapacityReservationId: aws.String("cr-123"),
		ReservationType:       types.CapacityReservationTypeCapacityBlock,
		State:                 types.CapacityReservationStateActive,
	}); ok {
		t.Fatal("capacity block reservation should be skipped")
	}
	if _, ok := newEC2CapacityReservationResource(types.CapacityReservation{
		State: types.CapacityReservationStateActive,
	}); ok {
		t.Fatal("capacity reservation with empty ID should be skipped")
	}
}

func TestNewEC2HostResource(t *testing.T) {
	resource, ok := newEC2HostResource(types.Host{
		AvailabilityZone: aws.String("us-east-1a"),
		HostId:           aws.String("h-123"),
		HostProperties:   &types.HostProperties{InstanceFamily: aws.String("m5")},
		State:            types.AllocationStateAvailable,
	})
	if !ok {
		t.Fatal("newEC2HostResource() ok = false, want true")
	}
	if got := resource.InstanceInfo.Type; got != ec2HostResourceType {
		t.Fatalf("resource type = %q, want %q", got, ec2HostResourceType)
	}
	if got := resource.InstanceState.ID; got != "h-123" {
		t.Fatalf("resource ID = %q, want h-123", got)
	}
	if len(resource.IgnoreKeys) != 0 {
		t.Fatalf("IgnoreKeys = %v, want empty for family-only host", resource.IgnoreKeys)
	}

	resource, ok = newEC2HostResource(types.Host{
		AvailabilityZone: aws.String("us-east-1a"),
		HostId:           aws.String("h-456"),
		HostProperties: &types.HostProperties{
			InstanceFamily: aws.String("m5"),
			InstanceType:   aws.String("m5.large"),
		},
		State: types.AllocationStateAvailable,
	})
	if !ok {
		t.Fatal("newEC2HostResource(instance type) ok = false, want true")
	}
	if got, want := resource.IgnoreKeys, []string{"^instance_family$"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("IgnoreKeys = %v, want %v", got, want)
	}

	if _, ok := newEC2HostResource(types.Host{
		HostId: aws.String("h-123"),
		State:  types.AllocationStateReleased,
	}); ok {
		t.Fatal("released host should be skipped")
	}
	if _, ok := newEC2HostResource(types.Host{
		State: types.AllocationStateAvailable,
	}); ok {
		t.Fatal("host with empty ID should be skipped")
	}
	if _, ok := newEC2HostResource(types.Host{
		HostId: aws.String("h-123"),
		State:  types.AllocationStateAvailable,
	}); ok {
		t.Fatal("host without sizing properties should be skipped")
	}
	if _, ok := newEC2HostResource(types.Host{
		HostId:         aws.String("h-123"),
		HostProperties: &types.HostProperties{InstanceFamily: aws.String("m5")},
		State:          types.AllocationStatePending,
	}); ok {
		t.Fatal("pending host should be skipped")
	}
}

func TestNewEC2TrafficMirrorFilterRuleResource(t *testing.T) {
	resource, ok := newEC2TrafficMirrorFilterRuleResource(types.TrafficMirrorFilterRule{
		TrafficMirrorFilterId:     aws.String("tmf-123"),
		TrafficMirrorFilterRuleId: aws.String("tmfr-123"),
	})
	if !ok {
		t.Fatal("newEC2TrafficMirrorFilterRuleResource() ok = false, want true")
	}
	if got := resource.InstanceInfo.Type; got != ec2TrafficMirrorFilterRuleResourceType {
		t.Fatalf("resource type = %q, want %q", got, ec2TrafficMirrorFilterRuleResourceType)
	}
	if got := resource.InstanceState.ID; got != "tmfr-123" {
		t.Fatalf("resource ID = %q, want tmfr-123", got)
	}
	if got := resource.InstanceState.Attributes["traffic_mirror_filter_id"]; got != "tmf-123" {
		t.Fatalf("traffic_mirror_filter_id = %q, want tmf-123", got)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "tmf-123:tmfr-123" {
		t.Fatalf("import_id = %#v, want tmf-123:tmfr-123", got)
	}

	if _, ok := newEC2TrafficMirrorFilterRuleResource(types.TrafficMirrorFilterRule{
		TrafficMirrorFilterRuleId: aws.String("tmfr-123"),
	}); ok {
		t.Fatal("filter rule with empty parent filter ID should be skipped")
	}
	if _, ok := newEC2TrafficMirrorFilterRuleResource(types.TrafficMirrorFilterRule{
		TrafficMirrorFilterId: aws.String("tmf-123"),
	}); ok {
		t.Fatal("filter rule with empty rule ID should be skipped")
	}
}

func TestEC2CoreResourceNamesPreservePartBoundaries(t *testing.T) {
	left := terraformutils.TfSanitize(ec2CoreResourceName("traffic_mirror_filter_rule", "ab", "c"))
	right := terraformutils.TfSanitize(ec2CoreResourceName("traffic_mirror_filter_rule", "a", "bc"))
	if left == right {
		t.Fatalf("resource names collide: %q", left)
	}
}

func TestEC2TrafficMirrorFilterRuleImportID(t *testing.T) {
	if got, want := ec2TrafficMirrorFilterRuleImportID("tmf-123", "tmfr-123"), "tmf-123:tmfr-123"; got != want {
		t.Fatalf("ec2TrafficMirrorFilterRuleImportID() = %q, want %q", got, want)
	}
	if got := ec2TrafficMirrorFilterRuleImportID("", "tmfr-123"); got != "" {
		t.Fatalf("ec2TrafficMirrorFilterRuleImportID(empty filter) = %q, want empty", got)
	}
}
