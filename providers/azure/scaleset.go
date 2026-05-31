// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v8"
	"github.com/chenrui333/terraformer/terraformutils"
)

type ScaleSetGenerator struct {
	AzureService
}

func (g *ScaleSetGenerator) InitResources() error {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	client, err := armcompute.NewVirtualMachineScaleSetsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return err
	}

	var scaleSets []*armcompute.VirtualMachineScaleSet
	if rg := g.Args["resource_group"].(string); rg != "" {
		pager := client.NewListPager(rg, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return err
			}
			scaleSets = append(scaleSets, page.Value...)
		}
	} else {
		pager := client.NewListAllPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return err
			}
			scaleSets = append(scaleSets, page.Value...)
		}
	}

	var resources []terraformutils.Resource
	for _, scaleSet := range scaleSets {
		resources = append(resources, terraformutils.NewSimpleResource(
			*scaleSet.ID,
			*scaleSet.Name,
			"azurerm_virtual_machine_scale_set",
			"azurerm",
			[]string{}))
	}
	g.Resources = resources
	return nil
}
