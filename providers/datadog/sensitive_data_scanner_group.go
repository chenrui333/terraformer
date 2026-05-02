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
	// SensitiveDataScannerGroupAllowEmptyValues ...
	SensitiveDataScannerGroupAllowEmptyValues = []string{"filter.*.query", "product_list"}
)

// SensitiveDataScannerGroupGenerator ...
type SensitiveDataScannerGroupGenerator struct {
	DatadogService
}

func (g *SensitiveDataScannerGroupGenerator) createResources(groups []datadogV2.SensitiveDataScannerGroupIncludedItem) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, group := range groups {
		resource, err := g.createResource(group)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *SensitiveDataScannerGroupGenerator) createResource(group datadogV2.SensitiveDataScannerGroupIncludedItem) (terraformutils.Resource, error) {
	groupID := group.GetId()
	if groupID == "" {
		return terraformutils.Resource{}, fmt.Errorf("sensitive data scanner group missing id")
	}

	return terraformutils.NewSimpleResource(
		groupID,
		fmt.Sprintf("sensitive_data_scanner_group_%s", groupID),
		"datadog_sensitive_data_scanner_group",
		"datadog",
		SensitiveDataScannerGroupAllowEmptyValues,
	), nil
}

func (g *SensitiveDataScannerGroupGenerator) PostConvertHook() error {
	for i := range g.Resources {
		resource := &g.Resources[i]
		if resource.Item == nil {
			resource.Item = map[string]interface{}{}
		}
		if _, ok := resource.Item["product_list"]; ok {
			continue
		}
		if !sensitiveDataScannerStateHasEmptyList(resource, "product_list") {
			continue
		}
		resource.Item["product_list"] = []interface{}{}
	}
	return nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each sensitive_data_scanner_group create 1 TerraformResource.
func (g *SensitiveDataScannerGroupGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewSensitiveDataScannerApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	config, err := listSensitiveDataScannerConfig(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(sensitiveDataScannerGroupsFromConfig(config))
	if err != nil {
		return err
	}

	g.Resources = resources
	return nil
}

func (g *SensitiveDataScannerGroupGenerator) filteredResources(auth context.Context, api *datadogV2.SensitiveDataScannerApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false
	var config *datadogV2.SensitiveDataScannerGetConfigResponse

	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("sensitive_data_scanner_group") {
			continue
		}

		filtered = true
		if config == nil {
			response, err := listSensitiveDataScannerConfig(auth, api)
			if err != nil {
				return nil, true, err
			}
			config = &response
		}
		for _, value := range filter.AcceptableValues {
			group, ok := sensitiveDataScannerGroupByID(*config, value)
			if !ok {
				return nil, true, fmt.Errorf("sensitive data scanner group %q not found", value)
			}
			resource, err := g.createResource(group)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, filtered, nil
}

func listSensitiveDataScannerConfig(auth context.Context, api *datadogV2.SensitiveDataScannerApi) (datadogV2.SensitiveDataScannerGetConfigResponse, error) {
	response, httpResponse, err := api.ListScanningGroups(auth)
	defer closeDatadogResponseBody(httpResponse)
	if err != nil {
		return datadogV2.SensitiveDataScannerGetConfigResponse{}, err
	}
	return response, nil
}

func sensitiveDataScannerGroupsFromConfig(config datadogV2.SensitiveDataScannerGetConfigResponse) []datadogV2.SensitiveDataScannerGroupIncludedItem {
	groups := []datadogV2.SensitiveDataScannerGroupIncludedItem{}
	for _, included := range config.GetIncluded() {
		if included.SensitiveDataScannerGroupIncludedItem == nil {
			continue
		}
		groups = append(groups, *included.SensitiveDataScannerGroupIncludedItem)
	}
	return groups
}

func sensitiveDataScannerGroupByID(config datadogV2.SensitiveDataScannerGetConfigResponse, groupID string) (datadogV2.SensitiveDataScannerGroupIncludedItem, bool) {
	for _, group := range sensitiveDataScannerGroupsFromConfig(config) {
		if group.GetId() == groupID {
			return group, true
		}
	}
	return datadogV2.SensitiveDataScannerGroupIncludedItem{}, false
}

func sensitiveDataScannerStateHasEmptyList(resource *terraformutils.Resource, key string) bool {
	if resource == nil || resource.InstanceState == nil || resource.InstanceState.Attributes == nil {
		return false
	}
	return resource.InstanceState.Attributes[key+".#"] == "0"
}
