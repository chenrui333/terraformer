// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/security/armsecurity"
	"github.com/chenrui333/terraformer/terraformutils"
)

type SecurityCenterSubscriptionPricingGenerator struct {
	AzureService
}

func (g SecurityCenterSubscriptionPricingGenerator) listSubscriptionPricing() ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	if rg := g.Args["resource_group"].(string); rg != "" {
		return resources, nil
	}

	pricingsClient, err := armsecurity.NewPricingsClient(credential, clientOptions)
	if err != nil {
		return resources, err
	}

	scopeID := "subscriptions/" + subscriptionID
	pricingList, err := pricingsClient.List(ctx, scopeID, nil)
	if err != nil {
		return resources, err
	}

	for _, pricing := range pricingList.Value {
		resources = append(resources, terraformutils.NewSimpleResource(
			*pricing.ID,
			*pricing.Name,
			"azurerm_security_center_subscription_pricing",
			g.ProviderName,
			[]string{}))
	}

	return resources, nil
}

func (g *SecurityCenterSubscriptionPricingGenerator) InitResources() error {
	resources, err := g.listSubscriptionPricing()
	if err != nil {
		return err
	}

	g.Resources = append(g.Resources, resources...)

	return nil
}
