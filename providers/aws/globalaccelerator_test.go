// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/globalaccelerator/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	testGlobalAcceleratorAcceleratorARN            = "arn:aws:globalaccelerator::123456789012:accelerator/accel-1"
	testGlobalAcceleratorListenerARN               = testGlobalAcceleratorAcceleratorARN + "/listener/listener-1"
	testGlobalAcceleratorEndpointGroupARN          = testGlobalAcceleratorListenerARN + "/endpoint-group/endpoint-group-1"
	testGlobalAcceleratorCustomAcceleratorARN      = "arn:aws:globalaccelerator::123456789012:accelerator/custom-accel-1"
	testGlobalAcceleratorCustomListenerARN         = testGlobalAcceleratorCustomAcceleratorARN + "/listener/custom-listener-1"
	testGlobalAcceleratorCustomEndpointGroupARN    = testGlobalAcceleratorCustomListenerARN + "/endpoint-group/custom-endpoint-group-1"
	testGlobalAcceleratorCrossAccountAttachmentARN = "arn:aws:globalaccelerator::123456789012:attachment/attachment-1"
)

func TestGlobalAcceleratorImportID(t *testing.T) {
	if got := globalAcceleratorImportID(testGlobalAcceleratorAcceleratorARN); got != testGlobalAcceleratorAcceleratorARN {
		t.Fatalf("globalAcceleratorImportID() = %q, want %q", got, testGlobalAcceleratorAcceleratorARN)
	}
}

func TestGlobalAcceleratorClientConfigUsesConcreteRegionForGlobalImport(t *testing.T) {
	config := globalAcceleratorClientConfig(aws.Config{Region: GlobalRegion})
	if got := config.Region; got != MainRegionPublicPartition {
		t.Fatalf("Region = %q, want %q", got, MainRegionPublicPartition)
	}

	config = globalAcceleratorClientConfig(aws.Config{Region: "us-west-2"})
	if got := config.Region; got != "us-west-2" {
		t.Fatalf("Region = %q, want us-west-2", got)
	}
}

func TestGlobalAcceleratorARNHelpers(t *testing.T) {
	if got, want := globalAcceleratorARNLastPart(testGlobalAcceleratorEndpointGroupARN), "endpoint-group-1"; got != want {
		t.Fatalf("globalAcceleratorARNLastPart() = %q, want %q", got, want)
	}
	if got := globalAcceleratorAcceleratorARNFromChildARN(testGlobalAcceleratorListenerARN); got != testGlobalAcceleratorAcceleratorARN {
		t.Fatalf("globalAcceleratorAcceleratorARNFromChildARN() = %q, want %q", got, testGlobalAcceleratorAcceleratorARN)
	}
	if got := globalAcceleratorListenerARNFromEndpointGroupARN(testGlobalAcceleratorEndpointGroupARN); got != testGlobalAcceleratorListenerARN {
		t.Fatalf("globalAcceleratorListenerARNFromEndpointGroupARN() = %q, want %q", got, testGlobalAcceleratorListenerARN)
	}
	if got := globalAcceleratorAcceleratorARNFromChildARN("not-a-child"); got != "" {
		t.Fatalf("globalAcceleratorAcceleratorARNFromChildARN(invalid) = %q, want empty", got)
	}
}

func TestNewGlobalAcceleratorAcceleratorResource(t *testing.T) {
	resource, ok := newGlobalAcceleratorAcceleratorResource(&types.Accelerator{
		AcceleratorArn: aws.String(testGlobalAcceleratorAcceleratorARN),
		Enabled:        aws.Bool(false),
		IpAddressType:  types.IpAddressTypeDualStack,
		Name:           aws.String("edge-main"),
		Status:         types.AcceleratorStatusDeployed,
	})
	if !ok {
		t.Fatal("newGlobalAcceleratorAcceleratorResource() ok = false, want true")
	}
	if resource.InstanceInfo.Type != globalAcceleratorAcceleratorResourceType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, globalAcceleratorAcceleratorResourceType)
	}
	if resource.InstanceState.ID != testGlobalAcceleratorAcceleratorARN {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, testGlobalAcceleratorAcceleratorARN)
	}
	if got, want := resource.InstanceState.Attributes["name"], "edge-main"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["enabled"], "false"; got != want {
		t.Fatalf("enabled = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["ip_address_type"], "DUAL_STACK"; got != want {
		t.Fatalf("ip_address_type = %q, want %q", got, want)
	}
	if got, want := resource.ResourceName, terraformutils.TfSanitize(globalAcceleratorResourceName("accelerator", "edge-main", "accel-1")); got != want {
		t.Fatalf("resource name = %q, want %q", got, want)
	}
}

