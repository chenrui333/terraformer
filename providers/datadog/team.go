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
	TeamAllowEmptyValues = []string{}
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
				teamResponse, _, err := api.GetTeam(auth, value)
				if err != nil {
					return err
				}
				team, ok := teamResponse.GetDataOk()
				if !ok {
					return fmt.Errorf("team %q not found", value)
				}

				resources = append(resources, g.createResource(*team))
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
