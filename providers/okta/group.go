// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
)

type GroupGenerator struct {
	OktaService
}

func (g GroupGenerator) createResources(groupList []*okta.Group) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, group := range groupList {
		resources = append(resources, terraformutils.NewSimpleResource(
			group.Id,
			"group_"+group.Profile.Name,
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

	filter := query.NewQueryParams(query.WithFilter("type eq \"OKTA_GROUP\""))
	output, resp, err := client.Group.ListGroups(ctx, filter)
	if err != nil {
		return err
	}

	for resp.HasNextPage() {
		var nextGroupSet []*okta.Group
		resp, err = resp.Next(ctx, &nextGroupSet)
		if err != nil {
			return err
		}
		output = append(output, nextGroupSet...)
	}

	g.Resources = g.createResources(output)
	return nil
}
