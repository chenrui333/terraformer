// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v6/okta"
)

type IdpSocialGenerator struct {
	OktaService
}

func (g IdpSocialGenerator) createResources(idpSocialList []okta.IdentityProvider) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, idp := range idpSocialList {
		resources = append(resources, terraformutils.NewSimpleResource(
			idp.GetId(),
			"idp_"+normalizeResourceName(idp.GetType()+"_"+idp.GetName()),
			"okta_idp_social",
			"okta",
			[]string{}))
	}
	return resources
}

// Generate Terraform Resources from Okta API,
func (g *IdpSocialGenerator) InitResources() error {
	ctx, client, err := g.Client()
	if err != nil {
		return err
	}

	identityProviders, err := getIdpSocials(ctx, client)
	if err != nil {
		return err
	}

	g.Resources = g.createResources(identityProviders)
	return nil
}

func getIdpSocials(ctx context.Context, client *okta.APIClient) ([]okta.IdentityProvider, error) {
	idpSocialTypes := []string{"APPLE", "FACEBOOK", "GOOGLE", "LINKEDIN", "MICROSOFT"}
	var allIDPSocials []okta.IdentityProvider

	for _, idpSocialType := range idpSocialTypes {
		output, resp, err := client.IdentityProviderAPI.ListIdentityProviders(ctx).Type_(idpSocialType).Limit(1).Execute()
		if err != nil {
			return nil, err
		}

		for resp.HasNextPage() {
			var nextIdpSocialSet []okta.IdentityProvider
			resp, err = resp.Next(&nextIdpSocialSet)
			if err != nil {
				return nil, err
			}
			output = append(output, nextIdpSocialSet...)
		}

		allIDPSocials = append(allIDPSocials, output...)
	}

	return allIDPSocials, nil
}
