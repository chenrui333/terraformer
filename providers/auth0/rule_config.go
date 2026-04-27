// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"gopkg.in/auth0.v5/management"
)

var (
	RuleConfigAllowEmptyValues = []string{}
)

type RuleConfigGenerator struct {
	Auth0Service
}

func (g RuleConfigGenerator) createResources(ruleConfigConfigs []*management.RuleConfig) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, ruleConfig := range ruleConfigConfigs {
		resourceName := *ruleConfig.Key
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			resourceName,
			"auth0_rule_config",
			"auth0",
			RuleConfigAllowEmptyValues,
		))
	}
	return resources
}

func (g *RuleConfigGenerator) InitResources() error {
	m := g.generateClient()

	list, err := m.RuleConfig.List()
	if err != nil {
		return err
	}

	g.Resources = g.createResources(list)
	return nil
}
