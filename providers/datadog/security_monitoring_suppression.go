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
	// SecurityMonitoringSuppressionAllowEmptyValues ...
	SecurityMonitoringSuppressionAllowEmptyValues = []string{"rule_query", "suppression_query"}
)

// SecurityMonitoringSuppressionGenerator ...
type SecurityMonitoringSuppressionGenerator struct {
	DatadogService
}

func (g *SecurityMonitoringSuppressionGenerator) createResources(suppressions []datadogV2.SecurityMonitoringSuppression) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, suppression := range suppressions {
		resource, err := g.createResource(suppression)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *SecurityMonitoringSuppressionGenerator) createResource(suppression datadogV2.SecurityMonitoringSuppression) (terraformutils.Resource, error) {
	suppressionID := suppression.GetId()
	if suppressionID == "" {
		return terraformutils.Resource{}, fmt.Errorf("security monitoring suppression missing id")
	}

	return terraformutils.NewSimpleResource(
		suppressionID,
		fmt.Sprintf("security_monitoring_suppression_%s", suppressionID),
		"datadog_security_monitoring_suppression",
		"datadog",
		SecurityMonitoringSuppressionAllowEmptyValues,
	), nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each security_monitoring_suppression create 1 TerraformResource.
func (g *SecurityMonitoringSuppressionGenerator) InitResources() error {
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

	suppressions, err := listSecurityMonitoringSuppressions(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(suppressions)
	if err != nil {
		return err
	}

	g.Resources = resources
	return nil
}

func (g *SecurityMonitoringSuppressionGenerator) filteredResources(auth context.Context, api *datadogV2.SecurityMonitoringApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("security_monitoring_suppression") {
			continue
		}

		filtered = true
		for _, value := range filter.AcceptableValues {
			suppression, err := getSecurityMonitoringSuppression(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(suppression)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, filtered, nil
}

func getSecurityMonitoringSuppression(auth context.Context, api *datadogV2.SecurityMonitoringApi, suppressionID string) (datadogV2.SecurityMonitoringSuppression, error) {
	response, httpResponse, err := api.GetSecurityMonitoringSuppression(auth, suppressionID)
	defer closeDatadogResponseBody(httpResponse)
	if err != nil {
		return datadogV2.SecurityMonitoringSuppression{}, err
	}

	if suppression, ok := response.GetDataOk(); ok {
		return *suppression, nil
	}
	if suppression, ok := securityMonitoringSuppressionFromRawData(response.UnparsedObject["data"]); ok {
		return suppression, nil
	}

	return datadogV2.SecurityMonitoringSuppression{}, fmt.Errorf("security monitoring suppression %q not found", suppressionID)
}

func listSecurityMonitoringSuppressions(auth context.Context, api *datadogV2.SecurityMonitoringApi) ([]datadogV2.SecurityMonitoringSuppression, error) {
	suppressions := []datadogV2.SecurityMonitoringSuppression{}
	const pageSize int64 = 100
	var pageNumber int64

	for {
		optionalParams := datadogV2.NewListSecurityMonitoringSuppressionsOptionalParameters().
			WithPageSize(pageSize).
			WithPageNumber(pageNumber)

		response, httpResponse, err := api.ListSecurityMonitoringSuppressions(auth, *optionalParams)
		closeDatadogResponseBody(httpResponse)
		if err != nil {
			return nil, err
		}

		pageSuppressions := response.GetData()
		if len(pageSuppressions) == 0 {
			pageSuppressions = securityMonitoringSuppressionsFromRawData(response.UnparsedObject["data"])
		}
		suppressions = append(suppressions, pageSuppressions...)

		if !securityMonitoringSuppressionsHasNextPage(response, pageNumber, pageSize, len(pageSuppressions)) {
			break
		}
		pageNumber++
	}

	return suppressions, nil
}

func securityMonitoringSuppressionsHasNextPage(response datadogV2.SecurityMonitoringPaginatedSuppressionsResponse, pageNumber int64, pageSize int64, pageItems int) bool {
	if meta, ok := response.GetMetaOk(); ok {
		page := meta.GetPage()
		if totalCount, ok := page.GetTotalCountOk(); ok {
			return *totalCount > pageSize*(pageNumber+1)
		}
	}
	return pageItems == int(pageSize)
}

func securityMonitoringSuppressionsFromRawData(rawData interface{}) []datadogV2.SecurityMonitoringSuppression {
	rawSuppressions, ok := rawData.([]interface{})
	if !ok {
		return nil
	}

	suppressions := []datadogV2.SecurityMonitoringSuppression{}
	for _, rawSuppression := range rawSuppressions {
		suppression, ok := securityMonitoringSuppressionFromRawData(rawSuppression)
		if !ok {
			continue
		}
		suppressions = append(suppressions, suppression)
	}
	return suppressions
}

func securityMonitoringSuppressionFromRawData(rawData interface{}) (datadogV2.SecurityMonitoringSuppression, bool) {
	rawSuppression, ok := rawData.(map[string]interface{})
	if !ok {
		return datadogV2.SecurityMonitoringSuppression{}, false
	}
	if rawType, ok := rawSuppression["type"].(string); ok && rawType != string(datadogV2.SECURITYMONITORINGSUPPRESSIONTYPE_SUPPRESSIONS) {
		return datadogV2.SecurityMonitoringSuppression{}, false
	}
	rawID, ok := rawSuppression["id"].(string)
	if !ok || rawID == "" {
		return datadogV2.SecurityMonitoringSuppression{}, false
	}

	suppression := datadogV2.NewSecurityMonitoringSuppressionWithDefaults()
	suppression.SetId(rawID)
	return *suppression, true
}
