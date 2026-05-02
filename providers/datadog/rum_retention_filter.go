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

const rumRetentionFilterImportIDDelimiter = ":"

var (
	// RumRetentionFilterAllowEmptyValues ...
	RumRetentionFilterAllowEmptyValues = []string{"query"}
)

// RumRetentionFilterGenerator ...
type RumRetentionFilterGenerator struct {
	DatadogService
}

func (g *RumRetentionFilterGenerator) createResources(applicationID string, rumRetentionFilters []datadogV2.RumRetentionFilterData) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, rumRetentionFilter := range rumRetentionFilters {
		resource, err := g.createResource(applicationID, rumRetentionFilter)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *RumRetentionFilterGenerator) createResource(applicationID string, rumRetentionFilter datadogV2.RumRetentionFilterData) (terraformutils.Resource, error) {
	retentionFilterID := rumRetentionFilter.GetId()
	if retentionFilterID == "" {
		return terraformutils.Resource{}, fmt.Errorf("RUM retention filter missing id")
	}
	if applicationID == "" {
		return terraformutils.Resource{}, fmt.Errorf("RUM retention filter %q missing application id", retentionFilterID)
	}

	return terraformutils.NewResource(
		retentionFilterID,
		fmt.Sprintf("rum_retention_filter_%s_%s", applicationID, retentionFilterID),
		"datadog_rum_retention_filter",
		"datadog",
		map[string]string{
			"application_id": applicationID,
		},
		RumRetentionFilterAllowEmptyValues,
		map[string]interface{}{},
	), nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each RUM retention filter create 1 TerraformResource.
// Need RUM retention filter ID formatted as '<application_id>:<retention_filter_id>' for filter lookup.
func (g *RumRetentionFilterGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	rumAPI := datadogV2.NewRUMApi(datadogClient)
	retentionFilterAPI := datadogV2.NewRumRetentionFiltersApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, retentionFilterAPI)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	applications, err := listRumApplications(auth, rumAPI)
	if err != nil {
		return err
	}
	for _, application := range applications {
		applicationID := rumApplicationID(application)
		if applicationID == "" {
			continue
		}
		rumRetentionFilters, err := listRumRetentionFilters(auth, retentionFilterAPI, applicationID)
		if err != nil {
			return err
		}
		applicationResources, err := g.createResources(applicationID, rumRetentionFilters)
		if err != nil {
			return err
		}
		resources = append(resources, applicationResources...)
	}

	g.Resources = resources
	return nil
}

func (g *RumRetentionFilterGenerator) filteredResources(auth context.Context, api *datadogV2.RumRetentionFiltersApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for filterIndex, filter := range g.Filter {
		if !filter.IsApplicable("rum_retention_filter") {
			continue
		}

		switch filter.FieldPath {
		case "id":
			filtered = true
			filterIDs, err := parseRumRetentionFilterImportIDs(filter.AcceptableValues)
			if err != nil {
				return nil, true, err
			}
			for _, filterID := range filterIDs {
				rumRetentionFilter, err := getRumRetentionFilter(auth, api, filterID.applicationID, filterID.retentionFilterID)
				if err != nil {
					return nil, true, err
				}
				resource, err := g.createResource(filterID.applicationID, rumRetentionFilter)
				if err != nil {
					return nil, true, err
				}
				resources = append(resources, resource)
			}
			g.Filter[filterIndex].AcceptableValues = rumRetentionFilterIDs(filterIDs)
		case "application_id":
			filtered = true
			for _, applicationID := range filter.AcceptableValues {
				rumRetentionFilters, err := listRumRetentionFilters(auth, api, applicationID)
				if err != nil {
					return nil, true, err
				}
				applicationResources, err := g.createResources(applicationID, rumRetentionFilters)
				if err != nil {
					return nil, true, err
				}
				resources = append(resources, applicationResources...)
			}
		}
	}

	return resources, filtered, nil
}

type rumRetentionFilterImportID struct {
	applicationID     string
	retentionFilterID string
}

func parseRumRetentionFilterImportIDs(importIDs []string) ([]rumRetentionFilterImportID, error) {
	filterIDs := []rumRetentionFilterImportID{}
	for _, importID := range importIDs {
		applicationID, retentionFilterID, err := parseRumRetentionFilterImportID(importID)
		if err != nil {
			return nil, err
		}
		filterIDs = append(filterIDs, rumRetentionFilterImportID{applicationID: applicationID, retentionFilterID: retentionFilterID})
	}
	return filterIDs, nil
}

func rumRetentionFilterIDs(filterIDs []rumRetentionFilterImportID) []string {
	retentionFilterIDs := []string{}
	for _, filterID := range filterIDs {
		retentionFilterIDs = append(retentionFilterIDs, filterID.retentionFilterID)
	}
	return retentionFilterIDs
}

func parseRumRetentionFilterImportID(importID string) (string, string, error) {
	parts := strings.SplitN(importID, rumRetentionFilterImportIDDelimiter, 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("RUM retention filter import ID %q must be formatted as application_id:retention_filter_id", importID)
	}
	return parts[0], parts[1], nil
}

func getRumRetentionFilter(auth context.Context, api *datadogV2.RumRetentionFiltersApi, applicationID string, retentionFilterID string) (datadogV2.RumRetentionFilterData, error) {
	rumRetentionFilterResponse, httpResponse, err := api.GetRetentionFilter(auth, applicationID, retentionFilterID)
	if httpResponse != nil && httpResponse.Body != nil {
		defer httpResponse.Body.Close()
	}
	if err != nil {
		return datadogV2.RumRetentionFilterData{}, err
	}

	rumRetentionFilter := rumRetentionFilterResponse.GetData()
	if rumRetentionFilter.GetId() == "" {
		rumRetentionFilter.SetId(retentionFilterID)
	}
	return rumRetentionFilter, nil
}

func listRumRetentionFilters(auth context.Context, api *datadogV2.RumRetentionFiltersApi, applicationID string) ([]datadogV2.RumRetentionFilterData, error) {
	rumRetentionFiltersResponse, httpResponse, err := api.ListRetentionFilters(auth, applicationID)
	if httpResponse != nil && httpResponse.Body != nil {
		defer httpResponse.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	return rumRetentionFiltersResponse.GetData(), nil
}

func listRumApplications(auth context.Context, api *datadogV2.RUMApi) ([]datadogV2.RUMApplicationList, error) {
	rumApplicationsResponse, httpResponse, err := api.GetRUMApplications(auth)
	if httpResponse != nil && httpResponse.Body != nil {
		defer httpResponse.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	return rumApplicationsResponse.GetData(), nil
}

func rumApplicationID(application datadogV2.RUMApplicationList) string {
	applicationID := application.GetId()
	if applicationID != "" {
		return applicationID
	}
	attributes := application.GetAttributes()
	return attributes.GetApplicationId()
}
