// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
)

type IdpOIDCGenerator struct {
	OktaService
}

func (g IdpOIDCGenerator) createResources(idpOIDCList []*okta.IdentityProvider) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, idp := range idpOIDCList {
		resources = append(resources, terraformutils.NewSimpleResource(
			idp.Id,
			"idp_"+normalizeResourceName(idp.Type+"_"+idp.Name),
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

func getIdpOIDC(ctx context.Context, client *okta.Client) ([]*okta.IdentityProvider, error) {
	qp := &query.Params{Type: "OIDC", Limit: 1}
	output, resp, err := client.IdentityProvider.ListIdentityProviders(ctx, qp)
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var nextIdpOIDCSet []*okta.IdentityProvider
		resp, err = resp.Next(ctx, &nextIdpOIDCSet)
		if err != nil {
			return nil, err
		}
		output = append(output, nextIdpOIDCSet...)
	}

	return output, nil
}
