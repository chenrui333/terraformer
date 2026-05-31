// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"

	"github.com/auth0/go-auth0/v2/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	RuleConfigAllowEmptyValues = []string{}
)

type RuleConfigGenerator struct {
	Auth0Service
}

func (g RuleConfigGenerator) createResources(ruleConfigConfigs []*management.RulesConfig) []terraformutils.Resource {
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
	m, err := g.generateClient()
	if err != nil {
		return err
	}
	ctx := context.Background()

	list, err := m.RulesConfigs.List(ctx)
	if err != nil {
		return err
	}

	g.Resources = g.createResources(list)
	return nil
}