func TestNewGlobalAcceleratorAcceleratorResourceSkipsUnsafeEntries(t *testing.T) {
	tests := []struct {
		name        string
		accelerator *types.Accelerator
	}{
		{name: "nil", accelerator: nil},
		{name: "missing arn", accelerator: &types.Accelerator{Name: aws.String("edge-main"), Status: types.AcceleratorStatusDeployed}},
		{name: "missing name", accelerator: &types.Accelerator{AcceleratorArn: aws.String(testGlobalAcceleratorAcceleratorARN), Status: types.AcceleratorStatusDeployed}},
		{name: "in progress", accelerator: &types.Accelerator{AcceleratorArn: aws.String(testGlobalAcceleratorAcceleratorARN), Name: aws.String("edge-main"), Status: types.AcceleratorStatusInProgress}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := newGlobalAcceleratorAcceleratorResource(tt.accelerator); ok {
				t.Fatal("newGlobalAcceleratorAcceleratorResource() ok = true, want false")
			}
		})
	}
}

func TestNewGlobalAcceleratorListenerResource(t *testing.T) {
	resource, ok := newGlobalAcceleratorListenerResource(&types.Listener{
		ClientAffinity: types.ClientAffinitySourceIp,
		ListenerArn:    aws.String(testGlobalAcceleratorListenerARN),
		PortRanges: []types.PortRange{
			{FromPort: aws.Int32(80), ToPort: aws.Int32(80)},
			{FromPort: aws.Int32(443), ToPort: aws.Int32(443)},
		},
		Protocol: types.ProtocolTcp,
	})
	if !ok {
		t.Fatal("newGlobalAcceleratorListenerResource() ok = false, want true")
	}
	attrs := resource.InstanceState.Attributes
	if got := resource.InstanceState.ID; got != testGlobalAcceleratorListenerARN {
		t.Fatalf("resource ID = %q, want %q", got, testGlobalAcceleratorListenerARN)
	}
	if got := attrs["accelerator_arn"]; got != testGlobalAcceleratorAcceleratorARN {
		t.Fatalf("accelerator_arn = %q, want %q", got, testGlobalAcceleratorAcceleratorARN)
	}
	if got, want := attrs["protocol"], "TCP"; got != want {
		t.Fatalf("protocol = %q, want %q", got, want)
	}
	if got, want := attrs["port_range.#"], "2"; got != want {
		t.Fatalf("port_range.# = %q, want %q", got, want)
	}
	if got, want := attrs["port_range.1.from_port"], "443"; got != want {
		t.Fatalf("port_range.1.from_port = %q, want %q", got, want)
	}
}

