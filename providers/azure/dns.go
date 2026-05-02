// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
	"github.com/chenrui333/terraformer/terraformutils"
)

type DNSGenerator struct {
	AzureService
}

func (g *DNSGenerator) listRecordSets(resourceGroupName string, zoneName string) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	recordSetsClient, err := armdns.NewRecordSetsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	pager := recordSetsClient.NewListAllByDNSZonePager(resourceGroupName, zoneName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, recordSet := range page.Value {
			// NOTE:
			// Format example: "Microsoft.Network/dnszones/AAAA"
			recordTypeSplitted := strings.Split(*recordSet.Type, "/")
			recordType := recordTypeSplitted[len(recordTypeSplitted)-1]
			typeResourceNameMap := map[string]string{
				"A":     "azurerm_dns_a_record",
				"AAAA":  "azurerm_dns_aaaa_record",
				"CAA":   "azurerm_dns_caa_record",
				"CNAME": "azurerm_dns_cname_record",
				"MX":    "azurerm_dns_mx_record",
				"NS":    "azurerm_dns_ns_record",
				"PTR":   "azurerm_dns_ptr_record",
				"SRV":   "azurerm_dns_srv_record",
				"TXT":   "azurerm_dns_txt_record",
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

func (g *DNSGenerator) listAndAddForDNSZone() ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	zonesClient, err := armdns.NewZonesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	var zones []*armdns.Zone
	if rg := g.Args["resource_group"].(string); rg != "" {
		pager := zonesClient.NewListByResourceGroupPager(rg, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			zones = append(zones, page.Value...)
		}
	} else {
		pager := zonesClient.NewListPager(nil)
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
			"azurerm_dns_zone",
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
	}

	return resources, nil
}

func (g *DNSGenerator) InitResources() error {
	functions := []func() ([]terraformutils.Resource, error){
		g.listAndAddForDNSZone,
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
