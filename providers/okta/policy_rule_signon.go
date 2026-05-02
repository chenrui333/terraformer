// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/terraform-provider-okta/sdk"
)

type SignOnPolicyRuleGenerator struct {
	OktaService
}

func (g SignOnPolicyRuleGenerator) createResources(signOnPolicyRuleList []sdk.SdkPolicyRule, policyID string, policyName string) []terraformutils.Resource {
	var resources []terraformutils.Resource

	for _, policyRule := range signOnPolicyRuleList {
		resources = append(resources, terraformutils.NewResource(
			policyRule.Id,
			"policyrule_signon_"+normalizeResourceName(policyName+"_"+policyRule.Name),
			"okta_policy_rule_signon",
			"okta",
			map[string]string{
				"policy_id": policyID,
			},
			[]string{},
			map[string]interface{}{},
		))
	}

	return resources
}

func (g *SignOnPolicyRuleGenerator) InitResources() error {
	var resources []terraformutils.Resource

	ctx, client, e := g.Client()
	if e != nil {
		return e
	}

	signOnPolicies, err := getSignOnPolicies(ctx, client)
	if err != nil {
		return err
	}

	for _, policy := range signOnPolicies {
		output, err := getSignOnPolicyRules(g, policy.Id)
		if err != nil {
			return err
		}

		resources = append(resources, g.createResources(output, policy.Id, policy.Name)...)
	}

	g.Resources = resources
	return nil
}

func getSignOnPolicyRules(g *SignOnPolicyRuleGenerator, policyID string) ([]sdk.SdkPolicyRule, error) {
	ctx, client, e := g.APISupplementClient()
	if e != nil {
		return nil, e
	}

	output, resp, err := client.ListPolicyRules(ctx, policyID)
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var nextPolicySet []sdk.SdkPolicyRule
		resp, err = resp.Next(ctx, &nextPolicySet)
		if err != nil {
			return nil, err
		}
		output = append(output, nextPolicySet...)
	}

	return output, nil
}
