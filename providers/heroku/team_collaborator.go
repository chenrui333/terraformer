// SPDX-License-Identifier: Apache-2.0

package heroku

import (
	"context"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"
	heroku "github.com/heroku/heroku-go/v5"
)

type TeamCollaboratorGenerator struct {
	HerokuService
}

func (g TeamCollaboratorGenerator) createResources(svc *heroku.Service, teamList []heroku.Team) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, team := range teamList {
		apps, err := svc.TeamAppListByTeam(context.TODO(), team.ID, &heroku.ListRange{Field: "id"})
		if err != nil {
			log.Println(err)
		}
		for _, app := range apps {
			collaborators, err := svc.TeamAppCollaboratorList(context.TODO(), app.ID, &heroku.ListRange{Field: "id"})
			if err != nil {
				log.Println(err)
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
	return resources
}

func (g *TeamCollaboratorGenerator) InitResources() error {
	svc := g.generateService()
	output, err := svc.TeamList(context.TODO(), &heroku.ListRange{Field: "id"})
	if err != nil {
		return err
	}
	g.Resources = g.createResources(svc, output)
	return nil
}
