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

func (g ClientGrantGenerator) createResources(clientGrantGrants []*management.ClientGrant) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, clientGrant := range clientGrantGrants {
		resourceName := *clientGrant.ID
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			resourceName+"_"+*clientGrant.ClientID,
			"auth0_client_grant",
			"auth0",
			ClientGrantAllowEmptyValues,
		))
	}
	return resources
}

func (g *ClientGrantGenerator) InitResources() error {
	m := g.generateClient()
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

	g.Resources = g.createResources(list)
	return nil
}
