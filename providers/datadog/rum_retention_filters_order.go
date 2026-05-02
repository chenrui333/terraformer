// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"strconv"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// RumRetentionFiltersOrderAllowEmptyValues ...
	RumRetentionFiltersOrderAllowEmptyValues = []string{"retention_filter_ids"}
)

// RumRetentionFiltersOrderGenerator ...
type RumRetentionFiltersOrderGenerator struct {
	DatadogService
}

func (g *RumRetentionFiltersOrderGenerator) createResource(applicationID string, rumRetentionFilters []datadogV2.RumRetentionFilterData) (terraformutils.Resource, error) {
	if applicationID == "" {
		return terraformutils.Resource{}, fmt.Errorf("RUM retention filters order missing application id")
	}
	retentionFilterIDs, err := rumRetentionFilterIDsFromData(applicationID, rumRetentionFilters)
	if err != nil {
		return terraformutils.Resource{}, err
	}
	attributes := map[string]string{
		"application_id":         applicationID,
		"retention_filter_ids.#": strconv.Itoa(len(retentionFilterIDs)),
	}
	for index, retentionFilterID := range retentionFilterIDs {
		attributes[fmt.Sprintf("retention_filter_ids.%d", index)] = retentionFilterID
	}

	return terraformutils.NewResource(
		applicationID,
		fmt.Sprintf("rum_retention_filters_order_%s", applicationID),
		"datadog_rum_retention_filters_order",
		"datadog",
		attributes,
		RumRetentionFiltersOrderAllowEmptyValues,
		map[string]interface{}{},
	), nil
}

func (g *RumRetentionFiltersOrderGenerator) PostConvertHook() error {
	for i := range g.Resources {
		resource := &g.Resources[i]
		if resource.Item == nil {
			resource.Item = map[string]interface{}{}
		}
		if _, ok := resource.Item["retention_filter_ids"]; ok {
			continue
		}
		if !rumRetentionFiltersOrderStateHasEmptyRetentionFilterIDs(resource) {
			continue
		}
		resource.Item["retention_filter_ids"] = []interface{}{}
	}
	return nil
}

func rumRetentionFiltersOrderStateHasEmptyRetentionFilterIDs(resource *terraformutils.Resource) bool {
	if resource == nil || resource.InstanceState == nil || resource.InstanceState.Attributes == nil {
		return false
	}
	return resource.InstanceState.Attributes["retention_filter_ids.#"] == "0"
}

// InitResources Generate TerraformResources from Datadog API,
// from each RUM application create 1 RUM retention filters order TerraformResource.
func (g *RumRetentionFiltersOrderGenerator) InitResources() error {
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
		resource, err := g.resourceForApplication(auth, retentionFilterAPI, applicationID)
		if err != nil {
			return err
		}
		resources = append(resources, resource)
	}

	g.Resources = resources
	return nil
}

func (g *RumRetentionFiltersOrderGenerator) filteredResources(auth context.Context, api *datadogV2.RumRetentionFiltersApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for _, filter := range g.Filter {
		if !filter.IsApplicable("rum_retention_filters_order") {
			continue
		}

		switch filter.FieldPath {
		case "id", "application_id":
			filtered = true
			for _, applicationID := range filter.AcceptableValues {
				resource, err := g.resourceForApplication(auth, api, applicationID)
				if err != nil {
					return nil, true, err
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources, filtered, nil
}

func (g *RumRetentionFiltersOrderGenerator) resourceForApplication(auth context.Context, api *datadogV2.RumRetentionFiltersApi, applicationID string) (terraformutils.Resource, error) {
	rumRetentionFilters, err := listRumRetentionFilters(auth, api, applicationID)
	if err != nil {
		return terraformutils.Resource{}, err
	}
	return g.createResource(applicationID, rumRetentionFilters)
}

func rumRetentionFilterIDsFromData(applicationID string, rumRetentionFilters []datadogV2.RumRetentionFilterData) ([]string, error) {
	retentionFilterIDs := []string{}
	for _, rumRetentionFilter := range rumRetentionFilters {
		retentionFilterID := rumRetentionFilter.GetId()
		if retentionFilterID == "" {
			return nil, fmt.Errorf("RUM retention filter order for application %q has retention filter missing id", applicationID)
		}
		retentionFilterIDs = append(retentionFilterIDs, retentionFilterID)
	}
	return retentionFilterIDs, nil
}
