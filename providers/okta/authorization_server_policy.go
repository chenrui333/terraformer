// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v6/okta"
)

type AuthorizationServerPolicyGenerator struct {
	OktaService
}

func (g AuthorizationServerPolicyGenerator) createResources(authorizationServerPolicyList []okta.AuthorizationServerPolicy, authorizationServerID string, authorizationServerName string) []terraformutils.Resource {
	var resources []terraformutils.Resource

	for _, authorizationServerPolicy := range authorizationServerPolicyList {
		resources = append(resources, terraformutils.NewResource(
			authorizationServerPolicy.GetId(),
			normalizeResourceName("auth_server_"+authorizationServerName+"_policy_"+authorizationServerPolicy.GetName()),
			"okta_auth_server_policy",
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

func (g *AuthorizationServerPolicyGenerator) InitResources() error {
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
		output, _, err := client.AuthorizationServerPoliciesAPI.ListAuthorizationServerPolicies(ctx, authorizationServer.GetId()).Execute()
		if err != nil {
			return err
		}

		resources = append(resources, g.createResources(output, authorizationServer.GetId(), authorizationServer.GetName())...)
	}

	g.Resources = resources
	return nil
}
