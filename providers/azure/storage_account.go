// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v4"
	"github.com/chenrui333/terraformer/terraformutils"
)

type StorageAccountGenerator struct {
	AzureService
}

func (g *StorageAccountGenerator) InitResources() error {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	accountsClient, err := armstorage.NewAccountsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return err
	}

	var accounts []*armstorage.Account
	if rg := g.Args["resource_group"].(string); rg != "" {
		pager := accountsClient.NewListByResourceGroupPager(rg, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return err
			}
			accounts = append(accounts, page.Value...)
		}
	} else {
		pager := accountsClient.NewListPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return err
			}
			accounts = append(accounts, page.Value...)
		}
	}

	var resources []terraformutils.Resource
	for _, account := range accounts {
		resources = append(resources, terraformutils.NewSimpleResource(
			*account.ID,
			*account.Name,
			"azurerm_storage_account",
			"azurerm",
			[]string{}))
	}
	g.Resources = resources
	return nil
}
