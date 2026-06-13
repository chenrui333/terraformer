// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v6/okta"
)

type GroupRuleGenerator struct {
	OktaService
}

func (g GroupRuleGenerator) createResources(groupRuleList []okta.GroupRule) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, groupRule := range groupRuleList {
		resources = append(resources, terraformutils.NewSimpleResource(
			groupRule.GetId(),
			"grouprule_"+groupRule.GetName(),
			"okta_group_rule",
			"okta",
			[]string{}))
	}
	return resources
}

func (g *GroupRuleGenerator) InitResources() error {
	ctx, client, e := g.Client()
	if e != nil {
		return e
	}

	output, resp, err := client.GroupRuleAPI.ListGroupRules(ctx).Execute()
	if err != nil {
		return err
	}

	for resp.HasNextPage() {
		var nextGroupRuleSet []okta.GroupRule
		resp, err = resp.Next(&nextGroupRuleSet)
		if err != nil {
			return err
		}
		output = append(output, nextGroupRuleSet...)
	}

	g.Resources = g.createResources(output)
	return nil
}
