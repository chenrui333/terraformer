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

func (g ClientGenerator) createResources(clients []*management.Client) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, client := range clients {
		if client == nil {
			return nil, auth0MissingResource("auth0_client")
		}
		resourceName, err := auth0RequiredString("auth0_client", "client_id", client.ClientID)
		if err != nil {
			return nil, err
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			auth0ResourceName(client.Name, resourceName),
			"auth0_client",
			"auth0",
			ClientAllowEmptyValues,
		))
	}
	return resources, nil
}

func (g *ClientGenerator) InitResources() error {
	m, err := g.generateClient()
	if err != nil {
		return err
	}
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

	resources, err := g.createResources(list)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}
