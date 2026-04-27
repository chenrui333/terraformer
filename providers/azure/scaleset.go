// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-12-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/chenrui333/terraformer/terraformutils"
)

type ScaleSetGenerator struct {
	AzureService
}

func (g ScaleSetGenerator) createResourcesByResourceGroup(ctx context.Context, client compute.VirtualMachineScaleSetsClient, rg string) ([]terraformutils.Resource, error) {
	scaleSetIterator, err := client.ListComplete(ctx, rg)
	if err != nil {
		return nil, err
	}
	var resources []terraformutils.Resource
	for scaleSetIterator.NotDone() {
		scaleSet := scaleSetIterator.Value()
		newResource := terraformutils.NewSimpleResource(
			*scaleSet.ID,
			*scaleSet.Name,
			"azurerm_virtual_machine_scale_set",
			"azurerm",
			[]string{})
		resources = append(resources, newResource)
		if err := scaleSetIterator.Next(); err != nil {
			log.Println(err)
			return resources, err
		}
	}
	return resources, nil
}

func (g ScaleSetGenerator) createResources(ctx context.Context, client compute.VirtualMachineScaleSetsClient) ([]terraformutils.Resource, error) {
	scaleSetIterator, err := client.ListAllComplete(ctx)
	if err != nil {
		return nil, err
	}
	var resources []terraformutils.Resource
	for scaleSetIterator.NotDone() {
		scaleSet := scaleSetIterator.Value()
		newResource := terraformutils.NewSimpleResource(
			*scaleSet.ID,
			*scaleSet.Name,
			"azurerm_virtual_machine_scale_set",
			"azurerm",
			[]string{})
		resources = append(resources, newResource)
		if err := scaleSetIterator.Next(); err != nil {
			log.Println(err)
			return resources, err
		}
	}
	return resources, nil
}

func (g *ScaleSetGenerator) InitResources() error {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(providerConfig).CustomResourceManagerEndpoint
	ScaleSetClient := compute.NewVirtualMachineScaleSetsClientWithBaseURI(resourceManagerEndpoint, subscriptionID)

	ScaleSetClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

	if rg := g.Args["resource_group"].(string); rg != "" {
		var err error
		g.Resources, err = g.createResourcesByResourceGroup(ctx, ScaleSetClient, rg)
		return err
	}
	var err error
	g.Resources, err = g.createResources(ctx, ScaleSetClient)
	return err
}
