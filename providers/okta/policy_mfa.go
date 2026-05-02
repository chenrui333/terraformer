// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
)

type MFAPolicyGenerator struct {
	OktaService
}

func (g MFAPolicyGenerator) createResources(mfaPolicyList []*okta.Policy) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, mfaPolicy := range mfaPolicyList {
		resourceName := normalizeResourceName(mfaPolicy.Name)
		resourceType := "okta_policy_mfa"
		if mfaPolicy.Name == "Default Policy" {
			resourceType = "okta_policy_mfa_default"
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			mfaPolicy.Id,
			"policy_mfa_"+resourceName,
			resourceType,
			"okta",
			[]string{}))
	}
	return resources
}

func (g *MFAPolicyGenerator) InitResources() error {
	ctx, client, e := g.Client()
	if e != nil {
		return e
	}

	output, err := getMFAPolicies(ctx, client)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}

func getMFAPolicies(ctx context.Context, client *okta.Client) ([]*okta.Policy, error) {
	qp := query.NewQueryParams(query.WithType("MFA_ENROLL"))
	var policies []*okta.Policy
	data, resp, err := client.Policy.ListPolicies(ctx, qp)
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var nextPolicies []*okta.Policy
		resp, err = resp.Next(ctx, &nextPolicies)
		if err != nil {
			return nil, err
		}
		policies = append(policies, nextPolicies...)
	}
	for _, p := range data {
		policies = append(policies, p.(*okta.Policy))
	}

	return policies, nil
}
