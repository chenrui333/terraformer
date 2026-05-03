// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"

	"github.com/auth0/go-auth0/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	RuleAllowEmptyValues = []string{}
)

type RuleGenerator struct {
	Auth0Service
}

func (g RuleGenerator) createResources(rules []*management.Rule) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, rule := range rules {
		if rule == nil {
			return nil, auth0MissingResource("auth0_rule")
		}
		resourceName, err := auth0RequiredString("auth0_rule", "id", rule.ID)
		if err != nil {
			return nil, err
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			auth0ResourceName(rule.Name, resourceName),
			"auth0_rule",
			"auth0",
			RuleAllowEmptyValues,
		))
	}
	return resources, nil
}

func (g *RuleGenerator) InitResources() error {
	m, err := g.generateClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	list := []*management.Rule{}

	var page int
	for {
		l, err := m.Rule.List(ctx, management.Page(page))
		if err != nil {
			return err
		}
		list = append(list, l.Rules...)
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
