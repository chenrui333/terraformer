// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"

	"github.com/auth0/go-auth0/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	CustomDomainAllowEmptyValues = []string{}
)

type CustomDomainGenerator struct {
	Auth0Service
}

func (g CustomDomainGenerator) createResources(customDomains []*management.CustomDomain) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, customDomain := range customDomains {
		if customDomain == nil {
			return nil, auth0MissingResource("auth0_custom_domain")
		}
		resourceName, err := auth0RequiredString("auth0_custom_domain", "id", customDomain.ID)
		if err != nil {
			return nil, err
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			auth0ResourceName(customDomain.Domain, resourceName),
			"auth0_custom_domain",
			"auth0",
			CustomDomainAllowEmptyValues,
		))
	}
	return resources, nil
}

func (g *CustomDomainGenerator) InitResources() error {
	m, err := g.generateClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	list, err := m.CustomDomain.List(ctx)
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
