// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/chenrui333/terraformer/terraformutils"
)

type StorageAccountGenerator struct {
	AzureService
}

func (g StorageAccountGenerator) createResourcesByResourceGroup(ctx context.Context, client storage.AccountsClient, rg string) ([]terraformutils.Resource, error) {
	accountListResult, err := client.ListByResourceGroup(ctx, rg)
	if err != nil {
		return nil, err
	}
	var resources []terraformutils.Resource
	if accounts := accountListResult.Value; accounts != nil {
		for _, account := range *accounts {
			resources = append(resources, terraformutils.NewSimpleResource(
				*account.ID,
				*account.Name,
				"azurerm_storage_account",
				"azurerm",
				[]string{}))
		}
	}
	return resources, nil
}
func (g StorageAccountGenerator) createResources(ctx context.Context, client storage.AccountsClient) ([]terraformutils.Resource, error) {
	accountListResultIterator, err := client.ListComplete(ctx)
	if err != nil {
		return nil, err
	}
	var resources []terraformutils.Resource
	for accountListResultIterator.NotDone() {
		account := accountListResultIterator.Value()
		resources = append(resources, terraformutils.NewSimpleResource(
			*account.ID,
			*account.Name,
			"azurerm_storage_account",
			"azurerm",
			[]string{}))
		if err := accountListResultIterator.Next(); err != nil {
			log.Println(err)
			return resources, err
		}
	}
	return resources, nil
}

func (g *StorageAccountGenerator) InitResources() error {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(providerConfig).CustomResourceManagerEndpoint
	accountsClient := storage.NewAccountsClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	accountsClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)
	if rg := g.Args["resource_group"].(string); rg != "" {
		output, err := g.createResourcesByResourceGroup(ctx, accountsClient, rg)
		g.Resources = output
		return err
	}
	output, err := g.createResources(ctx, accountsClient)
	g.Resources = output
	return err
}
