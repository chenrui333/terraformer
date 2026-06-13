// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v6/okta"
)

type IdpOIDCGenerator struct {
	OktaService
}

func (g IdpOIDCGenerator) createResources(idpOIDCList []okta.IdentityProvider) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, idp := range idpOIDCList {
		resources = append(resources, terraformutils.NewSimpleResource(
			idp.GetId(),
			"idp_"+normalizeResourceName(idp.GetType()+"_"+idp.GetName()),
			"okta_idp_oidc",
			"okta",
			[]string{}))
	}
	return resources
}

func (g *IdpOIDCGenerator) InitResources() error {
	ctx, client, err := g.Client()
	if err != nil {
		return err
	}

	identityProviders, err := getIdpOIDC(ctx, client)
	if err != nil {
		return err
	}

	g.Resources = g.createResources(identityProviders)
	return nil
}

func getIdpOIDC(ctx context.Context, client *okta.APIClient) ([]okta.IdentityProvider, error) {
	output, resp, err := client.IdentityProviderAPI.ListIdentityProviders(ctx).Type_("OIDC").Limit(1).Execute()
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var nextIdpOIDCSet []okta.IdentityProvider
		resp, err = resp.Next(&nextIdpOIDCSet)
		if err != nil {
			return nil, err
		}
		output = append(output, nextIdpOIDCSet...)
	}

	return output, nil
}
