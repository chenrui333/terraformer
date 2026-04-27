// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/chenrui333/terraformer/terraformutils"
)

type VirtualMachineGenerator struct {
	AzureService
}

func (g VirtualMachineGenerator) createResources(virtualMachineListResultIterator compute.VirtualMachineListResultIterator) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	for virtualMachineListResultIterator.NotDone() {
		vm := virtualMachineListResultIterator.Value()
		var newResource terraformutils.Resource
		if vm.OsProfile == nil {
			if vm.StorageProfile.OsDisk.OsType == "Windows" {
				newResource = terraformutils.NewSimpleResource(
					*vm.ID,
					*vm.Name,
					"azurerm_windows_virtual_machine",
					"azurerm",
					[]string{})
			} else {
				newResource = terraformutils.NewSimpleResource(
					*vm.ID,
					*vm.Name,
					"azurerm_linux_virtual_machine",
					"azurerm",
					[]string{})
			}
		} else {
			if vm.OsProfile.WindowsConfiguration != nil {
				newResource = terraformutils.NewSimpleResource(
					*vm.ID,
					*vm.Name,
					"azurerm_windows_virtual_machine",
					"azurerm",
					[]string{})
			} else {
				newResource = terraformutils.NewSimpleResource(
					*vm.ID,
					*vm.Name,
					"azurerm_linux_virtual_machine",
					"azurerm",
					[]string{})
			}
		}

		resources = append(resources, newResource)
		if err := virtualMachineListResultIterator.Next(); err != nil {
			log.Println(err)
			return resources, err
		}
	}
	return resources, nil
}

func (g *VirtualMachineGenerator) InitResources() error {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(providerConfig).CustomResourceManagerEndpoint
	vmClient := compute.NewVirtualMachinesClientWithBaseURI(resourceManagerEndpoint, subscriptionID)

	vmClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

	var (
		output compute.VirtualMachineListResultIterator
		err    error
	)
	if rg := g.Args["resource_group"].(string); rg != "" {
		output, err = vmClient.ListComplete(ctx, rg)
	} else {
		output, err = vmClient.ListAllComplete(ctx)
	}
	if err != nil {
		return err
	}
	g.Resources, err = g.createResources(output)
	return err
}
