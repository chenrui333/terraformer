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
	// TeamHierarchyLinksAllowEmptyValues ...
	TeamHierarchyLinksAllowEmptyValues = []string{}
)

// TeamHierarchyLinksGenerator ...
type TeamHierarchyLinksGenerator struct {
	DatadogService
}

func (g *TeamHierarchyLinksGenerator) createResources(teamHierarchyLinks []datadogV2.TeamHierarchyLink) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, teamHierarchyLink := range teamHierarchyLinks {
		resource, err := g.createResource(teamHierarchyLink)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *TeamHierarchyLinksGenerator) createResource(teamHierarchyLink datadogV2.TeamHierarchyLink) (terraformutils.Resource, error) {
	linkID := teamHierarchyLink.GetId()
	if linkID == "" {
		return terraformutils.Resource{}, fmt.Errorf("team hierarchy link missing id")
	}
	parentTeamID := teamHierarchyLinkParentTeamID(teamHierarchyLink)
	if parentTeamID == "" {
		return terraformutils.Resource{}, fmt.Errorf("team hierarchy link %q missing parent team id", linkID)
	}
	subTeamID := teamHierarchyLinkSubTeamID(teamHierarchyLink)
	if subTeamID == "" {
		return terraformutils.Resource{}, fmt.Errorf("team hierarchy link %q missing sub team id", linkID)
	}

	return terraformutils.NewResource(
		linkID,
		fmt.Sprintf("team_hierarchy_links_%s_%s", parentTeamID, subTeamID),
		"datadog_team_hierarchy_links",
		"datadog",
		map[string]string{
			"parent_team_id": parentTeamID,
			"sub_team_id":    subTeamID,
		},
		TeamHierarchyLinksAllowEmptyValues,
		map[string]interface{}{},
	), nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each team hierarchy link create 1 TerraformResource.
// Need Team Hierarchy Link ID as ID for terraform resource.
func (g *TeamHierarchyLinksGenerator) InitResources() error {
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

	teamHierarchyLinks, err := listTeamHierarchyLinks(auth, api, *datadogV2.NewListTeamHierarchyLinksOptionalParameters())
	if err != nil {
		return err
	}
	g.Resources, err = g.createResources(teamHierarchyLinks)
	return err
}

func (g *TeamHierarchyLinksGenerator) filteredResources(auth context.Context, api *datadogV2.TeamsApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for _, filter := range g.Filter {
		if !filter.IsApplicable("team_hierarchy_links") {
			continue
		}

		switch filter.FieldPath {
		case "id":
			filtered = true
			for _, linkID := range filter.AcceptableValues {
				teamHierarchyLink, err := getTeamHierarchyLink(auth, api, linkID)
				if err != nil {
					return nil, true, err
				}
				resource, err := g.createResource(teamHierarchyLink)
				if err != nil {
					return nil, true, err
				}
				resources = append(resources, resource)
			}
		case "parent_team_id":
			filtered = true
			for _, parentTeamID := range filter.AcceptableValues {
				optionalParams := datadogV2.NewListTeamHierarchyLinksOptionalParameters().WithFilterParentTeam(parentTeamID)
				teamHierarchyLinks, err := listTeamHierarchyLinks(auth, api, *optionalParams)
				if err != nil {
					return nil, true, err
				}
				filteredResources, err := g.createResources(teamHierarchyLinks)
				if err != nil {
					return nil, true, err
				}
				resources = append(resources, filteredResources...)
			}
		case "sub_team_id":
			filtered = true
			for _, subTeamID := range filter.AcceptableValues {
				optionalParams := datadogV2.NewListTeamHierarchyLinksOptionalParameters().WithFilterSubTeam(subTeamID)
				teamHierarchyLinks, err := listTeamHierarchyLinks(auth, api, *optionalParams)
				if err != nil {
					return nil, true, err
				}
				filteredResources, err := g.createResources(teamHierarchyLinks)
				if err != nil {
					return nil, true, err
				}
				resources = append(resources, filteredResources...)
			}
		}
	}

	return resources, filtered, nil
}

func getTeamHierarchyLink(auth context.Context, api *datadogV2.TeamsApi, linkID string) (datadogV2.TeamHierarchyLink, error) {
	teamHierarchyLinkResponse, httpResponse, err := api.GetTeamHierarchyLink(auth, linkID)
	if httpResponse != nil && httpResponse.Body != nil {
		defer httpResponse.Body.Close()
	}
	if err != nil {
		return datadogV2.TeamHierarchyLink{}, err
	}

	teamHierarchyLink, ok := teamHierarchyLinkResponse.GetDataOk()
	if !ok {
		return datadogV2.TeamHierarchyLink{}, fmt.Errorf("team hierarchy link %q not found", linkID)
	}
	return *teamHierarchyLink, nil
}

func listTeamHierarchyLinks(auth context.Context, api *datadogV2.TeamsApi, optionalParams datadogV2.ListTeamHierarchyLinksOptionalParameters) ([]datadogV2.TeamHierarchyLink, error) {
	pageSize := int64(100)
	optionalParams.WithPageSize(pageSize)
	items, cancel := api.ListTeamHierarchyLinksWithPagination(auth, optionalParams)
	defer cancel()

	teamHierarchyLinks := []datadogV2.TeamHierarchyLink{}
	for item := range items {
		if item.Error != nil {
			return nil, item.Error
		}
		teamHierarchyLinks = append(teamHierarchyLinks, item.Item)
	}

	return teamHierarchyLinks, nil
}

func teamHierarchyLinkParentTeamID(teamHierarchyLink datadogV2.TeamHierarchyLink) string {
	relationships, ok := teamHierarchyLink.GetRelationshipsOk()
	if !ok {
		return ""
	}
	parentTeam, ok := relationships.GetParentTeamOk()
	if !ok {
		return ""
	}
	data, ok := parentTeam.GetDataOk()
	if !ok {
		return ""
	}
	return data.GetId()
}

func teamHierarchyLinkSubTeamID(teamHierarchyLink datadogV2.TeamHierarchyLink) string {
	relationships, ok := teamHierarchyLink.GetRelationshipsOk()
	if !ok {
		return ""
	}
	subTeam, ok := relationships.GetSubTeamOk()
	if !ok {
		return ""
	}
	data, ok := subTeam.GetDataOk()
	if !ok {
		return ""
	}
	return data.GetId()
}
