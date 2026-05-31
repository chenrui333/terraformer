// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"

	"github.com/auth0/go-auth0/v2/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	ResourceServerAllowEmptyValues = []string{}
)

type ResourceServerGenerator struct {
	Auth0Service
}

func (g ResourceServerGenerator) createResources(resourceServers []*management.ResourceServer) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, resourceServer := range resourceServers {
		if resourceServer == nil {
			return nil, auth0MissingResource("auth0_resource_server")
		}
		resourceName, err := auth0RequiredString("auth0_resource_server", "id", resourceServer.ID)
		if err != nil {
			return nil, err
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			auth0ResourceName(resourceServer.Name, resourceName),
			"auth0_resource_server",
			"auth0",
			ResourceServerAllowEmptyValues,
		))
	}
	return resources, nil
}

func (g *ResourceServerGenerator) InitResources() error {
	m, err := g.generateClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	page, err := m.ResourceServers.List(ctx, &management.ListResourceServerRequestParameters{})
	if err != nil {
		return err
	}
	list, err := auth0PageResults(ctx, page)
	if err != nil {
		return err
	}

	resources, err := g.createResources(list)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}
