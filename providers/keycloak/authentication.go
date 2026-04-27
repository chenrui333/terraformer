// SPDX-License-Identifier: Apache-2.0

package keycloak

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/keycloak/terraform-provider-keycloak/keycloak"
)

func (g RealmGenerator) createAuthenticationFlowResources(authenticationFlows []*keycloak.AuthenticationFlow) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, authenticationFlow := range authenticationFlows {
		resources = append(resources, terraformutils.NewResource(
			authenticationFlow.Id,
			"authentication_flow_"+normalizeResourceName(authenticationFlow.RealmId)+"_"+normalizeResourceName(authenticationFlow.Id),
			"keycloak_authentication_flow",
			"keycloak",
			map[string]string{
				"realm_id": authenticationFlow.RealmId,
				"alias":    authenticationFlow.Alias,
			},
			[]string{},
			map[string]interface{}{},
		))
	}
	return resources
}

func (g RealmGenerator) createAuthenticationSubFlowResource(authenticationSubFlow *keycloak.AuthenticationSubFlow) terraformutils.Resource {
	resource := terraformutils.NewResource(
		authenticationSubFlow.Id,
		"authentication_subflow_"+normalizeResourceName(authenticationSubFlow.RealmId)+"_"+normalizeResourceName(authenticationSubFlow.Id),
		"keycloak_authentication_subflow",
		"keycloak",
		map[string]string{
			"realm_id":          authenticationSubFlow.RealmId,
			"parent_flow_alias": authenticationSubFlow.ParentFlowAlias,
			"alias":             authenticationSubFlow.Alias,
			"requirement":       authenticationSubFlow.Requirement,
		},
		[]string{},
		map[string]interface{}{},
	)
	return resource
}

func (g RealmGenerator) createAuthenticationExecutionResource(authenticationExecution *keycloak.AuthenticationExecution) terraformutils.Resource {
	resource := terraformutils.NewResource(
		authenticationExecution.Id,
		"authentication_execution_"+normalizeResourceName(authenticationExecution.RealmId)+"_"+normalizeResourceName(authenticationExecution.Id),
		"keycloak_authentication_execution",
		"keycloak",
		map[string]string{
			"realm_id":          authenticationExecution.RealmId,
			"parent_flow_alias": authenticationExecution.ParentFlowAlias,
			"authenticator":     authenticationExecution.Authenticator,
		},
		[]string{},
		map[string]interface{}{},
	)
	return resource
}

func (g RealmGenerator) createAuthenticationExecutionConfigResource(authenticationExecutionConfig *keycloak.AuthenticationExecutionConfig) terraformutils.Resource {
	return terraformutils.NewResource(
		authenticationExecutionConfig.Id,
		"authentication_execution_config_"+normalizeResourceName(authenticationExecutionConfig.RealmId)+"_"+normalizeResourceName(authenticationExecutionConfig.Id),
		"keycloak_authentication_execution_config",
		"keycloak",
		map[string]string{
			"realm_id":     authenticationExecutionConfig.RealmId,
			"execution_id": authenticationExecutionConfig.ExecutionId,
			"alias":        authenticationExecutionConfig.Alias,
		},
		[]string{},
		map[string]interface{}{
			"config": authenticationExecutionConfig.Config,
		},
	)
}
