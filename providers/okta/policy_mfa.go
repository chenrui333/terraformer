// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v6/okta"
)

type MFAPolicyGenerator struct {
	OktaService
}

func (g MFAPolicyGenerator) createResources(mfaPolicyList []okta.ListPolicies200ResponseInner) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, mfaPolicy := range mfaPolicyList {
		policy, ok := oktaPolicySummaryFromListPolicy(mfaPolicy)
		if !ok {
			continue
		}
		resourceName := normalizeResourceName(policy.Name)
		resourceType := "okta_policy_mfa"
		if policy.Name == "Default Policy" {
			resourceType = "okta_policy_mfa_default"
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			policy.ID,
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

func getMFAPolicies(ctx context.Context, client *okta.APIClient) ([]okta.ListPolicies200ResponseInner, error) {
	policies, resp, err := client.PolicyAPI.ListPolicies(ctx).Type_("MFA_ENROLL").Execute()
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
