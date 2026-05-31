// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v6/okta"
)

type AppSignOnPolicyRuleGenerator struct {
	OktaService
}

func (g AppSignOnPolicyRuleGenerator) createResources(signOnPolicyRuleList []okta.ListPolicyRules200ResponseInner, policyID string) []terraformutils.Resource {
	var resources []terraformutils.Resource

	for _, policyRule := range signOnPolicyRuleList {
		if policyRule.AccessPolicyRule == nil {
			continue
		}

		resourceName := normalizeResourceNameWithRandom(policyRule.AccessPolicyRule.GetName())

		resources = append(resources, terraformutils.NewResource(
			policyRule.AccessPolicyRule.GetId(),
			resourceName,
			"okta_app_signon_policy_rule",
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

func (g *AppSignOnPolicyRuleGenerator) InitResources() error {
	ctx, client, err := g.ClientV6()
	if err != nil {
		return err
	}

	policies, err := getAppSignOnPolicies(ctx, client)
	if err != nil {
		return err
	}

	var allResources []terraformutils.Resource

	for _, policy := range policies {
		if policy.AccessPolicy == nil {
			continue
		}

		policyID := policy.AccessPolicy.GetId()

		policyRules, err := getAppSignOnPolicyRules(ctx, client, policyID)
		if err != nil {
			return err
		}

		resources := g.createResources(policyRules, policyID)

		allResources = append(allResources, resources...)
	}

	g.Resources = allResources

	return nil
}

func getAppSignOnPolicyRules(ctx context.Context, client *okta.APIClient, policyID string) ([]okta.ListPolicyRules200ResponseInner, error) {
	policyRules, _, err := client.PolicyAPI.ListPolicyRules(ctx, policyID).Execute()
	if err != nil {
		return nil, err
	}
	return policyRules, nil
}
