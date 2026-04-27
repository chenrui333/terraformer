// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v2/okta"
)

type AuthorizationServerScopeGenerator struct {
	OktaService
}

func (g AuthorizationServerScopeGenerator) createResources(authorizationServerScopeList []*okta.OAuth2Scope, authorizationServerID string, authorizationServerName string) []terraformutils.Resource {
	var resources []terraformutils.Resource

	for _, authorizationServerScope := range authorizationServerScopeList {
		resources = append(resources, terraformutils.NewResource(
			authorizationServerScope.Id,
			normalizeResourceName("auth_server_"+authorizationServerName+"_scope_"+authorizationServerScope.Name),
			"okta_auth_server_scope",
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

func (g *AuthorizationServerScopeGenerator) InitResources() error {
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
		output, _, err := client.AuthorizationServer.ListOAuth2Scopes(ctx, authorizationServer.Id, nil)
		if err != nil {
			return err
		}

		resources = append(resources, g.createResources(output, authorizationServer.Id, authorizationServer.Name)...)
	}

	g.Resources = resources
	return nil
}
