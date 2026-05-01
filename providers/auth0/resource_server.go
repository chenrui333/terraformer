// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"

	"github.com/auth0/go-auth0/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	ResourceServerAllowEmptyValues = []string{}
)

type ResourceServerGenerator struct {
	Auth0Service
}

func (g ResourceServerGenerator) createResources(resourceServers []*management.ResourceServer) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, resourceServer := range resourceServers {
		resourceName := *resourceServer.ID
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			resourceName+"_"+*resourceServer.Name,
			"auth0_resource_server",
			"auth0",
			ResourceServerAllowEmptyValues,
		))
	}
	return resources
}

func (g *ResourceServerGenerator) InitResources() error {
	m := g.generateClient()
	ctx := context.Background()
	list := []*management.ResourceServer{}

	var page int
	for {
		l, err := m.ResourceServer.List(ctx, management.Page(page))
		if err != nil {
			return err
		}
		list = append(list, l.ResourceServers...)
		if !l.HasNext() {
			break
		}
		page++
	}

	g.Resources = g.createResources(list)
	return nil
}
