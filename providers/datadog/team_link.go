// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// TeamLinkAllowEmptyValues ...
	TeamLinkAllowEmptyValues = []string{}
)

// TeamLinkGenerator ...
type TeamLinkGenerator struct {
	DatadogService
}

func (g *TeamLinkGenerator) createResources(teamID string, teamLinks []datadogV2.TeamLink) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, teamLink := range teamLinks {
		resource, err := g.createResource(teamID, teamLink)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *TeamLinkGenerator) createResource(teamID string, teamLink datadogV2.TeamLink) (terraformutils.Resource, error) {
	linkID := teamLink.GetId()
	if linkID == "" {
		return terraformutils.Resource{}, fmt.Errorf("team link missing id")
	}

	attributes := teamLink.GetAttributes()
	if attributeTeamID := attributes.GetTeamId(); attributeTeamID != "" {
		teamID = attributeTeamID
	}
	if teamID == "" {
		return terraformutils.Resource{}, fmt.Errorf("team link %q missing team id", linkID)
	}

	return terraformutils.NewResource(
		linkID,
		fmt.Sprintf("team_link_%s_%s", teamID, linkID),
		"datadog_team_link",
		"datadog",
		map[string]string{
			"team_id": teamID,
		},
		TeamLinkAllowEmptyValues,
		map[string]interface{}{},
	), nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each team link create 1 TerraformResource.
// Need Team Link ID formatted as '<team_id>:<link_id>' for filter lookup.
func (g *TeamLinkGenerator) InitResources() error {
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

	teams, err := listTeams(auth, api)
	if err != nil {
		return err
	}
	for _, team := range teams {
		teamID := team.GetId()
		teamLinks, err := listTeamLinks(auth, api, teamID)
		if err != nil {
			return err
		}
		teamResources, err := g.createResources(teamID, teamLinks)
		if err != nil {
			return err
		}
		resources = append(resources, teamResources...)
	}

	g.Resources = resources
	return nil
}

func (g *TeamLinkGenerator) filteredResources(auth context.Context, api *datadogV2.TeamsApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for filterIndex, filter := range g.Filter {
		if !filter.IsApplicable("team_link") {
			continue
		}

		switch filter.FieldPath {
		case "id":
			filtered = true
			filterIDs, err := parseTeamLinkImportIDs(filter.AcceptableValues)
			if err != nil {
				return nil, true, err
			}
			for _, filterID := range filterIDs {
				teamLink, err := getTeamLink(auth, api, filterID.teamID, filterID.linkID)
				if err != nil {
					return nil, true, err
				}
				resource, err := g.createResource(filterID.teamID, teamLink)
				if err != nil {
					return nil, true, err
				}
				resources = append(resources, resource)
			}
			g.Filter[filterIndex].AcceptableValues = teamLinkIDs(filterIDs)
		case "team_id":
			filtered = true
			for _, teamID := range filter.AcceptableValues {
				teamLinks, err := listTeamLinks(auth, api, teamID)
				if err != nil {
					return nil, true, err
				}
				teamResources, err := g.createResources(teamID, teamLinks)
				if err != nil {
					return nil, true, err
				}
				resources = append(resources, teamResources...)
			}
		}
	}

	return resources, filtered, nil
}

type teamLinkFilterID struct {
	teamID string
	linkID string
}

func parseTeamLinkImportIDs(importIDs []string) ([]teamLinkFilterID, error) {
	filterIDs := []teamLinkFilterID{}
	for _, importID := range importIDs {
		teamID, linkID, err := parseTeamLinkImportID(importID)
		if err != nil {
			return nil, err
		}
		filterIDs = append(filterIDs, teamLinkFilterID{teamID: teamID, linkID: linkID})
	}
	return filterIDs, nil
}

func teamLinkIDs(filterIDs []teamLinkFilterID) []string {
	linkIDs := []string{}
	for _, filterID := range filterIDs {
		linkIDs = append(linkIDs, filterID.linkID)
	}
	return linkIDs
}

func parseTeamLinkImportID(importID string) (string, string, error) {
	parts := strings.SplitN(importID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("team link import ID %q must be formatted as team_id:link_id", importID)
	}
	return parts[0], parts[1], nil
}

func getTeamLink(auth context.Context, api *datadogV2.TeamsApi, teamID string, linkID string) (datadogV2.TeamLink, error) {
	teamLinkResponse, httpResponse, err := api.GetTeamLink(auth, teamID, linkID)
	if httpResponse != nil && httpResponse.Body != nil {
		defer httpResponse.Body.Close()
	}
	if err != nil {
		return datadogV2.TeamLink{}, err
	}

	teamLink, ok := teamLinkResponse.GetDataOk()
	if !ok {
		return datadogV2.TeamLink{}, fmt.Errorf("team link %q not found", fmt.Sprintf("%s:%s", teamID, linkID))
	}
	return *teamLink, nil
}

func listTeamLinks(auth context.Context, api *datadogV2.TeamsApi, teamID string) ([]datadogV2.TeamLink, error) {
	teamLinksResponse, httpResponse, err := api.GetTeamLinks(auth, teamID)
	if httpResponse != nil && httpResponse.Body != nil {
		defer httpResponse.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	return teamLinksResponse.GetData(), nil
}
