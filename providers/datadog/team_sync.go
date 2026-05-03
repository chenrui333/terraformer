// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"net/http"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// TeamSyncAllowEmptyValues ...
	TeamSyncAllowEmptyValues = []string{}
)

// TeamSyncGenerator ...
type TeamSyncGenerator struct {
	DatadogService
}

// InitResources Generate TerraformResources from Datadog API,
// from each configured team sync source create 1 TerraformResource.
// Need Team Sync source as ID for terraform resource.
func (g *TeamSyncGenerator) InitResources() error {
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

	teamSync, found, err := getTeamSync(auth, api, datadogV2.TEAMSYNCATTRIBUTESSOURCE_GITHUB)
	if err != nil {
		return err
	}
	if found {
		resource, err := g.createResource(teamSync)
		if err != nil {
			return err
		}
		g.Resources = []terraformutils.Resource{resource}
	}
	return nil
}

func (g *TeamSyncGenerator) filteredResources(auth context.Context, api *datadogV2.TeamsApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for _, filter := range g.Filter {
		if !filter.IsApplicable("team_sync") {
			continue
		}

		switch filter.FieldPath {
		case "id", "source":
			filtered = true
			for _, value := range filter.AcceptableValues {
				teamSync, found, err := getTeamSync(auth, api, datadogV2.TeamSyncAttributesSource(value))
				if err != nil {
					return nil, true, err
				}
				if !found {
					continue
				}
				resource, err := g.createResource(teamSync)
				if err != nil {
					return nil, true, err
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources, filtered, nil
}

func (g *TeamSyncGenerator) createResource(teamSync datadogV2.TeamSyncData) (terraformutils.Resource, error) {
	attributes := teamSync.GetAttributes()
	source := string(attributes.GetSource())
	if source == "" {
		return terraformutils.Resource{}, fmt.Errorf("team sync missing source")
	}
	syncType := string(attributes.GetType())
	if syncType == "" {
		return terraformutils.Resource{}, fmt.Errorf("team sync %q missing type", source)
	}

	return terraformutils.NewResource(
		source,
		fmt.Sprintf("team_sync_%s", source),
		"datadog_team_sync",
		"datadog",
		map[string]string{
			"source": source,
			"type":   syncType,
		},
		TeamSyncAllowEmptyValues,
		map[string]interface{}{},
	), nil
}

func getTeamSync(auth context.Context, api *datadogV2.TeamsApi, source datadogV2.TeamSyncAttributesSource) (datadogV2.TeamSyncData, bool, error) {
	teamSyncResponse, httpResponse, err := api.GetTeamSync(auth, source)
	if httpResponse != nil && httpResponse.Body != nil {
		defer httpResponse.Body.Close()
	}
	if err != nil {
		if httpResponse != nil && httpResponse.StatusCode == http.StatusNotFound {
			return datadogV2.TeamSyncData{}, false, nil
		}
		return datadogV2.TeamSyncData{}, false, err
	}

	teamSyncData := teamSyncResponse.GetData()
	if len(teamSyncData) == 0 {
		return datadogV2.TeamSyncData{}, false, nil
	}
	return teamSyncData[0], true, nil
}
