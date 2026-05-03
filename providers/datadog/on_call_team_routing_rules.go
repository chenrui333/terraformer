// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// OnCallTeamRoutingRulesAllowEmptyValues ...
	OnCallTeamRoutingRulesAllowEmptyValues = []string{"rule.query"}
	errOnCallTeamRoutingRulesNotFound      = errors.New("On-Call team routing rules not found")
)

// OnCallTeamRoutingRulesGenerator ...
type OnCallTeamRoutingRulesGenerator struct {
	DatadogService
}

func (g *OnCallTeamRoutingRulesGenerator) createResource(teamRoutingRules datadogV2.TeamRoutingRules) (terraformutils.Resource, error) {
	data := teamRoutingRules.GetData()
	teamID := data.GetId()
	if teamID == "" {
		return terraformutils.Resource{}, fmt.Errorf("On-Call team routing rules missing team id")
	}

	return terraformutils.NewSimpleResource(
		teamID,
		fmt.Sprintf("on_call_team_routing_rules_%s", teamID),
		"datadog_on_call_team_routing_rules",
		"datadog",
		OnCallTeamRoutingRulesAllowEmptyValues,
	), nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each team's On-Call routing rules create 1 TerraformResource.
// Need Team ID as ID for terraform resource.
func (g *OnCallTeamRoutingRulesGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	onCallAPI := datadogV2.NewOnCallApi(datadogClient)
	teamAPI := datadogV2.NewTeamsApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, onCallAPI)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	teams, err := listTeams(auth, teamAPI)
	if err != nil {
		return err
	}
	for _, team := range teams {
		teamRoutingRules, err := getOnCallTeamRoutingRules(auth, onCallAPI, team.GetId())
		if err != nil {
			if errors.Is(err, errOnCallTeamRoutingRulesNotFound) {
				continue
			}
			return err
		}
		resource, err := g.createResource(teamRoutingRules)
		if err != nil {
			return err
		}
		resources = append(resources, resource)
	}

	g.Resources = resources
	return nil
}

func (g *OnCallTeamRoutingRulesGenerator) filteredResources(auth context.Context, api *datadogV2.OnCallApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for _, filter := range g.Filter {
		if !filter.IsApplicable("on_call_team_routing_rules") || filter.FieldPath != "id" {
			continue
		}

		filtered = true
		for _, value := range filter.AcceptableValues {
			teamRoutingRules, err := getOnCallTeamRoutingRules(auth, api, value)
			if err != nil {
				if errors.Is(err, errOnCallTeamRoutingRulesNotFound) {
					continue
				}
				return nil, true, err
			}
			resource, err := g.createResource(teamRoutingRules)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, filtered, nil
}

func getOnCallTeamRoutingRules(auth context.Context, api *datadogV2.OnCallApi, teamID string) (datadogV2.TeamRoutingRules, error) {
	include := "rules"
	teamRoutingRules, httpResp, err := api.GetOnCallTeamRoutingRules(auth, teamID, datadogV2.GetOnCallTeamRoutingRulesOptionalParameters{Include: &include})
	if httpResp != nil && httpResp.Body != nil {
		defer httpResp.Body.Close()
	}
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return datadogV2.TeamRoutingRules{}, errOnCallTeamRoutingRulesNotFound
		}
		return datadogV2.TeamRoutingRules{}, err
	}

	data := teamRoutingRules.GetData()
	if data.GetId() == "" {
		data.SetId(teamID)
		teamRoutingRules.SetData(data)
	}
	return teamRoutingRules, nil
}
