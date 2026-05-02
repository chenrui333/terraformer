// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/analysisservices/armanalysisservices"
	"github.com/chenrui333/terraformer/terraformutils"
)

type AnalysisGenerator struct {
	AzureService
}

func (g *AnalysisGenerator) listServiceServers() ([]terraformutils.Resource, error) {
	log.Println("\tImporting Service Servers")
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	client, err := armanalysisservices.NewServersClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	rg := g.Args["resource_group"].(string)
	if rg != "" {
		pager := client.NewListByResourceGroupPager(rg, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, svr := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*svr.ID,
					*svr.Name,
					"azurerm_analysis_services_server",
					g.ProviderName,
					[]string{}))
			}
		}
	} else {
		pager := client.NewListPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, svr := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*svr.ID,
					*svr.Name,
					"azurerm_analysis_services_server",
					g.ProviderName,
					[]string{}))
			}
		}
	}

	return resources, nil
}

func (g *AnalysisGenerator) InitResources() error {
	functions := []func() ([]terraformutils.Resource, error){
		g.listServiceServers,
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
