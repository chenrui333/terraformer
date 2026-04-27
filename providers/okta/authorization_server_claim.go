// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v2/okta"
)

type AuthorizationServerClaimGenerator struct {
	OktaService
}

func (g AuthorizationServerClaimGenerator) createResources(authorizationServerClaimList []*okta.OAuth2Claim, authorizationServerID string, authorizationServerName string) []terraformutils.Resource {
	var resources []terraformutils.Resource

	for _, authorizationServerClaim := range authorizationServerClaimList {
		resourceType := "okta_auth_server_claim"
		if authorizationServerClaim.Name == "sub" {
			resourceType = "okta_auth_server_claim_default"
		}
		resources = append(resources, terraformutils.NewResource(
			authorizationServerClaim.Id,
			normalizeResourceName("auth_server_"+authorizationServerName+"_claim_"+authorizationServerClaim.Id),
			resourceType,
			"okta",
			map[string]string{
				"auth_server_id": authorizationServerID,
			},
			[]string{},
			map[string]interface{}{},
		))
	}
	return resources
}

func (g *AuthorizationServerClaimGenerator) InitResources() error {
	var resources []terraformutils.Resource
	ctx, client, e := g.Client()
	if e != nil {
		return e
	}

	authorizationServers, err := getAuthorizationServers(ctx, client)
	if err != nil {
		return err
	}

	for _, authorizationServer := range authorizationServers {
		output, _, err := client.AuthorizationServer.ListOAuth2Claims(ctx, authorizationServer.Id)
		if err != nil {
			return err
		}

		resources = append(resources, g.createResources(output, authorizationServer.Id, authorizationServer.Name)...)
	}

	g.Resources = resources
	return nil
}
