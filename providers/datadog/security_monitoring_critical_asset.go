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
	if response.UnparsedObject != nil {
		if rawData, ok := response.UnparsedObject["data"]; ok {
			criticalAsset, err := securityMonitoringCriticalAssetFromRawData(rawData)
			if err != nil {
				return datadogV2.SecurityMonitoringCriticalAsset{}, err
			}
			return criticalAsset, nil
		}
		return datadogV2.SecurityMonitoringCriticalAsset{}, fmt.Errorf("security monitoring critical asset raw response missing data")
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
	if len(criticalAssets) > 0 {
		return criticalAssets, nil
	}
	if response.UnparsedObject != nil {
		if rawData, ok := response.UnparsedObject["data"]; ok {
			return securityMonitoringCriticalAssetsFromRawData(rawData)
		}
		return nil, fmt.Errorf("security monitoring critical assets raw response missing data")
	}
	return criticalAssets, nil
}

func securityMonitoringCriticalAssetsFromRawData(rawData interface{}) ([]datadogV2.SecurityMonitoringCriticalAsset, error) {
	rawCriticalAssets, ok := rawData.([]interface{})
	if !ok {
		return nil, fmt.Errorf("security monitoring critical assets raw data is not a list")
	}

	criticalAssets := []datadogV2.SecurityMonitoringCriticalAsset{}
	for index, rawCriticalAsset := range rawCriticalAssets {
		criticalAsset, err := securityMonitoringCriticalAssetFromRawData(rawCriticalAsset)
		if err != nil {
			return nil, fmt.Errorf("parse security monitoring critical asset raw data[%d]: %w", index, err)
		}
		criticalAssets = append(criticalAssets, criticalAsset)
	}
	return criticalAssets, nil
}

func securityMonitoringCriticalAssetFromRawData(rawData interface{}) (datadogV2.SecurityMonitoringCriticalAsset, error) {
	rawCriticalAsset, ok := rawData.(map[string]interface{})
	if !ok {
		return datadogV2.SecurityMonitoringCriticalAsset{}, fmt.Errorf("raw critical asset is not an object")
	}
	if rawType, ok := rawCriticalAsset["type"].(string); ok && rawType != string(datadogV2.SECURITYMONITORINGCRITICALASSETTYPE_CRITICAL_ASSETS) {
		return datadogV2.SecurityMonitoringCriticalAsset{}, fmt.Errorf("unexpected critical asset type %q", rawType)
	}
	rawID, ok := rawCriticalAsset["id"].(string)
	if !ok || rawID == "" {
		return datadogV2.SecurityMonitoringCriticalAsset{}, fmt.Errorf("raw critical asset missing id")
	}

	criticalAsset := datadogV2.NewSecurityMonitoringCriticalAssetWithDefaults()
	criticalAsset.SetId(rawID)
	return *criticalAsset, nil
}
