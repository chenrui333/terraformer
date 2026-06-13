// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v6/okta"
)

type IdpSAMLGenerator struct {
	OktaService
}

func (g IdpSAMLGenerator) createResources(idpSAMLList []okta.IdentityProvider) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, idp := range idpSAMLList {
		resources = append(resources, terraformutils.NewSimpleResource(
			idp.GetId(),
			"idp_"+normalizeResourceName(idp.GetType()+"_"+idp.GetName()),
			"okta_idp_saml",
			"okta",
			[]string{}))
	}
	return resources
}

func (g *IdpSAMLGenerator) InitResources() error {
	ctx, client, err := g.Client()
	if err != nil {
		return err
	}

	identityProviders, err := getIdpSAML(ctx, client)
	if err != nil {
		return err
	}

	g.Resources = g.createResources(identityProviders)
	return nil
}

func getIdpSAML(ctx context.Context, client *okta.APIClient) ([]okta.IdentityProvider, error) {
	output, resp, err := client.IdentityProviderAPI.ListIdentityProviders(ctx).Type_("SAML2").Limit(1).Execute()
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var nextIdpSAMLSet []okta.IdentityProvider
		resp, err = resp.Next(&nextIdpSAMLSet)
		if err != nil {
			return nil, err
		}
		output = append(output, nextIdpSAMLSet...)
	}

	return output, nil
}
