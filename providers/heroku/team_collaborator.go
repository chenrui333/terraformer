// SPDX-License-Identifier: Apache-2.0

package heroku

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	heroku "github.com/heroku/heroku-go/v5"
)

type TeamCollaboratorGenerator struct {
	HerokuService
}

func (g TeamCollaboratorGenerator) createResources(svc *heroku.Service, teamList []heroku.Team) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	for _, team := range teamList {
		apps, err := svc.TeamAppListByTeam(context.TODO(), team.ID, &heroku.ListRange{Field: "id"})
		if err != nil {
			return nil, err
		}
		for _, app := range apps {
			collaborators, err := svc.TeamAppCollaboratorList(context.TODO(), app.ID, &heroku.ListRange{Field: "id"})
			if err != nil {
				return nil, err
			}
			for _, collaborator := range collaborators {
				resources = append(resources, terraformutils.NewResource(
					collaborator.ID,
					collaborator.ID,
					"heroku_team_collaborator",
					"heroku",
					map[string]string{"app": app.Name},
					[]string{},
					map[string]interface{}{}))
			}
		}
	}
	return resources, nil
}

func (g *TeamCollaboratorGenerator) InitResources() error {
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
