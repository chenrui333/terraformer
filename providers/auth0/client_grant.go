// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"

	"github.com/auth0/go-auth0/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	ClientGrantAllowEmptyValues = []string{}
)

type ClientGrantGenerator struct {
	Auth0Service
}

func (g ClientGrantGenerator) createResources(clientGrantGrants []*management.ClientGrant) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, clientGrant := range clientGrantGrants {
		if clientGrant == nil {
			return nil, auth0MissingResource("auth0_client_grant")
		}
		resourceName, err := auth0RequiredString("auth0_client_grant", "id", clientGrant.ID)
		if err != nil {
			return nil, err
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			auth0ResourceName(clientGrant.ClientID, resourceName),
			"auth0_client_grant",
			"auth0",
			ClientGrantAllowEmptyValues,
		))
	}
	return resources, nil
}

func (g *ClientGrantGenerator) InitResources() error {
	m, err := g.generateClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	list := []*management.ClientGrant{}

	var page int
	for {
		l, err := m.ClientGrant.List(ctx, management.Page(page))
		if err != nil {
			return err
		}
		list = append(list, l.ClientGrants...)
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
