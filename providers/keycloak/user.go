// SPDX-License-Identifier: Apache-2.0

package keycloak

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/keycloak/terraform-provider-keycloak/keycloak"
)

func (g RealmGenerator) createUserResources(users []*keycloak.User) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, user := range users {
		resources = append(resources, terraformutils.NewResource(
			user.Id,
			"user_"+normalizeResourceName(user.RealmId)+"_"+normalizeResourceName(user.Username),
			"keycloak_user",
			"keycloak",
			map[string]string{
				"realm_id": user.RealmId,
			},
			[]string{},
			map[string]interface{}{},
		))
	}
	return resources
}
