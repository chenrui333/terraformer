// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/chenrui333/terraformer/terraformutils"
)

type DiskGenerator struct {
	AzureService
}

func (g *DiskGenerator) InitResources() error {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	client, err := armcompute.NewDisksClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return err
	}

	var disks []*armcompute.Disk
	rg := g.Args["resource_group"].(string)
	if rg != "" {
		pager := client.NewListByResourceGroupPager(rg, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return err
			}
			disks = append(disks, page.Value...)
		}
	} else {
		pager := client.NewListPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return err
			}
			disks = append(disks, page.Value...)
		}
	}

	var resources []terraformutils.Resource
	for _, disk := range disks {
		resources = append(resources, terraformutils.NewSimpleResource(
			*disk.ID,
			*disk.Name,
			"azurerm_managed_disk",
			"azurerm",
			[]string{}))
	}
	g.Resources = resources
	return nil
}
