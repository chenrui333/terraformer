// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
)

type IdpSAMLGenerator struct {
	OktaService
}

func (g IdpSAMLGenerator) createResources(idpSAMLList []*okta.IdentityProvider) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, idp := range idpSAMLList {
		resources = append(resources, terraformutils.NewSimpleResource(
			idp.Id,
			"idp_"+normalizeResourceName(idp.Type+"_"+idp.Name),
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

func getIdpSAML(ctx context.Context, client *okta.Client) ([]*okta.IdentityProvider, error) {
	qp := &query.Params{Type: "SAML2", Limit: 1}
	output, resp, err := client.IdentityProvider.ListIdentityProviders(ctx, qp)
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var nextIdpSAMLSet []*okta.IdentityProvider
		resp, err = resp.Next(ctx, &nextIdpSAMLSet)
		if err != nil {
			return nil, err
		}
		output = append(output, nextIdpSAMLSet...)
	}

	return output, nil
}
