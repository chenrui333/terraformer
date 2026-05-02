// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v2/okta"
)

type UserTypeGenerator struct {
	OktaService
}

func (g UserTypeGenerator) createResources(userTypeList []*okta.UserType) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, userType := range userTypeList {
		resources = append(resources, terraformutils.NewSimpleResource(
			userType.Id,
			"usertype_"+userType.Name,
			"okta_user_type",
			"okta",
			[]string{}))
	}
	return resources
}

func (g *UserTypeGenerator) InitResources() error {
	ctx, client, e := g.Client()
	if e != nil {
		return e
	}

	output, resp, err := client.UserType.ListUserTypes(ctx)
	if err != nil {
		return err
	}

	for resp.HasNextPage() {
		var nextUserTypeSet []*okta.UserType
		resp, err = resp.Next(ctx, &nextUserTypeSet)
		if err != nil {
			return err
		}
		output = append(output, nextUserTypeSet...)
	}

	g.Resources = g.createResources(output)
	return nil
}
