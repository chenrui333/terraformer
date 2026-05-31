// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v4"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	containerIDFormat = "https://%s.blob.core.windows.net/%s"
)

type StorageContainerGenerator struct {
	AzureService
}

func NewStorageContainerGenerator(subscriptionID string, credential azcore.TokenCredential, clientOptions *arm.ClientOptions, rg string) *StorageContainerGenerator {
	storageContainerGenerator := new(StorageContainerGenerator)
	storageContainerGenerator.Args = map[string]interface{}{}
	storageContainerGenerator.Args["config"] = providerConfig{SubscriptionID: subscriptionID}
	storageContainerGenerator.Args["credential"] = credential
	storageContainerGenerator.Args["clientOptions"] = clientOptions
	storageContainerGenerator.Args["resource_group"] = rg

	return storageContainerGenerator
}

func (g StorageContainerGenerator) ListBlobContainers() ([]terraformutils.Resource, error) {
	var containerResources []terraformutils.Resource
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)
	ctx := context.Background()

	blobContainersClient, err := armstorage.NewBlobContainersClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return containerResources, err
	}

	accounts, err := g.getStorageAccounts()
	if err != nil {
		return containerResources, err
	}

	for _, storageAccount := range accounts {
		parsedStorageAccountResourceID, err := ParseAzureResourceID(*storageAccount.ID)
		if err != nil {
			return containerResources, err
		}

		pager := blobContainersClient.NewListPager(parsedStorageAccountResourceID.ResourceGroup, *storageAccount.Name, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return containerResources, err
			}
			for _, containerItem := range page.Value {
				containerResources = append(containerResources,
					terraformutils.NewResource(
						fmt.Sprintf(containerIDFormat, *storageAccount.Name, *containerItem.Name),
						*containerItem.Name,
						"azurerm_storage_container",
						"azurerm",
						map[string]string{
							"storage_account_name": *storageAccount.Name,
							"name":                 *containerItem.Name,
						},
						[]string{},
						map[string]interface{}{}))
			}
		}
	}

	return containerResources, nil
}

func (g *StorageContainerGenerator) getStorageAccounts() ([]*armstorage.Account, error) {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	accountsClient, err := armstorage.NewAccountsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	var accounts []*armstorage.Account
	if rg := g.Args["resource_group"].(string); rg != "" {
		pager := accountsClient.NewListByResourceGroupPager(rg, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			accounts = append(accounts, page.Value...)
		}
	} else {
		pager := accountsClient.NewListPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			accounts = append(accounts, page.Value...)
		}
	}

	return accounts, nil
}

func (g *StorageContainerGenerator) InitResources() error {
	storageAccounts, err := g.ListBlobContainers()
	if err != nil {
		return err
	}

	g.Resources = storageAccounts

	return nil
}
