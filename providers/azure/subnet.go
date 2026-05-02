// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

type SubnetGenerator struct {
	AzureService
}

func (az *SubnetGenerator) lisSubnets() ([]*armnetwork.Subnet, error) {
	subscriptionID, resourceGroup, credential, clientOptions := az.getClientArgs()
	subnetClient, err := armnetwork.NewSubnetsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	vnetClient, err := armnetwork.NewVirtualNetworksClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	var vnets []*armnetwork.VirtualNetwork
	if resourceGroup != "" {
		pager := vnetClient.NewListPager(resourceGroup, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			vnets = append(vnets, page.Value...)
		}
	} else {
		pager := vnetClient.NewListAllPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			vnets = append(vnets, page.Value...)
		}
	}

	var resources []*armnetwork.Subnet
	for _, vnet := range vnets {
		vnetID, err := ParseAzureResourceID(*vnet.ID)
		if err != nil {
			return nil, err
		}
		pager := subnetClient.NewListPager(vnetID.ResourceGroup, *vnet.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			resources = append(resources, page.Value...)
		}
	}
	return resources, nil
}

func (az *SubnetGenerator) AppendSubnet(subnet *armnetwork.Subnet) {
	az.AppendSimpleResource(*subnet.ID, *subnet.Name, "azurerm_subnet")
}

func (az *SubnetGenerator) appendRouteTable(subnet *armnetwork.Subnet) {
	if props := subnet.Properties; props != nil {
		if prop := props.RouteTable; prop != nil {
			az.appendSimpleAssociation(
				*subnet.ID, *subnet.Name, prop.Name,
				"azurerm_subnet_route_table_association",
				map[string]string{
					"subnet_id":      *subnet.ID,
					"route_table_id": *prop.ID,
				})
		}
	}
}

func (az *SubnetGenerator) appendNetworkSecurityGroupAssociation(subnet *armnetwork.Subnet) {
	if props := subnet.Properties; props != nil {
		if prop := props.NetworkSecurityGroup; prop != nil {
			az.appendSimpleAssociation(
				*subnet.ID, *subnet.Name, prop.Name,
				"azurerm_subnet_network_security_group_association",
				map[string]string{
					"subnet_id":                 *subnet.ID,
					"network_security_group_id": *prop.ID,
				})
		}
	}
}

func (az *SubnetGenerator) appendNatGateway(subnet *armnetwork.Subnet) {
	if props := subnet.Properties; props != nil {
		if prop := props.NatGateway; prop != nil {
			az.appendSimpleAssociation(
				*subnet.ID, *subnet.Name, nil,
				"azurerm_subnet_nat_gateway_association",
				map[string]string{
					"subnet_id":      *subnet.ID,
					"nat_gateway_id": *prop.ID,
				})
		}
	}
}

func (az *SubnetGenerator) appendServiceEndpointPolicies() error {
	subscriptionID, resourceGroup, credential, clientOptions := az.getClientArgs()
	client, err := armnetwork.NewServiceEndpointPoliciesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return err
	}
	ctx := context.Background()

	if resourceGroup != "" {
		pager := client.NewListByResourceGroupPager(resourceGroup, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return err
			}
			for _, item := range page.Value {
				az.AppendSimpleResource(*item.ID, *item.Name, "azurerm_subnet_service_endpoint_storage_policy")
			}
		}
	} else {
		pager := client.NewListPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return err
			}
			for _, item := range page.Value {
				az.AppendSimpleResource(*item.ID, *item.Name, "azurerm_subnet_service_endpoint_storage_policy")
			}
		}
	}
	return nil
}

func (az *SubnetGenerator) InitResources() error {
	subnets, err := az.lisSubnets()
	if err != nil {
		return err
	}
	for _, subnet := range subnets {
		az.AppendSubnet(subnet)
		az.appendRouteTable(subnet)
		az.appendNetworkSecurityGroupAssociation(subnet)
		az.appendNatGateway(subnet)
	}
	if err := az.appendServiceEndpointPolicies(); err != nil {
		return err
	}
	return nil
}

func (az *SubnetGenerator) PostConvertHook() error {
	for _, resource := range az.Resources {
		if resource.InstanceInfo.Type != "azurerm_subnet" {
			continue
		}
		delete(resource.Item, "address_prefix")
	}
	return nil
}
