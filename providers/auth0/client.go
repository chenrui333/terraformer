// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"

	"github.com/auth0/go-auth0/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	ClientAllowEmptyValues = []string{}
)

type ClientGenerator struct {
	Auth0Service
}

func (g ClientGenerator) createResources(clients []*management.Client) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, client := range clients {
		resourceName := *client.ClientID
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			resourceName+"_"+*client.Name,
			"auth0_client",
			"auth0",
			ClientAllowEmptyValues,
		))
	}
	return resources
}

func (g *ClientGenerator) InitResources() error {
	m := g.generateClient()
	ctx := context.Background()
	list := []*management.Client{}

	var page int
	for {
		l, err := m.Client.List(ctx, management.Page(page))
		if err != nil {
			return err
		}
		list = append(list, l.Clients...)
		if !l.HasNext() {
			break
		}
		page++
	}

	g.Resources = g.createResources(list)
	return nil
}