func TestNewGlobalAcceleratorListenerResourceSkipsIncomplete(t *testing.T) {
	tests := []struct {
		name     string
		listener *types.Listener
	}{
		{name: "nil", listener: nil},
		{name: "missing arn", listener: &types.Listener{Protocol: types.ProtocolTcp, PortRanges: []types.PortRange{{FromPort: aws.Int32(80), ToPort: aws.Int32(80)}}}},
		{name: "missing parent", listener: &types.Listener{ListenerArn: aws.String("arn:aws:globalaccelerator::123456789012:listener/listener-1"), Protocol: types.ProtocolTcp, PortRanges: []types.PortRange{{FromPort: aws.Int32(80), ToPort: aws.Int32(80)}}}},
		{name: "missing protocol", listener: &types.Listener{ListenerArn: aws.String(testGlobalAcceleratorListenerARN), PortRanges: []types.PortRange{{FromPort: aws.Int32(80), ToPort: aws.Int32(80)}}}},
		{name: "missing complete port range", listener: &types.Listener{ListenerArn: aws.String(testGlobalAcceleratorListenerARN), Protocol: types.ProtocolTcp, PortRanges: []types.PortRange{{FromPort: aws.Int32(80)}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := newGlobalAcceleratorListenerResource(tt.listener); ok {
				t.Fatal("newGlobalAcceleratorListenerResource() ok = true, want false")
			}
		})
	}
}

func TestNewGlobalAcceleratorEndpointGroupResource(t *testing.T) {
	trafficDial := float32(75.5)
	resource, ok := newGlobalAcceleratorEndpointGroupResource(&types.EndpointGroup{
		EndpointGroupArn:    aws.String(testGlobalAcceleratorEndpointGroupARN),
		EndpointGroupRegion: aws.String("us-east-1"),
		EndpointDescriptions: []types.EndpointDescription{
			{
				ClientIPPreservationEnabled: aws.Bool(false),
				EndpointId:                  aws.String("arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/app-lb/123"),
				Weight:                      aws.Int32(0),
			},
		},
		HealthCheckIntervalSeconds: aws.Int32(10),
		HealthCheckPath:            aws.String("/healthz"),
		HealthCheckPort:            aws.Int32(8080),
		HealthCheckProtocol:        types.HealthCheckProtocolHttps,
		PortOverrides: []types.PortOverride{
			{EndpointPort: aws.Int32(8443), ListenerPort: aws.Int32(443)},
		},
		ThresholdCount:        aws.Int32(5),
		TrafficDialPercentage: &trafficDial,
	})
	if !ok {
		t.Fatal("newGlobalAcceleratorEndpointGroupResource() ok = false, want true")
	}
	attrs := resource.InstanceState.Attributes
	if got := attrs["listener_arn"]; got != testGlobalAcceleratorListenerARN {
		t.Fatalf("listener_arn = %q, want %q", got, testGlobalAcceleratorListenerARN)
	}
	if got, want := attrs["endpoint_group_region"], "us-east-1"; got != want {
		t.Fatalf("endpoint_group_region = %q, want %q", got, want)
	}
	if got, want := attrs["endpoint_configuration.0.client_ip_preservation_enabled"], "false"; got != want {
		t.Fatalf("endpoint_configuration.0.client_ip_preservation_enabled = %q, want %q", got, want)
	}
	if got, want := attrs["endpoint_configuration.0.weight"], "0"; got != want {
		t.Fatalf("endpoint_configuration.0.weight = %q, want %q", got, want)
	}
	if got, want := attrs["traffic_dial_percentage"], "75.5"; got != want {
		t.Fatalf("traffic_dial_percentage = %q, want %q", got, want)
	}
	if got, want := attrs["port_override.0.endpoint_port"], "8443"; got != want {
		t.Fatalf("port_override.0.endpoint_port = %q, want %q", got, want)
	}
}

func TestNewGlobalAcceleratorCustomRoutingAcceleratorResource(t *testing.T) {
	resource, ok := newGlobalAcceleratorCustomRoutingAcceleratorResource(&types.CustomRoutingAccelerator{
		AcceleratorArn: aws.String(testGlobalAcceleratorCustomAcceleratorARN),
		Enabled:        aws.Bool(true),
		IpAddressType:  types.IpAddressTypeIpv4,
		Name:           aws.String("edge-custom"),
		Status:         types.CustomRoutingAcceleratorStatusDeployed,
	})
	if !ok {
		t.Fatal("newGlobalAcceleratorCustomRoutingAcceleratorResource() ok = false, want true")
	}
	if resource.InstanceInfo.Type != globalAcceleratorCustomRoutingAcceleratorResourceType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, globalAcceleratorCustomRoutingAcceleratorResourceType)
	}
	if got, want := resource.InstanceState.Attributes["name"], "edge-custom"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
}

func TestNewGlobalAcceleratorCustomRoutingListenerResource(t *testing.T) {
	resource, ok := newGlobalAcceleratorCustomRoutingListenerResource(&types.CustomRoutingListener{
		ListenerArn: aws.String(testGlobalAcceleratorCustomListenerARN),
		PortRanges: []types.PortRange{
			{FromPort: aws.Int32(10000), ToPort: aws.Int32(10010)},
		},
	})
	if !ok {
		t.Fatal("newGlobalAcceleratorCustomRoutingListenerResource() ok = false, want true")
	}
	attrs := resource.InstanceState.Attributes
	if resource.InstanceInfo.Type != globalAcceleratorCustomRoutingListenerResourceType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, globalAcceleratorCustomRoutingListenerResourceType)
	}
	if got := attrs["accelerator_arn"]; got != testGlobalAcceleratorCustomAcceleratorARN {
		t.Fatalf("accelerator_arn = %q, want %q", got, testGlobalAcceleratorCustomAcceleratorARN)
	}
	if got, want := attrs["port_range.0.to_port"], "10010"; got != want {
		t.Fatalf("port_range.0.to_port = %q, want %q", got, want)
	}
}

