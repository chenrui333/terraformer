// SPDX-License-Identifier: Apache-2.0

package keycloak

import (
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/keycloak/terraform-provider-keycloak/keycloak"
)

func (g RealmGenerator) createScopeResources(realmID string, openidClientScopes []*keycloak.OpenidClientScope) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, openidClientScope := range openidClientScopes {
		resources = append(resources, terraformutils.NewResource(
			openidClientScope.Id,
			"openid_client_scope_"+normalizeResourceName(realmID)+"_"+normalizeResourceName(openidClientScope.Name),
			"keycloak_openid_client_scope",
			"keycloak",
			map[string]string{
				"realm_id": realmID,
			},
			[]string{},
			map[string]interface{}{},
		))
	}
	return resources
}

func (g RealmGenerator) createOpenidClientScopesResources(realmID, clientID, clientClientID, t string, openidClientScopes *[]keycloak.OpenidClientScope) terraformutils.Resource {
	var scopes []string
	for _, openidClientScope := range *openidClientScopes {
		scopes = append(scopes, openidClientScope.Name)
	}
	return terraformutils.NewResource(
		realmID+"/"+clientID,
		"openid_client_"+t+"_scopes_"+normalizeResourceName(realmID)+"_"+normalizeResourceName(clientClientID),
		"keycloak_openid_client_"+t+"_scopes",
		"keycloak",
		map[string]string{
			"realm_id":    realmID,
			"client_id":   clientID,
			t + "_scopes": strings.Join(scopes, ","),
		},
		[]string{},
		map[string]interface{}{},
	)
}
