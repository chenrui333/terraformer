// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v6/okta"
)

type PasswordPolicyGenerator struct {
	OktaService
}

func (g PasswordPolicyGenerator) createResources(passwordPolicyList []okta.ListPolicies200ResponseInner) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, passwordPolicy := range passwordPolicyList {
		policy, ok := oktaPolicySummaryFromListPolicy(passwordPolicy)
		if !ok {
			continue
		}
		resourceName := normalizeResourceName(policy.Name)
		resourceType := "okta_policy_password"
		if policy.Name == "Default Policy" {
			resourceType = "okta_policy_password_default"
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			policy.ID,
			"policy_password_"+resourceName,
			resourceType,
			"okta",
			[]string{}))
	}
	return resources
}

func (g *PasswordPolicyGenerator) InitResources() error {
	ctx, client, e := g.Client()
	if e != nil {
		return e
	}

	output, err := getPasswordPolicies(ctx, client)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}

func getPasswordPolicies(ctx context.Context, client *okta.APIClient) ([]okta.ListPolicies200ResponseInner, error) {
	policies, resp, err := client.PolicyAPI.ListPolicies(ctx).Type_("PASSWORD").Execute()
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var nextPolicies []okta.ListPolicies200ResponseInner
		resp, err = resp.Next(&nextPolicies)
		if err != nil {
			return nil, err
		}
		policies = append(policies, nextPolicies...)
	}

	return policies, nil
}
