// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"gopkg.in/auth0.v5/management"
)

var (
	CustomDomainAllowEmptyValues = []string{}
)

type CustomDomainGenerator struct {
	Auth0Service
}

func (g CustomDomainGenerator) createResources(customDomains []*management.CustomDomain) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, CustomDomain := range customDomains {
		resourceName := *CustomDomain.ID
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			resourceName+"_"+*CustomDomain.Domain,
			"auth0_custom_domain",
			"auth0",
			CustomDomainAllowEmptyValues,
		))
	}
	return resources
}

func (g *CustomDomainGenerator) InitResources() error {
	m := g.generateClient()
	list, err := m.CustomDomain.List()
	if err != nil {
		return err
	}

	g.Resources = g.createResources(list)
	return nil
}
