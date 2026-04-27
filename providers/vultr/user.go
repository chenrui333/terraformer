// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr"
)

type UserGenerator struct {
	VultrService
}

func (g UserGenerator) createResources(userList []govultr.User) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, user := range userList {
		resources = append(resources, terraformutils.NewSimpleResource(
			user.UserID,
			user.UserID,
			"vultr_user",
			"vultr",
			[]string{}))
	}
	return resources
}

func (g *UserGenerator) InitResources() error {
	client := g.generateClient()
	output, err := client.User.List(context.Background())
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
