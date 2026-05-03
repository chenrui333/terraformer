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
	// TeamConnectionAllowEmptyValues ...
	TeamConnectionAllowEmptyValues = []string{}
)

// TeamConnectionGenerator ...
type TeamConnectionGenerator struct {
	DatadogService
}

func (g *TeamConnectionGenerator) createResources(teamConnections []datadogV2.TeamConnection) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, teamConnection := range teamConnections {
		resource, err := g.createResource(teamConnection)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *TeamConnectionGenerator) createResource(teamConnection datadogV2.TeamConnection) (terraformutils.Resource, error) {
	connectionID := teamConnection.GetId()
	if connectionID == "" {
		return terraformutils.Resource{}, fmt.Errorf("team connection missing id")
	}
	teamID, teamType := teamConnectionTeamRef(teamConnection)
	if teamID == "" {
		return terraformutils.Resource{}, fmt.Errorf("team connection %q missing team id", connectionID)
	}
	connectedTeamID, connectedTeamType := teamConnectionConnectedTeamRef(teamConnection)
	if connectedTeamID == "" {
		return terraformutils.Resource{}, fmt.Errorf("team connection %q missing connected team id", connectionID)
	}
	attributes := map[string]string{
		"team.id":             teamID,
		"team.type":           teamType,
		"connected_team.id":   connectedTeamID,
		"connected_team.type": connectedTeamType,
	}
	if connectionAttributes, ok := teamConnection.GetAttributesOk(); ok {
		if source, ok := connectionAttributes.GetSourceOk(); ok {
			attributes["source"] = *source
		}
	}

	return terraformutils.NewResource(
		connectionID,
		fmt.Sprintf("team_connection_%s", connectionID),
		"datadog_team_connection",
		"datadog",
		attributes,
		TeamConnectionAllowEmptyValues,
		map[string]interface{}{},
	), nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each team connection create 1 TerraformResource.
// Need Team Connection ID as ID for terraform resource.
func (g *TeamConnectionGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewTeamsApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	teamConnections, err := listTeamConnections(auth, api, *datadogV2.NewListTeamConnectionsOptionalParameters())
	if err != nil {
		return err
	}
	g.Resources, err = g.createResources(teamConnections)
	return err
}

func (g *TeamConnectionGenerator) filteredResources(auth context.Context, api *datadogV2.TeamsApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for _, filter := range g.Filter {
		if !filter.IsApplicable("team_connection") {
			continue
		}

		switch filter.FieldPath {
		case "id":
			filtered = true
			for _, connectionID := range filter.AcceptableValues {
				teamConnection, err := getTeamConnection(auth, api, connectionID)
				if err != nil {
					return nil, true, err
				}
				resource, err := g.createResource(teamConnection)
				if err != nil {
					return nil, true, err
				}
				resources = append(resources, resource)
			}
		case "source":
			filtered = true
			optionalParams := datadogV2.NewListTeamConnectionsOptionalParameters().WithFilterSources(filter.AcceptableValues)
			teamConnections, err := listTeamConnections(auth, api, *optionalParams)
			if err != nil {
				return nil, true, err
			}
			filteredResources, err := g.createResources(teamConnections)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, filteredResources...)
		}
	}

	return resources, filtered, nil
}

func getTeamConnection(auth context.Context, api *datadogV2.TeamsApi, connectionID string) (datadogV2.TeamConnection, error) {
	optionalParams := datadogV2.NewListTeamConnectionsOptionalParameters().WithFilterConnectionIds([]string{connectionID})
	teamConnections, err := listTeamConnections(auth, api, *optionalParams)
	if err != nil {
		return datadogV2.TeamConnection{}, err
	}
	for _, teamConnection := range teamConnections {
		if teamConnection.GetId() == connectionID {
			return teamConnection, nil
		}
	}
	return datadogV2.TeamConnection{}, fmt.Errorf("team connection %q not found", connectionID)
}

func listTeamConnections(auth context.Context, api *datadogV2.TeamsApi, optionalParams datadogV2.ListTeamConnectionsOptionalParameters) ([]datadogV2.TeamConnection, error) {
	pageSize := int64(100)
	optionalParams.WithPageSize(pageSize)
	items, cancel := api.ListTeamConnectionsWithPagination(auth, optionalParams)
	defer cancel()

	teamConnections := []datadogV2.TeamConnection{}
	for item := range items {
		if item.Error != nil {
			return nil, item.Error
		}
		teamConnections = append(teamConnections, item.Item)
	}

	return teamConnections, nil
}

func teamConnectionTeamRef(teamConnection datadogV2.TeamConnection) (string, string) {
	relationships, ok := teamConnection.GetRelationshipsOk()
	if !ok {
		return "", ""
	}
	team, ok := relationships.GetTeamOk()
	if !ok {
		return "", ""
	}
	data, ok := team.GetDataOk()
	if !ok {
		return "", ""
	}
	return data.GetId(), string(data.GetType())
}

func teamConnectionConnectedTeamRef(teamConnection datadogV2.TeamConnection) (string, string) {
	relationships, ok := teamConnection.GetRelationshipsOk()
	if !ok {
		return "", ""
	}
	connectedTeam, ok := relationships.GetConnectedTeamOk()
	if !ok {
		return "", ""
	}
	data, ok := connectedTeam.GetDataOk()
	if !ok {
		return "", ""
	}
	return data.GetId(), string(data.GetType())
}
