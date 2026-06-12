// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v6/okta"
)

type SignOnPolicyGenerator struct {
	OktaService
}

type oktaPolicySummary struct {
	ID   string
	Name string
}

func (g SignOnPolicyGenerator) createResources(signOnPolicyList []okta.ListPolicies200ResponseInner) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, signOnPolicy := range signOnPolicyList {
		policy, ok := oktaPolicySummaryFromListPolicy(signOnPolicy)
		if !ok {
			continue
		}
		resourceName := normalizeResourceName(policy.Name)
		resourceType := "okta_policy_signon"

		resources = append(resources, terraformutils.NewSimpleResource(
			policy.ID,
			"policy_signon_"+resourceName,
			resourceType,
			"okta",
			[]string{}))
	}
	return resources
}

func (g *SignOnPolicyGenerator) InitResources() error {
	ctx, client, e := g.Client()
	if e != nil {
		return e
	}

	output, err := getSignOnPolicies(ctx, client)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}

func getSignOnPolicies(ctx context.Context, client *okta.APIClient) ([]okta.ListPolicies200ResponseInner, error) {
	policies, resp, err := client.PolicyAPI.ListPolicies(ctx).Type_("OKTA_SIGN_ON").Execute()
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

func oktaPolicySummaryFromListPolicy(policy okta.ListPolicies200ResponseInner) (oktaPolicySummary, bool) {
	switch {
	case policy.AccessPolicy != nil:
		return oktaPolicySummary{ID: policy.AccessPolicy.GetId(), Name: policy.AccessPolicy.GetName()}, true
	case policy.AuthenticatorEnrollmentPolicy != nil:
		return oktaPolicySummary{ID: policy.AuthenticatorEnrollmentPolicy.GetId(), Name: policy.AuthenticatorEnrollmentPolicy.GetName()}, true
	case policy.OktaSignOnPolicy != nil:
		return oktaPolicySummary{ID: policy.OktaSignOnPolicy.GetId(), Name: policy.OktaSignOnPolicy.GetName()}, true
	case policy.PasswordPolicy != nil:
		return oktaPolicySummary{ID: policy.PasswordPolicy.GetId(), Name: policy.PasswordPolicy.GetName()}, true
	case policy.ProfileEnrollmentPolicy != nil:
		return oktaPolicySummary{ID: policy.ProfileEnrollmentPolicy.GetId(), Name: policy.ProfileEnrollmentPolicy.GetName()}, true
	default:
		return oktaPolicySummary{}, false
	}
}
