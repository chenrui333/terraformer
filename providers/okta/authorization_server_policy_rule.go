// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v6/okta"
)

type AuthorizationServerPolicyRuleGenerator struct {
	OktaService
}

func (g AuthorizationServerPolicyRuleGenerator) createResources(authorizationServerPolicyRuleList []okta.AuthorizationServerPolicyRule, authorizationServerID string, authorizationServerName string, authorizationServerPolicyID string, authorizationServerPolicyName string) []terraformutils.Resource {
	var resources []terraformutils.Resource

	for _, authorizationServerPolicyRule := range authorizationServerPolicyRuleList {
		resources = append(resources, terraformutils.NewResource(
			authorizationServerPolicyRule.GetId(),
			normalizeResourceName("auth_server_"+authorizationServerName+"_policy_"+authorizationServerPolicyName+"_rule_"+authorizationServerPolicyRule.GetName()),
			"okta_auth_server_policy_rule",
			"okta",
			map[string]string{
				"auth_server_id": authorizationServerID,
				"policy_id":      authorizationServerPolicyID,
			},
			[]string{},
			map[string]interface{}{},
		))
	}
	return resources
}

func (g *AuthorizationServerPolicyRuleGenerator) InitResources() error {
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
		authorizationServerPolicies, err := getAuthorizationServerPolicies(ctx, client, authorizationServer.GetId())
		if err != nil {
			return err
		}

		for _, authorizationServerPolicy := range authorizationServerPolicies {
			output, _, err := client.AuthorizationServerRulesAPI.ListAuthorizationServerPolicyRules(ctx, authorizationServer.GetId(), authorizationServerPolicy.GetId()).Execute()
			if err != nil {
				return err
			}

			resources = append(resources, g.createResources(output, authorizationServer.GetId(), authorizationServer.GetName(), authorizationServerPolicy.GetId(), authorizationServerPolicy.GetName())...)
		}
	}

	g.Resources = resources
	return nil
}
