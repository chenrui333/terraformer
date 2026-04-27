// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package azure

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/redis/mgmt/2018-03-01/redis"
	"github.com/chenrui333/terraformer/terraformutils"
)

type RedisGenerator struct {
	AzureService
}

func (g *RedisGenerator) listRedisServers() ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	ctx := context.Background()
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(providerConfig).CustomResourceManagerEndpoint
	RedisClient := redis.NewClientWithBaseURI(resourceManagerEndpoint, subscriptionID)

	redisServersIterator, err := RedisClient.ListComplete(ctx)
	if err != nil {
		return nil, err
	}

	for redisServersIterator.NotDone() {
		redisServer := redisServersIterator.Value()
		resources = append(resources, terraformutils.NewSimpleResource(
			*redisServer.ID,
			*redisServer.Name,
			"azurerm_redis_cache",
			g.ProviderName,
			[]string{}))

		if err := redisServersIterator.Next(); err != nil {
			log.Println(err)
			break
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
