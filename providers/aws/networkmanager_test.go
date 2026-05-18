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
	assertNetworkManagerResource(t, resource, ok, networkManagerGlobalNetworkResourceType, "global-network-123", nil)

	if _, ok := newNetworkManagerGlobalNetworkResource(networkmanagertypes.GlobalNetwork{
		GlobalNetworkId: aws.String("global-network-deleting"),
		State:           networkmanagertypes.GlobalNetworkStateDeleting,
	}); ok {
		t.Fatal("deleting global network should be skipped")
	}
}

func TestNetworkManagerSiteDeviceLinkConnectionResources(t *testing.T) {
	site, ok := newNetworkManagerSiteResource(networkmanagertypes.Site{
		GlobalNetworkId: aws.String("global-network-123"),
		SiteId:          aws.String("site-123"),
		State:           networkmanagertypes.SiteStateAvailable,
	})
	assertNetworkManagerResource(t, site, ok, networkManagerSiteResourceType, "site-123", map[string]string{
		"global_network_id": "global-network-123",
	})

	device, ok := newNetworkManagerDeviceResource(networkmanagertypes.Device{
		DeviceId:        aws.String("device-123"),
		GlobalNetworkId: aws.String("global-network-123"),
		SiteId:          aws.String("site-123"),
		State:           networkmanagertypes.DeviceStateAvailable,
	})
	assertNetworkManagerResource(t, device, ok, networkManagerDeviceResourceType, "device-123", map[string]string{
		"global_network_id": "global-network-123",
		"site_id":           "site-123",
	})

	link, ok := newNetworkManagerLinkResource(networkmanagertypes.Link{
		GlobalNetworkId: aws.String("global-network-123"),
		LinkId:          aws.String("link-123"),
		SiteId:          aws.String("site-123"),
		State:           networkmanagertypes.LinkStateAvailable,
	})
	assertNetworkManagerResource(t, link, ok, networkManagerLinkResourceType, "link-123", map[string]string{
		"global_network_id": "global-network-123",
		"site_id":           "site-123",
	})

	connection, ok := newNetworkManagerConnectionResource(networkmanagertypes.Connection{
		ConnectedDeviceId: aws.String("device-456"),
		ConnectionId:      aws.String("connection-123"),
		DeviceId:          aws.String("device-123"),
		GlobalNetworkId:   aws.String("global-network-123"),
		State:             networkmanagertypes.ConnectionStateAvailable,
	})
	assertNetworkManagerResource(t, connection, ok, networkManagerConnectionResourceType, "connection-123", map[string]string{
		"connected_device_id": "device-456",
		"device_id":           "device-123",
		"global_network_id":   "global-network-123",
	})
}

func TestNetworkManagerStatePredicates(t *testing.T) {
	if _, ok := newNetworkManagerSiteResource(networkmanagertypes.Site{
		GlobalNetworkId: aws.String("global-network-123"),
		SiteId:          aws.String("site-deleting"),
		State:           networkmanagertypes.SiteStateDeleting,
	}); ok {
		t.Fatal("deleting site should be skipped")
	}

	if _, ok := newNetworkManagerLinkResource(networkmanagertypes.Link{
		GlobalNetworkId: aws.String("global-network-123"),
		LinkId:          aws.String("link-missing-site"),
		State:           networkmanagertypes.LinkStateAvailable,
	}); ok {
		t.Fatal("link without required site ID should be skipped")
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
}

func assertNetworkManagerResource(t *testing.T, resource terraformutils.Resource, ok bool, resourceType, id string, attributes map[string]string) {
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
	for key, want := range attributes {
		if got := resource.InstanceState.Attributes[key]; got != want {
			t.Fatalf("attribute %s = %q, want %q", key, got, want)
		}
	}
}
