// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/security/armsecurity"
	"github.com/chenrui333/terraformer/terraformutils"
)

type SecurityCenterContactGenerator struct {
	AzureService
}

func (g SecurityCenterContactGenerator) listContacts() ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	if rg := g.Args["resource_group"].(string); rg != "" {
		return resources, nil
	}

	contactsClient, err := armsecurity.NewContactsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return resources, err
	}

	pager := contactsClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return resources, err
		}
		for _, contact := range page.Value {
			resources = append(resources, terraformutils.NewSimpleResource(
				*contact.ID,
				*contact.Name,
				"azurerm_security_center_contact",
				g.ProviderName,
				[]string{}))
		}
	}

	return resources, nil
}

func (g *SecurityCenterContactGenerator) InitResources() error {
	resources, err := g.listContacts()
	if err != nil {
		return err
	}

	g.Resources = append(g.Resources, resources...)

	return nil
}
