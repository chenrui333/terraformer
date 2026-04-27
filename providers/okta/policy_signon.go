// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
)

type SignOnPolicyGenerator struct {
	OktaService
}

func (g SignOnPolicyGenerator) createResources(signOnPolicyList []*okta.Policy) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, signOnPolicy := range signOnPolicyList {
		resourceName := normalizeResourceName(signOnPolicy.Name)
		resourceType := "okta_policy_signon"

		resources = append(resources, terraformutils.NewSimpleResource(
			signOnPolicy.Id,
			"policy_signon_"+resourceName,
			resourceType,
			"okta",
			[]string{}))
	}
	return resources
}

func (g *SignOnPolicyGenerator) InitResources() error {
	var output []*okta.Policy
	ctx, client, e := g.Client()
	if e != nil {
		return e
	}

	output, _ = getSignOnPolicies(ctx, client)
	g.Resources = g.createResources(output)
	return nil
}

func getSignOnPolicies(ctx context.Context, client *okta.Client) ([]*okta.Policy, error) {
	qp := query.NewQueryParams(query.WithType("OKTA_SIGN_ON"))
	var policies []*okta.Policy
	data, resp, err := client.Policy.ListPolicies(ctx, qp)
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var nextPolicies []*okta.Policy
		resp, _ = resp.Next(ctx, &nextPolicies)
		policies = append(policies, nextPolicies...)
	}
	for _, p := range data {
		policies = append(policies, p.(*okta.Policy))
	}

	return policies, nil
}
