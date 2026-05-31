// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v8"
	"github.com/chenrui333/terraformer/terraformutils"
)

type VirtualMachineGenerator struct {
	AzureService
}

func (g *VirtualMachineGenerator) InitResources() error {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	client, err := armcompute.NewVirtualMachinesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return err
	}

	rg := g.Args["resource_group"].(string)
	var vms []*armcompute.VirtualMachine
	if rg != "" {
		pager := client.NewListPager(rg, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return err
			}
			vms = append(vms, page.Value...)
		}
	} else {
		pager := client.NewListAllPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return err
			}
			vms = append(vms, page.Value...)
		}
	}

	var resources []terraformutils.Resource
	for _, vm := range vms {
		isWindows := false
		if vm.Properties != nil {
			if vm.Properties.OSProfile == nil {
				if vm.Properties.StorageProfile != nil &&
					vm.Properties.StorageProfile.OSDisk != nil &&
					vm.Properties.StorageProfile.OSDisk.OSType != nil &&
					*vm.Properties.StorageProfile.OSDisk.OSType == armcompute.OperatingSystemTypesWindows {
					isWindows = true
				}
			} else if vm.Properties.OSProfile.WindowsConfiguration != nil {
				isWindows = true
			}
		}

		resourceType := "azurerm_linux_virtual_machine"
		if isWindows {
			resourceType = "azurerm_windows_virtual_machine"
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			*vm.ID, *vm.Name, resourceType, "azurerm", []string{}))
	}
	g.Resources = resources
	return nil
}
