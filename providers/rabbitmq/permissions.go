// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"encoding/json"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
)

type PermissionsGenerator struct {
	RBTService
}

type Permissions struct {
	User  string `json:"user"`
	Vhost string `json:"vhost"`
}

type AllPermissions []Permissions

var PermissionsAllowEmptyValues = []string{"configure", "write", "read"}
var PermissionsAdditionalFields = map[string]interface{}{}

func (g PermissionsGenerator) createResources(allPermissions AllPermissions) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, permissions := range allPermissions {
		resources = append(resources, terraformutils.NewResource(
			fmt.Sprintf("%s@%s", permissions.User, permissions.Vhost),
			fmt.Sprintf("permissions_%s_%s", normalizeResourceName(permissions.User), normalizeResourceName(permissions.Vhost)),
			"rabbitmq_permissions",
			"rabbitmq",
			map[string]string{
				"user":  permissions.User,
				"vhost": permissions.Vhost,
			},
			PermissionsAllowEmptyValues,
			PermissionsAdditionalFields,
		))
	}
	return resources
}

func (g *PermissionsGenerator) InitResources() error {
	body, err := g.generateRequest("/api/permissions?columns=user,vhost")
	if err != nil {
		return err
	}
	var permissions AllPermissions
	err = json.Unmarshal(body, &permissions)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(permissions)
	return nil
}
