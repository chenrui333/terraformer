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
	// SecurityMonitoringCriticalAssetAllowEmptyValues ...
	SecurityMonitoringCriticalAssetAllowEmptyValues = []string{"query", "rule_query", "tags."}
)

// SecurityMonitoringCriticalAssetGenerator ...
type SecurityMonitoringCriticalAssetGenerator struct {
	DatadogService
}

func (g *SecurityMonitoringCriticalAssetGenerator) createResources(criticalAssets []datadogV2.SecurityMonitoringCriticalAsset) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, criticalAsset := range criticalAssets {
		resource, err := g.createResource(criticalAsset)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *SecurityMonitoringCriticalAssetGenerator) createResource(criticalAsset datadogV2.SecurityMonitoringCriticalAsset) (terraformutils.Resource, error) {
	criticalAssetID := criticalAsset.GetId()
	if criticalAssetID == "" {
		return terraformutils.Resource{}, fmt.Errorf("security monitoring critical asset missing id")
	}

	return terraformutils.NewSimpleResource(
		criticalAssetID,
		fmt.Sprintf("security_monitoring_critical_asset_%s", criticalAssetID),
		"datadog_security_monitoring_critical_asset",
		"datadog",
		SecurityMonitoringCriticalAssetAllowEmptyValues,
	), nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each security_monitoring_critical_asset create 1 TerraformResource.
func (g *SecurityMonitoringCriticalAssetGenerator) InitResources() error {
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

	criticalAssets, err := listSecurityMonitoringCriticalAssets(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(criticalAssets)
	if err != nil {
		return err
	}

	g.Resources = resources
	return nil
}

func (g *SecurityMonitoringCriticalAssetGenerator) filteredResources(auth context.Context, api *datadogV2.SecurityMonitoringApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("security_monitoring_critical_asset") {
			continue
		}

		filtered = true
		for _, value := range filter.AcceptableValues {
			criticalAsset, err := getSecurityMonitoringCriticalAsset(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(criticalAsset)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, filtered, nil
}

func getSecurityMonitoringCriticalAsset(auth context.Context, api *datadogV2.SecurityMonitoringApi, criticalAssetID string) (datadogV2.SecurityMonitoringCriticalAsset, error) {
	response, httpResponse, err := api.GetSecurityMonitoringCriticalAsset(auth, criticalAssetID)
	defer closeDatadogResponseBody(httpResponse)
	if err != nil {
		return datadogV2.SecurityMonitoringCriticalAsset{}, err
	}

	if criticalAsset, ok := response.GetDataOk(); ok {
		return *criticalAsset, nil
	}
	if criticalAsset, ok := securityMonitoringCriticalAssetFromRawData(response.UnparsedObject["data"]); ok {
		return criticalAsset, nil
	}

	return datadogV2.SecurityMonitoringCriticalAsset{}, fmt.Errorf("security monitoring critical asset %q not found", criticalAssetID)
}

func listSecurityMonitoringCriticalAssets(auth context.Context, api *datadogV2.SecurityMonitoringApi) ([]datadogV2.SecurityMonitoringCriticalAsset, error) {
	response, httpResponse, err := api.ListSecurityMonitoringCriticalAssets(auth)
	defer closeDatadogResponseBody(httpResponse)
	if err != nil {
		return nil, err
	}

	criticalAssets := response.GetData()
	if len(criticalAssets) == 0 {
		criticalAssets = securityMonitoringCriticalAssetsFromRawData(response.UnparsedObject["data"])
	}
	return criticalAssets, nil
}

func securityMonitoringCriticalAssetsFromRawData(rawData interface{}) []datadogV2.SecurityMonitoringCriticalAsset {
	rawCriticalAssets, ok := rawData.([]interface{})
	if !ok {
		return nil
	}

	criticalAssets := []datadogV2.SecurityMonitoringCriticalAsset{}
	for _, rawCriticalAsset := range rawCriticalAssets {
		criticalAsset, ok := securityMonitoringCriticalAssetFromRawData(rawCriticalAsset)
		if !ok {
			continue
		}
		criticalAssets = append(criticalAssets, criticalAsset)
	}
	return criticalAssets
}

func securityMonitoringCriticalAssetFromRawData(rawData interface{}) (datadogV2.SecurityMonitoringCriticalAsset, bool) {
	rawCriticalAsset, ok := rawData.(map[string]interface{})
	if !ok {
		return datadogV2.SecurityMonitoringCriticalAsset{}, false
	}
	if rawType, ok := rawCriticalAsset["type"].(string); ok && rawType != string(datadogV2.SECURITYMONITORINGCRITICALASSETTYPE_CRITICAL_ASSETS) {
		return datadogV2.SecurityMonitoringCriticalAsset{}, false
	}
	rawID, ok := rawCriticalAsset["id"].(string)
	if !ok || rawID == "" {
		return datadogV2.SecurityMonitoringCriticalAsset{}, false
	}

	criticalAsset := datadogV2.NewSecurityMonitoringCriticalAssetWithDefaults()
	criticalAsset.SetId(rawID)
	return *criticalAsset, true
}
