// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"gopkg.in/auth0.v5/management"
)

var (
	RuleAllowEmptyValues = []string{}
)

type RuleGenerator struct {
	Auth0Service
}

func (g RuleGenerator) createResources(rules []*management.Rule) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, rule := range rules {
		resourceName := *rule.ID
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			resourceName+"_"+*rule.Name,
			"auth0_rule",
			"auth0",
			RuleAllowEmptyValues,
		))
	}
	return resources
}

func (g *RuleGenerator) InitResources() error {
	m := g.generateClient()
	list := []*management.Rule{}

	var page int
	for {
		l, err := m.Rule.List(management.Page(page))
		if err != nil {
			return err
		}
		list = append(list, l.Rules...)
		if !l.HasNext() {
			break
		}
		page++
	}

	g.Resources = g.createResources(list)
	return nil
}
