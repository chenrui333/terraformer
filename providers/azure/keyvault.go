// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault/v2"
	"github.com/chenrui333/terraformer/terraformutils"
)

type KeyVaultGenerator struct {
	AzureService
}

func (g *KeyVaultGenerator) InitResources() error {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	client, err := armkeyvault.NewVaultsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return err
	}

	rg := g.Args["resource_group"].(string)
	var resources []terraformutils.Resource
	if rg != "" {
		pager := client.NewListByResourceGroupPager(rg, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return err
			}
			for _, vault := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*vault.ID,
					*vault.Name,
					"azurerm_key_vault",
					"azurerm",
					[]string{}))
			}
		}
	} else {
		pager := client.NewListPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return err
			}
			for _, vault := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*vault.ID,
					*vault.Name,
					"azurerm_key_vault",
					"azurerm",
					[]string{}))
			}
		}
	}
	g.Resources = resources
	return nil
}
