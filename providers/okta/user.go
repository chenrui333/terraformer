// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v5/okta"
)

type UserGenerator struct {
	OktaService
}

func (g UserGenerator) createResources(userList []okta.User) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, user := range userList {
		resources = append(resources, terraformutils.NewSimpleResource(
			user.GetId(),
			"user_"+user.GetId(),
			"okta_user",
			"okta",
			[]string{}))
	}
	return resources
}

func (g *UserGenerator) InitResources() error {
	ctx, client, err := g.ClientV5()
	if err != nil {
		return err
	}

	output, resp, err := client.UserAPI.ListUsers(ctx).Execute()
	if err != nil {
		return err
	}

	for resp.HasNextPage() {
		var nextUserSet []okta.User
		resp, _ = resp.Next(&nextUserSet)
		output = append(output, nextUserSet...)
	}

	g.Resources = g.createResources(output)
	return nil
}
