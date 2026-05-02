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
	// TeamPermissionSettingAllowEmptyValues ...
	TeamPermissionSettingAllowEmptyValues = []string{}
)

// TeamPermissionSettingGenerator ...
type TeamPermissionSettingGenerator struct {
	DatadogService
}

func (g *TeamPermissionSettingGenerator) createResources(teamID string, teamPermissionSettings []datadogV2.TeamPermissionSetting) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, teamPermissionSetting := range teamPermissionSettings {
		resource, err := g.createResource(teamID, teamPermissionSetting)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *TeamPermissionSettingGenerator) createResource(teamID string, teamPermissionSetting datadogV2.TeamPermissionSetting) (terraformutils.Resource, error) {
	permissionID := teamPermissionSetting.GetId()
	if permissionID == "" {
		return terraformutils.Resource{}, fmt.Errorf("team permission setting missing id")
	}
	if teamID == "" {
		return terraformutils.Resource{}, fmt.Errorf("team permission setting %q missing team id", permissionID)
	}

	attributes := teamPermissionSetting.GetAttributes()
	action := string(attributes.GetAction())
	if action == "" {
		return terraformutils.Resource{}, fmt.Errorf("team permission setting %q missing action", permissionID)
	}
	value := string(attributes.GetValue())
	if value == "" {
		return terraformutils.Resource{}, fmt.Errorf("team permission setting %q missing value", permissionID)
	}

	return terraformutils.NewResource(
		permissionID,
		fmt.Sprintf("team_permission_setting_%s_%s", teamID, action),
		"datadog_team_permission_setting",
		"datadog",
		map[string]string{
			"team_id": teamID,
			"action":  action,
			"value":   value,
		},
		TeamPermissionSettingAllowEmptyValues,
		map[string]interface{}{},
	), nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each team permission setting create 1 TerraformResource.
// Need Team Permission Setting ID formatted as '<team_id>:<action>' for filter lookup.
func (g *TeamPermissionSettingGenerator) InitResources() error {
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
		teamPermissionSettings, err := listTeamPermissionSettings(auth, api, teamID)
		if err != nil {
			return err
		}
		teamResources, err := g.createResources(teamID, teamPermissionSettings)
		if err != nil {
			return err
		}
		resources = append(resources, teamResources...)
	}

	g.Resources = resources
	return nil
}

func (g *TeamPermissionSettingGenerator) filteredResources(auth context.Context, api *datadogV2.TeamsApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for filterIndex, filter := range g.Filter {
		if !filter.IsApplicable("team_permission_setting") {
			continue
		}

		switch filter.FieldPath {
		case "id":
			filtered = true
			filterIDs, err := parseTeamPermissionSettingImportIDs(filter.AcceptableValues)
			if err != nil {
				return nil, true, err
			}
			permissionIDs := []string{}
			for _, filterID := range filterIDs {
				teamPermissionSetting, err := getTeamPermissionSetting(auth, api, filterID.teamID, filterID.action)
				if err != nil {
					return nil, true, err
				}
				resource, err := g.createResource(filterID.teamID, teamPermissionSetting)
				if err != nil {
					return nil, true, err
				}
				permissionIDs = append(permissionIDs, resource.InstanceState.ID)
				resources = append(resources, resource)
			}
			g.Filter[filterIndex].AcceptableValues = permissionIDs
		case "team_id":
			filtered = true
			for _, teamID := range filter.AcceptableValues {
				teamPermissionSettings, err := listTeamPermissionSettings(auth, api, teamID)
				if err != nil {
					return nil, true, err
				}
				teamResources, err := g.createResources(teamID, teamPermissionSettings)
				if err != nil {
					return nil, true, err
				}
				resources = append(resources, teamResources...)
			}
		}
	}

	return resources, filtered, nil
}

type teamPermissionSettingFilterID struct {
	teamID string
	action string
}

func parseTeamPermissionSettingImportIDs(importIDs []string) ([]teamPermissionSettingFilterID, error) {
	filterIDs := []teamPermissionSettingFilterID{}
	for _, importID := range importIDs {
		teamID, action, err := parseTeamPermissionSettingImportID(importID)
		if err != nil {
			return nil, err
		}
		filterIDs = append(filterIDs, teamPermissionSettingFilterID{teamID: teamID, action: action})
	}
	return filterIDs, nil
}

func parseTeamPermissionSettingImportID(importID string) (string, string, error) {
	parts := strings.SplitN(importID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("team permission setting import ID %q must be formatted as team_id:action", importID)
	}
	return parts[0], parts[1], nil
}

func getTeamPermissionSetting(auth context.Context, api *datadogV2.TeamsApi, teamID string, action string) (datadogV2.TeamPermissionSetting, error) {
	teamPermissionSettings, err := listTeamPermissionSettings(auth, api, teamID)
	if err != nil {
		return datadogV2.TeamPermissionSetting{}, err
	}

	for _, teamPermissionSetting := range teamPermissionSettings {
		attributes := teamPermissionSetting.GetAttributes()
		if string(attributes.GetAction()) == action {
			return teamPermissionSetting, nil
		}
	}
	return datadogV2.TeamPermissionSetting{}, fmt.Errorf("team permission setting %q not found", fmt.Sprintf("%s:%s", teamID, action))
}

func listTeamPermissionSettings(auth context.Context, api *datadogV2.TeamsApi, teamID string) ([]datadogV2.TeamPermissionSetting, error) {
	teamPermissionSettingsResponse, httpResponse, err := api.GetTeamPermissionSettings(auth, teamID)
	if httpResponse != nil && httpResponse.Body != nil {
		defer httpResponse.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	return teamPermissionSettingsResponse.GetData(), nil
}
