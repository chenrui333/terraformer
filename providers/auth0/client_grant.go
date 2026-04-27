// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"gopkg.in/auth0.v5/management"
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
	list := []*management.ClientGrant{}

	var page int
	for {
		l, err := m.ClientGrant.List(management.Page(page))
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
