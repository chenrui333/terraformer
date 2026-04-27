// Copyright 2020 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
