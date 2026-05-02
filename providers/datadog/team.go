// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// TeamAllowEmptyValues ...
	TeamAllowEmptyValues = []string{"description"}
)

// TeamGenerator ...
type TeamGenerator struct {
	DatadogService
}

func (g *TeamGenerator) createResources(teams []datadogV2.Team) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, team := range teams {
		resources = append(resources, g.createResource(team))
	}

	return resources
}

func (g *TeamGenerator) createResource(team datadogV2.Team) terraformutils.Resource {
	teamID := team.GetId()
	resourceName := teamID
	attributes := team.GetAttributes()
	if handle := attributes.GetHandle(); handle != "" {
		resourceName = fmt.Sprintf("%s_%s", handle, teamID)
	}

	return terraformutils.NewSimpleResource(
		teamID,
		fmt.Sprintf("team_%s", resourceName),
		"datadog_team",
		"datadog",
		TeamAllowEmptyValues,
	)
}

func (g *TeamGenerator) PostConvertHook() error {
	for i := range g.Resources {
		if g.Resources[i].Item == nil {
			g.Resources[i].Item = map[string]interface{}{}
		}
		if description, ok := g.Resources[i].Item["description"]; !ok || description == nil {
			g.Resources[i].Item["description"] = ""
		}
	}
	return nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each team create 1 TerraformResource.
// Need Team ID as ID for terraform resource.
func (g *TeamGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewTeamsApi(datadogClient)

	resources := []terraformutils.Resource{}
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("team") {
			for _, value := range filter.AcceptableValues {
				team, err := getTeam(auth, api, value)
				if err != nil {
					return err
				}

				resources = append(resources, g.createResource(team))
			}
		}
	}

	if len(resources) > 0 {
		g.Resources = resources
		return nil
	}

	teams, err := listTeams(auth, api)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(teams)
	return nil
}

func getTeam(auth context.Context, api *datadogV2.TeamsApi, teamID string) (datadogV2.Team, error) {
	teamResponse, httpResponse, err := api.GetTeam(auth, teamID)
	if httpResponse != nil && httpResponse.Body != nil {
		defer httpResponse.Body.Close()
	}
	if err != nil {
		return datadogV2.Team{}, err
	}

	team, ok := teamResponse.GetDataOk()
	if !ok {
		return datadogV2.Team{}, fmt.Errorf("team %q not found", teamID)
	}

	return *team, nil
}

func listTeams(auth context.Context, api *datadogV2.TeamsApi) ([]datadogV2.Team, error) {
	pageSize := int64(100)
	items, cancel := api.ListTeamsWithPagination(auth, *datadogV2.NewListTeamsOptionalParameters().WithPageSize(pageSize))
	defer cancel()

	teams := []datadogV2.Team{}
	for item := range items {
		if item.Error != nil {
			return nil, item.Error
		}
		teams = append(teams, item.Item)
	}

	return teams, nil
}
