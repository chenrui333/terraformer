// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis/v3"
	"github.com/chenrui333/terraformer/terraformutils"
)

type RedisGenerator struct {
	AzureService
}

func (g *RedisGenerator) listRedisServers() ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	client, err := armredis.NewClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return nil, err
	}

	rg := g.Args["resource_group"].(string)
	if rg != "" {
		pager := client.NewListByResourceGroupPager(rg, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, redisServer := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*redisServer.ID,
					*redisServer.Name,
					"azurerm_redis_cache",
					g.ProviderName,
					[]string{}))
			}
		}
	} else {
		pager := client.NewListBySubscriptionPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, redisServer := range page.Value {
				resources = append(resources, terraformutils.NewSimpleResource(
					*redisServer.ID,
					*redisServer.Name,
					"azurerm_redis_cache",
					g.ProviderName,
					[]string{}))
			}
		}
	}

	return resources, nil
}

func (g *RedisGenerator) InitResources() error {
	functions := []func() ([]terraformutils.Resource, error){
		g.listRedisServers,
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
