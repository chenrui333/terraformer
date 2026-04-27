// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/chenrui333/terraformer/terraformutils"
)

type DiskGenerator struct {
	AzureService
}

func (g DiskGenerator) createResources(diskListIterator compute.DiskListIterator) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	for diskListIterator.NotDone() {
		disk := diskListIterator.Value()
		resources = append(resources, terraformutils.NewSimpleResource(
			*disk.ID,
			*disk.Name,
			"azurerm_managed_disk",
			"azurerm",
			[]string{}))
		if err := diskListIterator.Next(); err != nil {
			log.Println(err)
			return resources, err
		}
	}
	return resources, nil
}

func (g *DiskGenerator) InitResources() error {
	ctx := context.Background()
	resourceManagerEndpoint := g.Args["config"].(providerConfig).CustomResourceManagerEndpoint
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	disksClient := compute.NewDisksClientWithBaseURI(resourceManagerEndpoint, subscriptionID)

	disksClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

	var (
		output compute.DiskListIterator
		err    error
	)

	if rg := g.Args["resource_group"].(string); rg != "" {
		output, err = disksClient.ListByResourceGroupComplete(ctx, rg)
	} else {
		output, err = disksClient.ListComplete(ctx)
	}
	if err != nil {
		return err
	}
	g.Resources, err = g.createResources(output)
	return err
}
