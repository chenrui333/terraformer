// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/purview/armpurview"
)

type PurviewGenerator struct {
	AzureService
}

func (az *PurviewGenerator) listAccounts() ([]*armpurview.Account, error) {
	subscriptionID, resourceGroup, credential, clientOptions := az.getClientArgs()
	client, err := armpurview.NewAccountsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var resources []*armpurview.Account
	if resourceGroup != "" {
		pager := client.NewListByResourceGroupPager(resourceGroup, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			resources = append(resources, page.Value...)
		}
	} else {
		pager := client.NewListBySubscriptionPager(nil)
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

func (az *PurviewGenerator) AppendAccount(account *armpurview.Account) {
	az.AppendSimpleResource(*account.ID, *account.Name, "azurerm_purview_account")
}

func (az *PurviewGenerator) InitResources() error {
	accounts, err := az.listAccounts()
	if err != nil {
		return err
	}
	for _, account := range accounts {
		az.AppendAccount(account)
	}
	return nil
}
