// SPDX-License-Identifier: Apache-2.0

package heroku

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	heroku "github.com/heroku/heroku-go/v5"
)

type TeamMemberGenerator struct {
	HerokuService
}

func (g TeamMemberGenerator) createResources(svc *heroku.Service, teamList []heroku.Team) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	for _, team := range teamList {
		output, err := svc.TeamMemberList(context.TODO(), team.ID, &heroku.ListRange{Field: "id"})
		if err != nil {
			return nil, err
		}
		for _, member := range output {
			resources = append(resources, terraformutils.NewSimpleResource(
				fmt.Sprintf("%s:%s", team.ID, member.Email),
				member.ID,
				"heroku_team_member",
				"heroku",
				[]string{}))
		}
	}
	return resources, nil
}

func (g *TeamMemberGenerator) InitResources() error {
	svc := g.generateService()
	output, err := svc.TeamList(context.TODO(), &heroku.ListRange{Field: "id"})
	if err != nil {
		return err
	}
	resources, err := g.createResources(svc, output)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}
