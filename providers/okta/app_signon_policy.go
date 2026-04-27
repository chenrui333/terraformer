// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v5/okta"
)

type AppSignOnPolicyGenerator struct {
	OktaService
}

func (g AppSignOnPolicyGenerator) createResources(policies []okta.ListPolicies200ResponseInner) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, policy := range policies {
		if policy.AccessPolicy == nil {
			continue
		}

		resourceName := normalizeResourceNameWithRandom(policy.AccessPolicy.GetName())
		resourceID := policy.AccessPolicy.GetId()

		resources = append(resources, terraformutils.NewSimpleResource(
			resourceID,
			resourceName,
			"okta_app_signon_policy",
			"okta",
			[]string{}))
	}
	return resources
}

func (g *AppSignOnPolicyGenerator) InitResources() error {
	ctx, client, err := g.ClientV5()
	if err != nil {
		return err
	}

	policies, err := getAppSignOnPolicies(ctx, client)
	if err != nil {
		return err
	}

	g.Resources = g.createResources(policies)
	return nil
}

func getAppSignOnPolicies(ctx context.Context, client *okta.APIClient) ([]okta.ListPolicies200ResponseInner, error) {
	policies, _, err := client.PolicyAPI.ListPolicies(ctx).Type_("ACCESS_POLICY").Execute()
	if err != nil {
		return nil, err
	}
	return policies, nil
}
