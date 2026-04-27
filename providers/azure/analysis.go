// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/analysisservices/mgmt/2017-08-01/analysisservices"
	"github.com/Azure/go-autorest/autorest"
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
	resourceManagerEndpoint := g.Args["config"].(providerConfig).CustomResourceManagerEndpoint
	AnalysisClient := analysisservices.NewServersClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	AnalysisClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

	var (
		servers analysisservices.Servers
		err     error
	)

	if rg := g.Args["resource_group"].(string); rg != "" {
		servers, err = AnalysisClient.ListByResourceGroup(ctx, rg)
	} else {
		servers, err = AnalysisClient.List(ctx)
	}
	if err != nil {
		return nil, err
	}
	for _, svr := range *servers.Value {
		resources = append(resources, terraformutils.NewSimpleResource(
			*svr.ID,
			*svr.Name,
			"azurerm_analysis_services_server",
			g.ProviderName,
			[]string{}))
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
