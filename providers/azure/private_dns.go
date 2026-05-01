// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
	"github.com/chenrui333/terraformer/terraformutils"
)

type PrivateDNSGenerator struct {
	AzureService
}

func (g *PrivateDNSGenerator) listRecordSets(resourceGroupName string, privateZoneName string) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	recordSetsClient, err := armprivatedns.NewRecordSetsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	pager := recordSetsClient.NewListPager(resourceGroupName, privateZoneName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, recordSet := range page.Value {
			// NOTE:
			// Format example: "Microsoft.Network/privateDnsZones/CNAME"
			recordTypeSplitted := strings.Split(*recordSet.Type, "/")
			recordType := recordTypeSplitted[len(recordTypeSplitted)-1]
			typeResourceNameMap := map[string]string{
				"A":     "azurerm_private_dns_a_record",
				"AAAA":  "azurerm_private_dns_aaaa_record",
				"CNAME": "azurerm_private_dns_cname_record",
				"MX":    "azurerm_private_dns_mx_record",
				"PTR":   "azurerm_private_dns_ptr_record",
				"SRV":   "azurerm_private_dns_srv_record",
				"TXT":   "azurerm_private_dns_txt_record",
			}
			if resName, exist := typeResourceNameMap[recordType]; exist {
				resources = append(resources, terraformutils.NewSimpleResource(
					*recordSet.ID,
					*recordSet.Name,
					resName,
					g.ProviderName,
					[]string{}))
			}
		}
	}
	return resources, nil
}

func (g *PrivateDNSGenerator) listVirtualNetworkLinks(resourceGroupName string, privateZoneName string) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	virtualNetworkLinksClient, err := armprivatedns.NewVirtualNetworkLinksClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	pager := virtualNetworkLinksClient.NewListPager(resourceGroupName, privateZoneName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, virtualNetworkLink := range page.Value {
			resources = append(resources, terraformutils.NewSimpleResource(
				*virtualNetworkLink.ID,
				*virtualNetworkLink.Name,
				"azurerm_private_dns_zone_virtual_network_link",
				g.ProviderName,
				[]string{}))
		}
	}

	return resources, nil
}

func (g *PrivateDNSGenerator) listAndAddForPrivateDNSZone() ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	privateZonesClient, err := armprivatedns.NewPrivateZonesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	var zones []*armprivatedns.PrivateZone
	if rg := g.Args["resource_group"].(string); rg != "" {
		pager := privateZonesClient.NewListByResourceGroupPager(rg, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			zones = append(zones, page.Value...)
		}
	} else {
		pager := privateZonesClient.NewListPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			zones = append(zones, page.Value...)
		}
	}

	for _, zone := range zones {
		resources = append(resources, terraformutils.NewSimpleResource(
			*zone.ID,
			*zone.Name,
			"azurerm_private_dns_zone",
			g.ProviderName,
			[]string{}))

		id, err := ParseAzureResourceID(*zone.ID)
		if err != nil {
			return nil, err
		}

		records, err := g.listRecordSets(id.ResourceGroup, *zone.Name)
		if err != nil {
			return nil, err
		}
		resources = append(resources, records...)

		networkLinks, err := g.listVirtualNetworkLinks(id.ResourceGroup, *zone.Name)
		if err != nil {
			return nil, err
		}
		resources = append(resources, networkLinks...)
	}

	return resources, nil
}

func (g *PrivateDNSGenerator) InitResources() error {
	functions := []func() ([]terraformutils.Resource, error){
		g.listAndAddForPrivateDNSZone,
	}

	for _, f := range functions {
		resources, err := f()
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, resources...)
	}

	return nil
}
