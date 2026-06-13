// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v6/okta"
)

type GroupGenerator struct {
	OktaService
}

func (g GroupGenerator) createResources(groupList []okta.Group) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, group := range groupList {
		groupName := getGroupName(group)
		resources = append(resources, terraformutils.NewSimpleResource(
			group.GetId(),
			"group_"+groupName,
			"okta_group",
			"okta",
			[]string{}))
	}
	return resources
}

func (g *GroupGenerator) InitResources() error {
	ctx, client, e := g.Client()
	if e != nil {
		return e
	}

	output, resp, err := client.GroupAPI.ListGroups(ctx).Filter("type eq \"OKTA_GROUP\"").Execute()
	if err != nil {
		return err
	}

	for resp.HasNextPage() {
		var nextGroupSet []okta.Group
		resp, err = resp.Next(&nextGroupSet)
		if err != nil {
			return err
		}
		output = append(output, nextGroupSet...)
	}

	g.Resources = g.createResources(output)
	return nil
}

func getGroupName(group okta.Group) string {
	profile := group.GetProfile()
	if profile.OktaUserGroupProfile != nil {
		return profile.OktaUserGroupProfile.GetName()
	}
	if profile.OktaActiveDirectoryGroupProfile != nil {
		return profile.OktaActiveDirectoryGroupProfile.GetName()
	}
	return group.GetId()
}
