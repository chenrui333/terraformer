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
	// TeamMembershipAllowEmptyValues ...
	TeamMembershipAllowEmptyValues = []string{}
)

// TeamMembershipGenerator ...
type TeamMembershipGenerator struct {
	DatadogService
}

func (g *TeamMembershipGenerator) createResources(teamID string, teamMemberships []datadogV2.UserTeam) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, teamMembership := range teamMemberships {
		resource, err := g.createResource(teamID, teamMembership)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *TeamMembershipGenerator) createResource(teamID string, teamMembership datadogV2.UserTeam) (terraformutils.Resource, error) {
	teamID = teamIDFromTeamMembership(teamID, teamMembership)
	if teamID == "" {
		return terraformutils.Resource{}, fmt.Errorf("team membership %q missing team relationship id", teamMembership.GetId())
	}
	userID := userIDFromTeamMembership(teamMembership)
	if userID == "" {
		return terraformutils.Resource{}, fmt.Errorf("team membership %q missing user relationship id", teamMembership.GetId())
	}

	resourceID := fmt.Sprintf("%s:%s", teamID, userID)
	return terraformutils.NewSimpleResource(
		resourceID,
		fmt.Sprintf("team_membership_%s_%s", teamID, userID),
		"datadog_team_membership",
		"datadog",
		TeamMembershipAllowEmptyValues,
	), nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each team membership create 1 TerraformResource.
// Need Team Membership ID formatted as '<team_id>:<user_id>' as ID for terraform resource.
func (g *TeamMembershipGenerator) InitResources() error {
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
		teamMemberships, err := listTeamMemberships(auth, api, teamID)
		if err != nil {
			return err
		}
		teamResources, err := g.createResources(teamID, teamMemberships)
		if err != nil {
			return err
		}
		resources = append(resources, teamResources...)
	}

	g.Resources = resources
	return nil
}

func (g *TeamMembershipGenerator) filteredResources(auth context.Context, api *datadogV2.TeamsApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for _, filter := range g.Filter {
		if !filter.IsApplicable("team_membership") {
			continue
		}

		switch filter.FieldPath {
		case "id":
			filtered = true
			for _, value := range filter.AcceptableValues {
				teamID, userID, err := parseTeamMembershipImportID(value)
				if err != nil {
					return nil, true, err
				}
				teamMembership, err := getTeamMembership(auth, api, teamID, userID)
				if err != nil {
					return nil, true, err
				}
				resource, err := g.createResource(teamID, teamMembership)
				if err != nil {
					return nil, true, err
				}
				resources = append(resources, resource)
			}
		case "team_id":
			filtered = true
			for _, teamID := range filter.AcceptableValues {
				teamMemberships, err := listTeamMemberships(auth, api, teamID)
				if err != nil {
					return nil, true, err
				}
				teamResources, err := g.createResources(teamID, teamMemberships)
				if err != nil {
					return nil, true, err
				}
				resources = append(resources, teamResources...)
			}
		}
	}

	return resources, filtered, nil
}

func parseTeamMembershipImportID(importID string) (string, string, error) {
	parts := strings.SplitN(importID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("team membership import ID %q must be formatted as team_id:user_id", importID)
	}
	return parts[0], parts[1], nil
}

func getTeamMembership(auth context.Context, api *datadogV2.TeamsApi, teamID string, userID string) (datadogV2.UserTeam, error) {
	teamMemberships, err := listTeamMemberships(auth, api, teamID)
	if err != nil {
		return datadogV2.UserTeam{}, err
	}
	for _, teamMembership := range teamMemberships {
		if userIDFromTeamMembership(teamMembership) == userID {
			return teamMembership, nil
		}
	}
	return datadogV2.UserTeam{}, fmt.Errorf("team membership %q not found", fmt.Sprintf("%s:%s", teamID, userID))
}

func listTeamMemberships(auth context.Context, api *datadogV2.TeamsApi, teamID string) ([]datadogV2.UserTeam, error) {
	pageSize := int64(100)
	items, cancel := api.GetTeamMembershipsWithPagination(auth, teamID, *datadogV2.NewGetTeamMembershipsOptionalParameters().WithPageSize(pageSize))
	defer cancel()

	teamMemberships := []datadogV2.UserTeam{}
	for item := range items {
		if item.Error != nil {
			return nil, item.Error
		}
		teamMemberships = append(teamMemberships, item.Item)
	}

	return teamMemberships, nil
}

func teamIDFromTeamMembership(defaultTeamID string, teamMembership datadogV2.UserTeam) string {
	relationships := teamMembership.GetRelationships()
	teamRelationship := relationships.GetTeam()
	teamData := teamRelationship.GetData()
	if teamID := teamData.GetId(); teamID != "" {
		return teamID
	}
	return defaultTeamID
}

func userIDFromTeamMembership(teamMembership datadogV2.UserTeam) string {
	relationships := teamMembership.GetRelationships()
	userRelationship := relationships.GetUser()
	userData := userRelationship.GetData()
	return userData.GetId()
}
