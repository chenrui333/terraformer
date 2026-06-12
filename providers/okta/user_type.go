// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v6/okta"
)

type UserTypeGenerator struct {
	OktaService
}

func (g UserTypeGenerator) createResources(userTypeList []okta.UserType) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, userType := range userTypeList {
		resources = append(resources, terraformutils.NewSimpleResource(
			userType.GetId(),
			"usertype_"+getUserTypeName(userType),
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

	output, resp, err := client.UserTypeAPI.ListUserTypes(ctx).Execute()
	if err != nil {
		return err
	}

	for resp.HasNextPage() {
		var nextUserTypeSet []okta.UserType
		resp, err = resp.Next(&nextUserTypeSet)
		if err != nil {
			return err
		}
		output = append(output, nextUserTypeSet...)
	}

	g.Resources = g.createResources(output)
	return nil
}
