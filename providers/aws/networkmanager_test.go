// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	networkmanagertypes "github.com/aws/aws-sdk-go-v2/service/networkmanager/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNetworkManagerGlobalNetworkResource(t *testing.T) {
	resource, ok := newNetworkManagerGlobalNetworkResource(networkmanagertypes.GlobalNetwork{
		Description:     aws.String("core"),
		GlobalNetworkId: aws.String("global-network-123"),
		State:           networkmanagertypes.GlobalNetworkStateAvailable,
	})
	assertNetworkManagerResource(t, resource, ok, networkManagerGlobalNetworkResourceType, "global-network-123", "", nil)

	if _, ok := newNetworkManagerGlobalNetworkResource(networkmanagertypes.GlobalNetwork{
		GlobalNetworkId: aws.String("global-network-deleting"),
		State:           networkmanagertypes.GlobalNetworkStateDeleting,
	}); ok {
		t.Fatal("deleting global network should be skipped")
	}
}

func TestNetworkManagerSiteDeviceLinkConnectionResources(t *testing.T) {
	const (
		siteARN       = "arn:aws:networkmanager::123456789012:site/global-network-123/site-123"
		deviceARN     = "arn:aws:networkmanager::123456789012:device/global-network-123/device-123"
		linkARN       = "arn:aws:networkmanager::123456789012:link/global-network-123/link-123"
		connectionARN = "arn:aws:networkmanager::123456789012:connection/global-network-123/connection-123"
	)

	site, ok := newNetworkManagerSiteResource(networkmanagertypes.Site{
		GlobalNetworkId: aws.String("global-network-123"),
		SiteArn:         aws.String(siteARN),
		SiteId:          aws.String("site-123"),
		State:           networkmanagertypes.SiteStateAvailable,
	})
	assertNetworkManagerResource(t, site, ok, networkManagerSiteResourceType, "site-123", siteARN, map[string]string{
		"global_network_id": "global-network-123",
	})

	device, ok := newNetworkManagerDeviceResource(networkmanagertypes.Device{
		DeviceArn:       aws.String(deviceARN),
		DeviceId:        aws.String("device-123"),
		GlobalNetworkId: aws.String("global-network-123"),
		SiteId:          aws.String("site-123"),
		State:           networkmanagertypes.DeviceStateAvailable,
	})
	assertNetworkManagerResource(t, device, ok, networkManagerDeviceResourceType, "device-123", deviceARN, map[string]string{
		"global_network_id": "global-network-123",
		"site_id":           "site-123",
	})

	link, ok := newNetworkManagerLinkResource(networkmanagertypes.Link{
		GlobalNetworkId: aws.String("global-network-123"),
		LinkArn:         aws.String(linkARN),
		LinkId:          aws.String("link-123"),
		SiteId:          aws.String("site-123"),
		State:           networkmanagertypes.LinkStateAvailable,
	})
	assertNetworkManagerResource(t, link, ok, networkManagerLinkResourceType, "link-123", linkARN, map[string]string{
		"global_network_id": "global-network-123",
		"site_id":           "site-123",
	})

	connection, ok := newNetworkManagerConnectionResource(networkmanagertypes.Connection{
		ConnectedDeviceId: aws.String("device-456"),
		ConnectionArn:     aws.String(connectionARN),
		ConnectionId:      aws.String("connection-123"),
		DeviceId:          aws.String("device-123"),
		GlobalNetworkId:   aws.String("global-network-123"),
		State:             networkmanagertypes.ConnectionStateAvailable,
	})
	assertNetworkManagerResource(t, connection, ok, networkManagerConnectionResourceType, "connection-123", connectionARN, map[string]string{
		"connected_device_id": "device-456",
		"device_id":           "device-123",
		"global_network_id":   "global-network-123",
	})
	if _, ok := connection.InstanceState.Attributes["connected_link_id"]; ok {
		t.Fatal("empty connected_link_id should not be seeded")
	}
	if _, ok := connection.InstanceState.Attributes["link_id"]; ok {
		t.Fatal("empty link_id should not be seeded")
	}
}

func TestNetworkManagerStatePredicates(t *testing.T) {
	if _, ok := newNetworkManagerSiteResource(networkmanagertypes.Site{
		GlobalNetworkId: aws.String("global-network-123"),
		SiteArn:         aws.String("arn:aws:networkmanager::123456789012:site/global-network-123/site-deleting"),
		SiteId:          aws.String("site-deleting"),
		State:           networkmanagertypes.SiteStateDeleting,
	}); ok {
		t.Fatal("deleting site should be skipped")
	}

	if _, ok := newNetworkManagerLinkResource(networkmanagertypes.Link{
		GlobalNetworkId: aws.String("global-network-123"),
		LinkArn:         aws.String("arn:aws:networkmanager::123456789012:link/global-network-123/link-missing-site"),
		LinkId:          aws.String("link-missing-site"),
		State:           networkmanagertypes.LinkStateAvailable,
	}); ok {
		t.Fatal("link without required site ID should be skipped")
	}

	if _, ok := newNetworkManagerSiteResource(networkmanagertypes.Site{
		GlobalNetworkId: aws.String("global-network-123"),
		SiteId:          aws.String("site-missing-arn"),
		State:           networkmanagertypes.SiteStateAvailable,
	}); ok {
		t.Fatal("site without import ARN should be skipped")
	}
}

func TestNetworkManagerTypedFilterBehavior(t *testing.T) {
	g := NetworkManagerGenerator{}
	g.Filter = []terraformutils.ResourceFilter{
		{ServiceName: "networkmanager_site", FieldPath: "id", AcceptableValues: []string{"site-123"}},
	}

	if !g.shouldLoadNetworkManagerResource(networkManagerSiteResourceType) {
		t.Fatal("site typed filter should load sites")
	}
	if g.shouldLoadNetworkManagerResource(networkManagerDeviceResourceType) {
		t.Fatal("site typed filter should not load devices")
	}

	g.Filter = []terraformutils.ResourceFilter{
		{ServiceName: "aws_vpc", FieldPath: "id", AcceptableValues: []string{"vpc-123"}},
	}
	if !g.shouldLoadNetworkManagerResource(networkManagerGlobalNetworkResourceType) {
		t.Fatal("unrelated typed filter should preserve global network discovery")
	}
	if !g.shouldLoadNetworkManagerResource(networkManagerSiteResourceType) {
		t.Fatal("unrelated typed filter should preserve site discovery")
	}
	if !g.shouldLoadNetworkManagerResource(networkManagerConnectionResourceType) {
		t.Fatal("unrelated typed filter should preserve connection discovery")
	}
}

func assertNetworkManagerResource(t *testing.T, resource terraformutils.Resource, ok bool, resourceType, id, importID string, attributes map[string]string) {
	t.Helper()
	if !ok {
		t.Fatalf("expected %s resource", resourceType)
	}
	if resource.InstanceInfo.Type != resourceType {
		t.Fatalf("resource type = %q, want %s", resource.InstanceInfo.Type, resourceType)
	}
	if resource.InstanceState.ID != id {
		t.Fatalf("resource ID = %q, want %s", resource.InstanceState.ID, id)
	}
	if importID != "" {
		if got := resource.InstanceState.Meta["import_id"]; got != importID {
			t.Fatalf("import_id = %#v, want %q", got, importID)
		}
	}
	for key, want := range attributes {
		if got := resource.InstanceState.Attributes[key]; got != want {
			t.Fatalf("attribute %s = %q, want %q", key, got, want)
		}
	}
}
