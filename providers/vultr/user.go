// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr/v3"
)

type UserGenerator struct {
	VultrService
}

func (g UserGenerator) createResources(userList []govultr.User) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, user := range userList {
		resources = append(resources, terraformutils.NewSimpleResource(
			user.ID,
			user.ID,
			"vultr_user",
			"vultr",
			[]string{}))
	}
	return resources
}

func (g *UserGenerator) InitResources() error {
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	output, err := listAllVultrResources(context.Background(), client.User.List)
	if err != nil {
		return fmt.Errorf("list vultr users: %w", err)
	}
	g.Resources = g.createResources(output)
	return nil
}
