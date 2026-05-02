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
	// MonitorConfigPolicyAllowEmptyValues ...
	MonitorConfigPolicyAllowEmptyValues = []string{}
)

// MonitorConfigPolicyGenerator ...
type MonitorConfigPolicyGenerator struct {
	DatadogService
}

func (g *MonitorConfigPolicyGenerator) createResources(monitorConfigPolicies []datadogV2.MonitorConfigPolicyResponseData) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, monitorConfigPolicy := range monitorConfigPolicies {
		resource, err := g.createResource(monitorConfigPolicy)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *MonitorConfigPolicyGenerator) createResource(monitorConfigPolicy datadogV2.MonitorConfigPolicyResponseData) (terraformutils.Resource, error) {
	policyID := monitorConfigPolicy.GetId()
	if policyID == "" {
		return terraformutils.Resource{}, fmt.Errorf("monitor config policy missing id")
	}

	return terraformutils.NewSimpleResource(
		policyID,
		fmt.Sprintf("monitor_config_policy_%s", policyID),
		"datadog_monitor_config_policy",
		"datadog",
		MonitorConfigPolicyAllowEmptyValues,
	), nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each monitor config policy create 1 TerraformResource.
func (g *MonitorConfigPolicyGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewMonitorsApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	monitorConfigPolicies, err := listMonitorConfigPolicies(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(monitorConfigPolicies)
	if err != nil {
		return err
	}

	g.Resources = resources
	return nil
}

func (g *MonitorConfigPolicyGenerator) filteredResources(auth context.Context, api *datadogV2.MonitorsApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("monitor_config_policy") {
			continue
		}

		filtered = true
		for _, value := range filter.AcceptableValues {
			monitorConfigPolicy, err := getMonitorConfigPolicy(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(monitorConfigPolicy)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, filtered, nil
}

func getMonitorConfigPolicy(auth context.Context, api *datadogV2.MonitorsApi, policyID string) (datadogV2.MonitorConfigPolicyResponseData, error) {
	response, httpResponse, err := api.GetMonitorConfigPolicy(auth, policyID)
	defer closeDatadogResponseBody(httpResponse)
	if err != nil {
		return datadogV2.MonitorConfigPolicyResponseData{}, err
	}

	if policy, ok := response.GetDataOk(); ok {
		return *policy, nil
	}
	if policy, ok := monitorConfigPolicyFromRawData(response.UnparsedObject["data"]); ok {
		return policy, nil
	}

	return datadogV2.MonitorConfigPolicyResponseData{}, fmt.Errorf("monitor config policy %q not found", policyID)
}

func listMonitorConfigPolicies(auth context.Context, api *datadogV2.MonitorsApi) ([]datadogV2.MonitorConfigPolicyResponseData, error) {
	response, httpResponse, err := api.ListMonitorConfigPolicies(auth)
	defer closeDatadogResponseBody(httpResponse)
	if err != nil {
		return nil, err
	}

	policies := response.GetData()
	if len(policies) == 0 {
		policies = monitorConfigPoliciesFromRawData(response.UnparsedObject["data"])
	}

	return policies, nil
}

func monitorConfigPoliciesFromRawData(rawData interface{}) []datadogV2.MonitorConfigPolicyResponseData {
	rawPolicies, ok := rawData.([]interface{})
	if !ok {
		return nil
	}

	monitorConfigPolicies := []datadogV2.MonitorConfigPolicyResponseData{}
	for _, rawPolicy := range rawPolicies {
		monitorConfigPolicy, ok := monitorConfigPolicyFromRawData(rawPolicy)
		if !ok {
			continue
		}
		monitorConfigPolicies = append(monitorConfigPolicies, monitorConfigPolicy)
	}
	return monitorConfigPolicies
}

func monitorConfigPolicyFromRawData(rawData interface{}) (datadogV2.MonitorConfigPolicyResponseData, bool) {
	rawPolicy, ok := rawData.(map[string]interface{})
	if !ok {
		return datadogV2.MonitorConfigPolicyResponseData{}, false
	}

	rawPolicyID, ok := rawPolicy["id"].(string)
	if !ok || rawPolicyID == "" {
		return datadogV2.MonitorConfigPolicyResponseData{}, false
	}

	monitorConfigPolicy := datadogV2.NewMonitorConfigPolicyResponseDataWithDefaults()
	monitorConfigPolicy.SetId(rawPolicyID)
	return *monitorConfigPolicy, true
}
