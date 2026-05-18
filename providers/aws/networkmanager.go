// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	networkmanagertypes "github.com/aws/aws-sdk-go-v2/service/networkmanager/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	networkManagerGlobalNetworkResourceType = "aws_networkmanager_global_network"
	networkManagerSiteResourceType          = "aws_networkmanager_site"
	networkManagerDeviceResourceType        = "aws_networkmanager_device"
	networkManagerLinkResourceType          = "aws_networkmanager_link"
	networkManagerConnectionResourceType    = "aws_networkmanager_connection"
)

var networkManagerAllowEmptyValues = []string{"tags."}

type NetworkManagerGenerator struct {
	AWSService
}

func (g *NetworkManagerGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := networkmanager.NewFromConfig(config)

	loadGlobalNetworks := g.shouldLoadNetworkManagerResource(networkManagerGlobalNetworkResourceType)
	loadSites := g.shouldLoadNetworkManagerResource(networkManagerSiteResourceType)
	loadDevices := g.shouldLoadNetworkManagerResource(networkManagerDeviceResourceType)
	loadLinks := g.shouldLoadNetworkManagerResource(networkManagerLinkResourceType)
	loadConnections := g.shouldLoadNetworkManagerResource(networkManagerConnectionResourceType)
	if !(loadGlobalNetworks || loadSites || loadDevices || loadLinks || loadConnections) {
		return nil
	}

	p := networkmanager.NewDescribeGlobalNetworksPaginator(svc, &networkmanager.DescribeGlobalNetworksInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, globalNetwork := range page.GlobalNetworks {
			if !networkManagerGlobalNetworkImportable(globalNetwork) {
				continue
			}
			globalNetworkID := StringValue(globalNetwork.GlobalNetworkId)
			if loadGlobalNetworks {
				if resource, ok := newNetworkManagerGlobalNetworkResource(globalNetwork); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
			if loadSites {
				if err := g.loadNetworkManagerSites(svc, globalNetworkID); err != nil {
					return err
				}
			}
			if loadDevices {
				if err := g.loadNetworkManagerDevices(svc, globalNetworkID); err != nil {
					return err
				}
			}
			if loadLinks {
				if err := g.loadNetworkManagerLinks(svc, globalNetworkID); err != nil {
					return err
				}
			}
			if loadConnections {
				if err := g.loadNetworkManagerConnections(svc, globalNetworkID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (g *NetworkManagerGenerator) shouldLoadNetworkManagerResource(serviceNames ...string) bool {
	return shouldLoadAWSResourceForTypedFilters(g.Filter, serviceNames...)
}

func (g *NetworkManagerGenerator) loadNetworkManagerSites(svc *networkmanager.Client, globalNetworkID string) error {
	if globalNetworkID == "" {
		return nil
	}
	p := networkmanager.NewGetSitesPaginator(svc, &networkmanager.GetSitesInput{
		GlobalNetworkId: aws.String(globalNetworkID),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, site := range page.Sites {
			if resource, ok := newNetworkManagerSiteResource(site); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *NetworkManagerGenerator) loadNetworkManagerDevices(svc *networkmanager.Client, globalNetworkID string) error {
	if globalNetworkID == "" {
		return nil
	}
	p := networkmanager.NewGetDevicesPaginator(svc, &networkmanager.GetDevicesInput{
		GlobalNetworkId: aws.String(globalNetworkID),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, device := range page.Devices {
			if resource, ok := newNetworkManagerDeviceResource(device); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *NetworkManagerGenerator) loadNetworkManagerLinks(svc *networkmanager.Client, globalNetworkID string) error {
	if globalNetworkID == "" {
		return nil
	}
	p := networkmanager.NewGetLinksPaginator(svc, &networkmanager.GetLinksInput{
		GlobalNetworkId: aws.String(globalNetworkID),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, link := range page.Links {
			if resource, ok := newNetworkManagerLinkResource(link); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *NetworkManagerGenerator) loadNetworkManagerConnections(svc *networkmanager.Client, globalNetworkID string) error {
	if globalNetworkID == "" {
		return nil
	}
	p := networkmanager.NewGetConnectionsPaginator(svc, &networkmanager.GetConnectionsInput{
		GlobalNetworkId: aws.String(globalNetworkID),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, connection := range page.Connections {
			if resource, ok := newNetworkManagerConnectionResource(connection); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newNetworkManagerGlobalNetworkResource(globalNetwork networkmanagertypes.GlobalNetwork) (terraformutils.Resource, bool) {
	id := StringValue(globalNetwork.GlobalNetworkId)
	if id == "" || !networkManagerGlobalNetworkImportable(globalNetwork) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		id,
		networkManagerResourceName("global_network", StringValue(globalNetwork.Description), id),
		networkManagerGlobalNetworkResourceType,
		"aws",
		networkManagerAllowEmptyValues,
	), true
}

func newNetworkManagerSiteResource(site networkmanagertypes.Site) (terraformutils.Resource, bool) {
	id := StringValue(site.SiteId)
	globalNetworkID := StringValue(site.GlobalNetworkId)
	if id == "" || globalNetworkID == "" || !networkManagerSiteImportable(site) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		id,
		networkManagerResourceName("site", globalNetworkID, id),
		networkManagerSiteResourceType,
		"aws",
		map[string]string{
			"global_network_id": globalNetworkID,
		},
		networkManagerAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newNetworkManagerDeviceResource(device networkmanagertypes.Device) (terraformutils.Resource, bool) {
	id := StringValue(device.DeviceId)
	globalNetworkID := StringValue(device.GlobalNetworkId)
	if id == "" || globalNetworkID == "" || !networkManagerDeviceImportable(device) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"global_network_id": globalNetworkID,
	}
	putNetworkManagerString(attributes, "site_id", StringValue(device.SiteId))
	return terraformutils.NewResource(
		id,
		networkManagerResourceName("device", globalNetworkID, id),
		networkManagerDeviceResourceType,
		"aws",
		attributes,
		networkManagerAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newNetworkManagerLinkResource(link networkmanagertypes.Link) (terraformutils.Resource, bool) {
	id := StringValue(link.LinkId)
	globalNetworkID := StringValue(link.GlobalNetworkId)
	siteID := StringValue(link.SiteId)
	if id == "" || globalNetworkID == "" || siteID == "" || !networkManagerLinkImportable(link) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		id,
		networkManagerResourceName("link", globalNetworkID, siteID, id),
		networkManagerLinkResourceType,
		"aws",
		map[string]string{
			"global_network_id": globalNetworkID,
			"site_id":           siteID,
		},
		networkManagerAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newNetworkManagerConnectionResource(connection networkmanagertypes.Connection) (terraformutils.Resource, bool) {
	id := StringValue(connection.ConnectionId)
	globalNetworkID := StringValue(connection.GlobalNetworkId)
	if id == "" || globalNetworkID == "" || !networkManagerConnectionImportable(connection) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		id,
		networkManagerResourceName("connection", globalNetworkID, id),
		networkManagerConnectionResourceType,
		"aws",
		map[string]string{
			"connected_device_id": StringValue(connection.ConnectedDeviceId),
			"connected_link_id":   StringValue(connection.ConnectedLinkId),
			"device_id":           StringValue(connection.DeviceId),
			"global_network_id":   globalNetworkID,
			"link_id":             StringValue(connection.LinkId),
		},
		networkManagerAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func networkManagerGlobalNetworkImportable(globalNetwork networkmanagertypes.GlobalNetwork) bool {
	return globalNetwork.State != networkmanagertypes.GlobalNetworkStateDeleting
}

func networkManagerSiteImportable(site networkmanagertypes.Site) bool {
	return site.State != networkmanagertypes.SiteStateDeleting
}

func networkManagerDeviceImportable(device networkmanagertypes.Device) bool {
	return device.State != networkmanagertypes.DeviceStateDeleting
}

func networkManagerLinkImportable(link networkmanagertypes.Link) bool {
	return link.State != networkmanagertypes.LinkStateDeleting
}

func networkManagerConnectionImportable(connection networkmanagertypes.Connection) bool {
	return connection.State != networkmanagertypes.ConnectionStateDeleting
}

func putNetworkManagerString(attributes map[string]string, key, value string) {
	if value != "" {
		attributes[key] = value
	}
}

func networkManagerResourceName(parts ...string) string {
	return awsResourceNameWithLengths(parts...)
}
