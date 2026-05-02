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
	// SecurityMonitoringFilterAllowEmptyValues ...
	SecurityMonitoringFilterAllowEmptyValues = []string{}
)

// SecurityMonitoringFilterGenerator ...
type SecurityMonitoringFilterGenerator struct {
	DatadogService
}

func (g *SecurityMonitoringFilterGenerator) createResources(securityFilters []datadogV2.SecurityFilter) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, securityFilter := range securityFilters {
		resource, err := g.createResource(securityFilter)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *SecurityMonitoringFilterGenerator) createResource(securityFilter datadogV2.SecurityFilter) (terraformutils.Resource, error) {
	securityFilterID := securityFilter.GetId()
	if securityFilterID == "" {
		return terraformutils.Resource{}, fmt.Errorf("security monitoring filter missing id")
	}

	return terraformutils.NewSimpleResource(
		securityFilterID,
		fmt.Sprintf("security_monitoring_filter_%s", securityFilterID),
		"datadog_security_monitoring_filter",
		"datadog",
		SecurityMonitoringFilterAllowEmptyValues,
	), nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each security_monitoring_filter create 1 TerraformResource.
func (g *SecurityMonitoringFilterGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewSecurityMonitoringApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	securityFilters, err := listSecurityMonitoringFilters(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(securityFilters)
	if err != nil {
		return err
	}

	g.Resources = resources
	return nil
}

func (g *SecurityMonitoringFilterGenerator) filteredResources(auth context.Context, api *datadogV2.SecurityMonitoringApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("security_monitoring_filter") {
			continue
		}

		filtered = true
		for _, value := range filter.AcceptableValues {
			securityFilter, err := getSecurityMonitoringFilter(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(securityFilter)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, filtered, nil
}

func getSecurityMonitoringFilter(auth context.Context, api *datadogV2.SecurityMonitoringApi, securityFilterID string) (datadogV2.SecurityFilter, error) {
	response, httpResponse, err := api.GetSecurityFilter(auth, securityFilterID)
	defer closeDatadogResponseBody(httpResponse)
	if err != nil {
		return datadogV2.SecurityFilter{}, err
	}

	if securityFilter, ok := response.GetDataOk(); ok {
		return *securityFilter, nil
	}
	if securityFilter, ok := securityMonitoringFilterFromRawData(response.UnparsedObject["data"]); ok {
		return securityFilter, nil
	}

	return datadogV2.SecurityFilter{}, fmt.Errorf("security monitoring filter %q not found", securityFilterID)
}

func listSecurityMonitoringFilters(auth context.Context, api *datadogV2.SecurityMonitoringApi) ([]datadogV2.SecurityFilter, error) {
	response, httpResponse, err := api.ListSecurityFilters(auth)
	defer closeDatadogResponseBody(httpResponse)
	if err != nil {
		return nil, err
	}

	securityFilters := response.GetData()
	if len(securityFilters) == 0 {
		securityFilters = securityMonitoringFiltersFromRawData(response.UnparsedObject["data"])
	}
	return securityFilters, nil
}

func securityMonitoringFiltersFromRawData(rawData interface{}) []datadogV2.SecurityFilter {
	rawFilters, ok := rawData.([]interface{})
	if !ok {
		return nil
	}

	securityFilters := []datadogV2.SecurityFilter{}
	for _, rawFilter := range rawFilters {
		securityFilter, ok := securityMonitoringFilterFromRawData(rawFilter)
		if !ok {
			continue
		}
		securityFilters = append(securityFilters, securityFilter)
	}
	return securityFilters
}

func securityMonitoringFilterFromRawData(rawData interface{}) (datadogV2.SecurityFilter, bool) {
	rawFilter, ok := rawData.(map[string]interface{})
	if !ok {
		return datadogV2.SecurityFilter{}, false
	}
	if rawType, ok := rawFilter["type"].(string); ok && rawType != string(datadogV2.SECURITYFILTERTYPE_SECURITY_FILTERS) {
		return datadogV2.SecurityFilter{}, false
	}
	rawID, ok := rawFilter["id"].(string)
	if !ok || rawID == "" {
		return datadogV2.SecurityFilter{}, false
	}

	securityFilter := datadogV2.NewSecurityFilterWithDefaults()
	securityFilter.SetId(rawID)
	return *securityFilter, true
}
