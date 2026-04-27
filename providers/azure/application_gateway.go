// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/chenrui333/terraformer/terraformutils"
)

type ApplicationGatewayGenerator struct {
	AzureService
}

func (g ApplicationGatewayGenerator) createResources(ctx context.Context, iterator network.ApplicationGatewayListResultIterator) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	for iterator.NotDone() {
		applicationGateways := iterator.Value()
		resources = append(resources, terraformutils.NewSimpleResource(
			*applicationGateways.ID,
			*applicationGateways.Name,
			"azurerm_application_gateway",
			g.ProviderName,
			[]string{}))
		if err := iterator.NextWithContext(ctx); err != nil {
			log.Println(err)
			return resources, err
		}
	}
	return resources, nil
}

func (g *ApplicationGatewayGenerator) InitResources() error {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(providerConfig).CustomResourceManagerEndpoint
	applicationGatewaysClient := network.NewApplicationGatewaysClientWithBaseURI(resourceManagerEndpoint, subscriptionID)

	applicationGatewaysClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

	var (
		output network.ApplicationGatewayListResultIterator
		err    error
	)

	if rg := g.Args["resource_group"].(string); rg != "" {
		output, err = applicationGatewaysClient.ListComplete(ctx, rg)
	} else {
		output, err = applicationGatewaysClient.ListAllComplete(ctx)
	}
	if err != nil {
		return err
	}
	g.Resources, err = g.createResources(ctx, output)
	return err
}
