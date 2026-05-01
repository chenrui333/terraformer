// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/chenrui333/terraformer/terraformutils"
)

type ApplicationGatewayGenerator struct {
	AzureService
}

func (g *ApplicationGatewayGenerator) InitResources() error {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	client, err := armnetwork.NewApplicationGatewaysClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return err
	}

	rg := g.Args["resource_group"].(string)
	var gateways []*armnetwork.ApplicationGateway
	if rg != "" {
		pager := client.NewListPager(rg, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return err
			}
			gateways = append(gateways, page.Value...)
		}
	} else {
		pager := client.NewListAllPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return err
			}
			gateways = append(gateways, page.Value...)
		}
	}

	var resources []terraformutils.Resource
	for _, gw := range gateways {
		resources = append(resources, terraformutils.NewSimpleResource(
			*gw.ID,
			*gw.Name,
			"azurerm_application_gateway",
			g.ProviderName,
			[]string{}))
	}
	g.Resources = resources
	return nil
}
