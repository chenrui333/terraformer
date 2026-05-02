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
	// TeamNotificationRuleAllowEmptyValues ...
	TeamNotificationRuleAllowEmptyValues = []string{}
)

// TeamNotificationRuleGenerator ...
type TeamNotificationRuleGenerator struct {
	DatadogService
}

func (g *TeamNotificationRuleGenerator) createResources(teamID string, teamNotificationRules []datadogV2.TeamNotificationRule) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, teamNotificationRule := range teamNotificationRules {
		resource, err := g.createResource(teamID, teamNotificationRule)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *TeamNotificationRuleGenerator) createResource(teamID string, teamNotificationRule datadogV2.TeamNotificationRule) (terraformutils.Resource, error) {
	ruleID := teamNotificationRule.GetId()
	if ruleID == "" {
		return terraformutils.Resource{}, fmt.Errorf("team notification rule missing id")
	}
	if teamID == "" {
		return terraformutils.Resource{}, fmt.Errorf("team notification rule %q missing team id", ruleID)
	}

	return terraformutils.NewResource(
		ruleID,
		fmt.Sprintf("team_notification_rule_%s_%s", teamID, ruleID),
		"datadog_team_notification_rule",
		"datadog",
		map[string]string{
			"team_id": teamID,
		},
		TeamNotificationRuleAllowEmptyValues,
		map[string]interface{}{},
	), nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each team notification rule create 1 TerraformResource.
// Need Team Notification Rule ID formatted as '<team_id>:<rule_id>' for filter lookup.
func (g *TeamNotificationRuleGenerator) InitResources() error {
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
		teamNotificationRules, err := listTeamNotificationRules(auth, api, teamID)
		if err != nil {
			return err
		}
		teamResources, err := g.createResources(teamID, teamNotificationRules)
		if err != nil {
			return err
		}
		resources = append(resources, teamResources...)
	}

	g.Resources = resources
	return nil
}

func (g *TeamNotificationRuleGenerator) filteredResources(auth context.Context, api *datadogV2.TeamsApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for filterIndex, filter := range g.Filter {
		if !filter.IsApplicable("team_notification_rule") {
			continue
		}

		switch filter.FieldPath {
		case "id":
			filtered = true
			filterIDs, err := parseTeamNotificationRuleImportIDs(filter.AcceptableValues)
			if err != nil {
				return nil, true, err
			}
			for _, filterID := range filterIDs {
				teamNotificationRule, err := getTeamNotificationRule(auth, api, filterID.teamID, filterID.ruleID)
				if err != nil {
					return nil, true, err
				}
				resource, err := g.createResource(filterID.teamID, teamNotificationRule)
				if err != nil {
					return nil, true, err
				}
				resources = append(resources, resource)
			}
			g.Filter[filterIndex].AcceptableValues = teamNotificationRuleIDs(filterIDs)
		case "team_id":
			filtered = true
			for _, teamID := range filter.AcceptableValues {
				teamNotificationRules, err := listTeamNotificationRules(auth, api, teamID)
				if err != nil {
					return nil, true, err
				}
				teamResources, err := g.createResources(teamID, teamNotificationRules)
				if err != nil {
					return nil, true, err
				}
				resources = append(resources, teamResources...)
			}
		}
	}

	return resources, filtered, nil
}

type teamNotificationRuleFilterID struct {
	teamID string
	ruleID string
}

func parseTeamNotificationRuleImportIDs(importIDs []string) ([]teamNotificationRuleFilterID, error) {
	filterIDs := []teamNotificationRuleFilterID{}
	for _, importID := range importIDs {
		teamID, ruleID, err := parseTeamNotificationRuleImportID(importID)
		if err != nil {
			return nil, err
		}
		filterIDs = append(filterIDs, teamNotificationRuleFilterID{teamID: teamID, ruleID: ruleID})
	}
	return filterIDs, nil
}

func teamNotificationRuleIDs(filterIDs []teamNotificationRuleFilterID) []string {
	ruleIDs := []string{}
	for _, filterID := range filterIDs {
		ruleIDs = append(ruleIDs, filterID.ruleID)
	}
	return ruleIDs
}

func parseTeamNotificationRuleImportID(importID string) (string, string, error) {
	parts := strings.SplitN(importID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("team notification rule import ID %q must be formatted as team_id:rule_id", importID)
	}
	return parts[0], parts[1], nil
}

func getTeamNotificationRule(auth context.Context, api *datadogV2.TeamsApi, teamID string, ruleID string) (datadogV2.TeamNotificationRule, error) {
	teamNotificationRuleResponse, httpResponse, err := api.GetTeamNotificationRule(auth, teamID, ruleID)
	if httpResponse != nil && httpResponse.Body != nil {
		defer httpResponse.Body.Close()
	}
	if err != nil {
		return datadogV2.TeamNotificationRule{}, err
	}

	teamNotificationRule, ok := teamNotificationRuleFromResponse(teamNotificationRuleResponse, ruleID)
	if !ok {
		return datadogV2.TeamNotificationRule{}, fmt.Errorf("team notification rule %q not found", fmt.Sprintf("%s:%s", teamID, ruleID))
	}
	return teamNotificationRule, nil
}

func teamNotificationRuleFromResponse(teamNotificationRuleResponse datadogV2.TeamNotificationRuleResponse, ruleID string) (datadogV2.TeamNotificationRule, bool) {
	if teamNotificationRule, ok := teamNotificationRuleResponse.GetDataOk(); ok {
		return *teamNotificationRule, true
	}

	dataRaw, ok := teamNotificationRuleResponse.UnparsedObject["data"].(map[string]interface{})
	if !ok {
		return datadogV2.TeamNotificationRule{}, false
	}

	// Minimal API responses without attributes are stored as UnparsedObject by the SDK.
	if responseRuleID, ok := dataRaw["id"].(string); ok && responseRuleID != "" {
		ruleID = responseRuleID
	}
	if ruleID == "" {
		return datadogV2.TeamNotificationRule{}, false
	}

	teamNotificationRule := datadogV2.TeamNotificationRule{}
	teamNotificationRule.SetId(ruleID)
	return teamNotificationRule, true
}

func listTeamNotificationRules(auth context.Context, api *datadogV2.TeamsApi, teamID string) ([]datadogV2.TeamNotificationRule, error) {
	teamNotificationRulesResponse, httpResponse, err := api.GetTeamNotificationRules(auth, teamID)
	if httpResponse != nil && httpResponse.Body != nil {
		defer httpResponse.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	return teamNotificationRulesResponse.GetData(), nil
}
