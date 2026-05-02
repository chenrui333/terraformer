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
	// MonitorNotificationRuleAllowEmptyValues ...
	MonitorNotificationRuleAllowEmptyValues = []string{}
)

// MonitorNotificationRuleGenerator ...
type MonitorNotificationRuleGenerator struct {
	DatadogService
}

func (g *MonitorNotificationRuleGenerator) createResources(monitorNotificationRules []datadogV2.MonitorNotificationRuleData) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, monitorNotificationRule := range monitorNotificationRules {
		resource, err := g.createResource(monitorNotificationRule)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *MonitorNotificationRuleGenerator) createResource(monitorNotificationRule datadogV2.MonitorNotificationRuleData) (terraformutils.Resource, error) {
	ruleID := monitorNotificationRule.GetId()
	if ruleID == "" {
		return terraformutils.Resource{}, fmt.Errorf("monitor notification rule missing id")
	}

	return terraformutils.NewSimpleResource(
		ruleID,
		fmt.Sprintf("monitor_notification_rule_%s", ruleID),
		"datadog_monitor_notification_rule",
		"datadog",
		MonitorNotificationRuleAllowEmptyValues,
	), nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each monitor notification rule create 1 TerraformResource.
func (g *MonitorNotificationRuleGenerator) InitResources() error {
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

	monitorNotificationRules, err := listMonitorNotificationRules(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(monitorNotificationRules)
	if err != nil {
		return err
	}

	g.Resources = resources
	return nil
}

func (g *MonitorNotificationRuleGenerator) filteredResources(auth context.Context, api *datadogV2.MonitorsApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("monitor_notification_rule") {
			continue
		}

		filtered = true
		for _, value := range filter.AcceptableValues {
			monitorNotificationRule, err := getMonitorNotificationRule(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(monitorNotificationRule)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, filtered, nil
}

func getMonitorNotificationRule(auth context.Context, api *datadogV2.MonitorsApi, ruleID string) (datadogV2.MonitorNotificationRuleData, error) {
	response, httpResponse, err := api.GetMonitorNotificationRule(auth, ruleID)
	defer closeDatadogResponseBody(httpResponse)
	if err != nil {
		return datadogV2.MonitorNotificationRuleData{}, err
	}

	if rule, ok := response.GetDataOk(); ok {
		return *rule, nil
	}
	if rule, ok := monitorNotificationRuleFromRawData(response.UnparsedObject["data"]); ok {
		return rule, nil
	}

	return datadogV2.MonitorNotificationRuleData{}, fmt.Errorf("monitor notification rule %q not found", ruleID)
}

func listMonitorNotificationRules(auth context.Context, api *datadogV2.MonitorsApi) ([]datadogV2.MonitorNotificationRuleData, error) {
	monitorNotificationRules := []datadogV2.MonitorNotificationRuleData{}
	const pageSize = 1000
	var page int32

	for {
		optionalParams := datadogV2.NewGetMonitorNotificationRulesOptionalParameters().
			WithPage(page).
			WithPerPage(pageSize)

		response, httpResponse, err := api.GetMonitorNotificationRules(auth, *optionalParams)
		closeDatadogResponseBody(httpResponse)
		if err != nil {
			return nil, err
		}

		rules := response.GetData()
		if len(rules) == 0 {
			rules = monitorNotificationRulesFromRawData(response.UnparsedObject["data"])
		}
		monitorNotificationRules = append(monitorNotificationRules, rules...)
		if len(rules) < pageSize {
			break
		}
		page++
	}

	return monitorNotificationRules, nil
}

func monitorNotificationRulesFromRawData(rawData interface{}) []datadogV2.MonitorNotificationRuleData {
	rawRules, ok := rawData.([]interface{})
	if !ok {
		return nil
	}

	monitorNotificationRules := []datadogV2.MonitorNotificationRuleData{}
	for _, rawRule := range rawRules {
		monitorNotificationRule, ok := monitorNotificationRuleFromRawData(rawRule)
		if !ok {
			continue
		}
		monitorNotificationRules = append(monitorNotificationRules, monitorNotificationRule)
	}
	return monitorNotificationRules
}

func monitorNotificationRuleFromRawData(rawData interface{}) (datadogV2.MonitorNotificationRuleData, bool) {
	rawRule, ok := rawData.(map[string]interface{})
	if !ok {
		return datadogV2.MonitorNotificationRuleData{}, false
	}

	rawRuleID, ok := rawRule["id"].(string)
	if !ok || rawRuleID == "" {
		return datadogV2.MonitorNotificationRuleData{}, false
	}

	monitorNotificationRule := datadogV2.NewMonitorNotificationRuleDataWithDefaults()
	monitorNotificationRule.SetId(rawRuleID)
	return *monitorNotificationRule, true
}

func closeDatadogResponseBody(response *http.Response) {
	if response != nil && response.Body != nil {
		response.Body.Close()
	}
}