func TestNewGlobalAcceleratorCustomRoutingEndpointGroupResource(t *testing.T) {
	resource, ok := newGlobalAcceleratorCustomRoutingEndpointGroupResource(&types.CustomRoutingEndpointGroup{
		DestinationDescriptions: []types.CustomRoutingDestinationDescription{
			{
				FromPort:  aws.Int32(10000),
				Protocols: []types.Protocol{types.ProtocolTcp, types.ProtocolUdp},
				ToPort:    aws.Int32(10010),
			},
		},
		EndpointDescriptions: []types.CustomRoutingEndpointDescription{
			{EndpointId: aws.String("subnet-123")},
		},
		EndpointGroupArn:    aws.String(testGlobalAcceleratorCustomEndpointGroupARN),
		EndpointGroupRegion: aws.String("us-west-2"),
	})
	if !ok {
		t.Fatal("newGlobalAcceleratorCustomRoutingEndpointGroupResource() ok = false, want true")
	}
	attrs := resource.InstanceState.Attributes
	if got := attrs["listener_arn"]; got != testGlobalAcceleratorCustomListenerARN {
		t.Fatalf("listener_arn = %q, want %q", got, testGlobalAcceleratorCustomListenerARN)
	}
	if got, want := attrs["destination_configuration.#"], "1"; got != want {
		t.Fatalf("destination_configuration.# = %q, want %q", got, want)
	}
	if got, want := attrs["destination_configuration.0.protocols.1"], "UDP"; got != want {
		t.Fatalf("destination_configuration.0.protocols.1 = %q, want %q", got, want)
	}
	if got, want := attrs["endpoint_configuration.0.endpoint_id"], "subnet-123"; got != want {
		t.Fatalf("endpoint_configuration.0.endpoint_id = %q, want %q", got, want)
	}
}

func TestNewGlobalAcceleratorCustomRoutingEndpointGroupResourceSkipsIncompleteDestination(t *testing.T) {
	if _, ok := newGlobalAcceleratorCustomRoutingEndpointGroupResource(&types.CustomRoutingEndpointGroup{
		EndpointGroupArn:    aws.String(testGlobalAcceleratorCustomEndpointGroupARN),
		EndpointGroupRegion: aws.String("us-west-2"),
		DestinationDescriptions: []types.CustomRoutingDestinationDescription{
			{FromPort: aws.Int32(10000), ToPort: aws.Int32(10010)},
		},
	}); ok {
		t.Fatal("newGlobalAcceleratorCustomRoutingEndpointGroupResource() ok = true, want false")
	}
}

func TestNewGlobalAcceleratorCrossAccountAttachmentResource(t *testing.T) {
	resource, ok := newGlobalAcceleratorCrossAccountAttachmentResource(&types.Attachment{
		AttachmentArn: aws.String(testGlobalAcceleratorCrossAccountAttachmentARN),
		Name:          aws.String("shared-edge"),
		Principals:    []string{"111111111111", "222222222222"},
		Resources: []types.Resource{
			{EndpointId: aws.String("arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/net/nlb/123"), Region: aws.String("us-east-1")},
			{Cidr: aws.String("203.0.113.0/24")},
		},
	})
	if !ok {
		t.Fatal("newGlobalAcceleratorCrossAccountAttachmentResource() ok = false, want true")
	}
	attrs := resource.InstanceState.Attributes
	if resource.InstanceInfo.Type != globalAcceleratorCrossAccountAttachmentResourceType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, globalAcceleratorCrossAccountAttachmentResourceType)
	}
	if got, want := attrs["principals.#"], "2"; got != want {
		t.Fatalf("principals.# = %q, want %q", got, want)
	}
	if got, want := attrs["resource.#"], "2"; got != want {
		t.Fatalf("resource.# = %q, want %q", got, want)
	}
	if got, want := attrs["resource.0.region"], "us-east-1"; got != want {
		t.Fatalf("resource.0.region = %q, want %q", got, want)
	}
	if got, want := attrs["resource.1.cidr"], "203.0.113.0/24"; got != want {
		t.Fatalf("resource.1.cidr = %q, want %q", got, want)
	}
}

func TestGlobalAcceleratorResourceNameIncludesLengths(t *testing.T) {
	name := globalAcceleratorResourceName("listener", "ab", "c_d")
	if got, want := name, "8_listener_2_ab_3_c_d"; got != want {
		t.Fatalf("globalAcceleratorResourceName() = %q, want %q", got, want)
	}
}

func TestGlobalAcceleratorResourceNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "generic", err: errors.New("boom"), want: false},
		{name: "accelerator", err: &types.AcceleratorNotFoundException{}, want: true},
		{name: "listener", err: &types.ListenerNotFoundException{}, want: true},
		{name: "endpoint group", err: &types.EndpointGroupNotFoundException{}, want: true},
		{name: "attachment", err: &types.AttachmentNotFoundException{}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := globalAcceleratorResourceNotFound(tt.err); got != tt.want {
				t.Fatalf("globalAcceleratorResourceNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}
