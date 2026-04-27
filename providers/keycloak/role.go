// SPDX-License-Identifier: Apache-2.0

package keycloak

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/keycloak/terraform-provider-keycloak/keycloak"
)

func (g RealmGenerator) createRoleResources(roles []*keycloak.Role) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, role := range roles {
		resources = append(resources, terraformutils.NewResource(
			role.Id,
			"role_"+normalizeResourceName(role.RealmId)+normalizeResourceName(role.ContainerId)+"_"+normalizeResourceName(role.Name),
			"keycloak_role",
			"keycloak",
			map[string]string{
				"realm_id": role.RealmId,
			},
			[]string{},
			map[string]interface{}{},
		))
	}
	return resources
}
