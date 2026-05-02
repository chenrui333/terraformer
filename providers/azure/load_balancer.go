// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/chenrui333/terraformer/terraformutils"
)

type LoadBalancerGenerator struct {
	AzureService
}

func (g *LoadBalancerGenerator) listLoadBalancerProbes(ctx context.Context, resourceGroupName string, loadBalancerName string) ([]terraformutils.Resource, error) {
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	client, err := armnetwork.NewLoadBalancerProbesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	var resources []terraformutils.Resource
	pager := client.NewListPager(resourceGroupName, loadBalancerName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, probe := range page.Value {
			// NOTE:
			// This works out the loadBalancer resource id from current probe
			// /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/group1/providers/Microsoft.Network/loadBalancers/lb1
			// /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/group1/providers/Microsoft.Network/loadBalancers/lb1/probes/probe1
			//
			// As the related data_source in azurerm provider works by starting to look up with loadbalancer_id
			// https://github.com/terraform-providers/terraform-provider-azurerm/blob/v2.18.0/azurerm/internal/services/network/lb_probe_resource.go#L186
			re := regexp.MustCompile(`/probes/.*$`)
			loadBalancerID := re.ReplaceAllLiteralString(*probe.ID, "")
			resources = append(resources, terraformutils.NewResource(
				*probe.ID,
				*probe.Name,
				"azurerm_lb_probe",
				g.ProviderName,
				map[string]string{
					"loadbalancer_id": loadBalancerID,
				},
				[]string{},
				map[string]interface{}{},
			))
		}
	}

	return resources, nil
}

func (g *LoadBalancerGenerator) listInboundNatRules(ctx context.Context, resourceGroupName string, loadBalancerName string) ([]terraformutils.Resource, error) {
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	client, err := armnetwork.NewInboundNatRulesClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	var resources []terraformutils.Resource
	pager := client.NewListPager(resourceGroupName, loadBalancerName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, rule := range page.Value {
			// NOTE:
			// Similar to above explanation, work out loadbalancer_id for azurerm datasource impl
			re := regexp.MustCompile(`/inboundNatRules/.*$`)
			loadBalancerID := re.ReplaceAllLiteralString(*rule.ID, "")
			resources = append(resources, terraformutils.NewResource(
				*rule.ID,
				*rule.Name,
				"azurerm_lb_nat_rule",
				g.ProviderName,
				map[string]string{
					"loadbalancer_id": loadBalancerID,
				},
				[]string{},
				map[string]interface{}{},
			))
		}
	}

	return resources, nil
}

func (g *LoadBalancerGenerator) listLoadBalancerBackendAddressPools(ctx context.Context, resourceGroupName string, loadBalancerName string) ([]terraformutils.Resource, error) {
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	client, err := armnetwork.NewLoadBalancerBackendAddressPoolsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	var resources []terraformutils.Resource
	pager := client.NewListPager(resourceGroupName, loadBalancerName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, pool := range page.Value {
			// NOTE:
			// Similar to above explanation, work out loadbalancer_id for azurerm datasource impl
			re := regexp.MustCompile(`/backendAddressPools/.*$`)
			loadBalancerID := re.ReplaceAllLiteralString(*pool.ID, "")
			resources = append(resources, terraformutils.NewResource(
				*pool.ID,
				*pool.Name,
				"azurerm_lb_backend_address_pool",
				g.ProviderName,
				map[string]string{
					"loadbalancer_id": loadBalancerID,
				},
				[]string{},
				map[string]interface{}{},
			))
		}
	}

	return resources, nil
}

func (g *LoadBalancerGenerator) listAndAddForLoadBalancers() ([]terraformutils.Resource, error) {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	client, err := armnetwork.NewLoadBalancersClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	rg := g.Args["resource_group"].(string)
	var loadBalancers []*armnetwork.LoadBalancer
	if rg != "" {
		pager := client.NewListPager(rg, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			loadBalancers = append(loadBalancers, page.Value...)
		}
	} else {
		pager := client.NewListAllPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			loadBalancers = append(loadBalancers, page.Value...)
		}
	}

	var resources []terraformutils.Resource
	for _, lb := range loadBalancers {
		resources = append(resources, terraformutils.NewSimpleResource(
			*lb.ID,
			*lb.Name,
			"azurerm_lb",
			g.ProviderName,
			[]string{}))

		id, err := ParseAzureResourceID(*lb.ID)
		if err != nil {
			return nil, err
		}

		probes, err := g.listLoadBalancerProbes(ctx, id.ResourceGroup, *lb.Name)
		if err != nil {
			return nil, err
		}
		resources = append(resources, probes...)

		inboundNatRules, err := g.listInboundNatRules(ctx, id.ResourceGroup, *lb.Name)
		if err != nil {
			return nil, err
		}
		resources = append(resources, inboundNatRules...)

		backendAddressPools, err := g.listLoadBalancerBackendAddressPools(ctx, id.ResourceGroup, *lb.Name)
		if err != nil {
			return nil, err
		}
		resources = append(resources, backendAddressPools...)
	}

	return resources, nil
}

func (g *LoadBalancerGenerator) InitResources() error {
	functions := []func() ([]terraformutils.Resource, error){
		g.listAndAddForLoadBalancers,
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
