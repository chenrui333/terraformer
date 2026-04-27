// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"encoding/json"

	"github.com/chenrui333/terraformer/terraformutils"
)

type UserGenerator struct {
	RBTService
}

type User struct {
	Name string `json:"name"`
}

type Users []User

var UserAllowEmptyValues = []string{}

func (g UserGenerator) createResources(users Users) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, user := range users {
		resources = append(resources, terraformutils.NewSimpleResource(
			user.Name,
			"user_"+normalizeResourceName(user.Name),
			"rabbitmq_user",
			"rabbitmq",
			UserAllowEmptyValues,
		))
	}
	return resources
}

func (g *UserGenerator) InitResources() error {
	body, err := g.generateRequest("/api/users?columns=name")
	if err != nil {
		return err
	}
	var users Users
	err = json.Unmarshal(body, &users)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(users)
	return nil
}
