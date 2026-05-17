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
	// SecurityNotificationRuleAllowEmptyValues ...
	SecurityNotificationRuleAllowEmptyValues = []string{"selectors.query"}
)

// SecurityNotificationRuleGenerator ...
type SecurityNotificationRuleGenerator struct {
	DatadogService
}

func (g *SecurityNotificationRuleGenerator) createResources(notificationRules []datadogV2.NotificationRule) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, notificationRule := range notificationRules {
		resource, err := g.createResource(notificationRule)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *SecurityNotificationRuleGenerator) createResource(notificationRule datadogV2.NotificationRule) (terraformutils.Resource, error) {
	notificationRuleID := notificationRule.Id
	if notificationRuleID == "" {
		return terraformutils.Resource{}, fmt.Errorf("security notification rule missing id")
	}

	return terraformutils.NewSimpleResource(
		notificationRuleID,
		fmt.Sprintf("security_notification_rule_%s", notificationRuleID),
		"datadog_security_notification_rule",
		"datadog",
		SecurityNotificationRuleAllowEmptyValues,
	), nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each security_notification_rule create 1 TerraformResource.
func (g *SecurityNotificationRuleGenerator) InitResources() error {
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

	notificationRules, err := listSecurityNotificationRules(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(notificationRules)
	if err != nil {
		return err
	}

	g.Resources = resources
	return nil
}

func (g *SecurityNotificationRuleGenerator) filteredResources(auth context.Context, api *datadogV2.SecurityMonitoringApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("security_notification_rule") {
			continue
		}

		filtered = true
		for _, value := range filter.AcceptableValues {
			notificationRule, err := getSecurityNotificationRule(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(notificationRule)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, filtered, nil
}

func getSecurityNotificationRule(auth context.Context, api *datadogV2.SecurityMonitoringApi, notificationRuleID string) (datadogV2.NotificationRule, error) {
	response, httpResponse, err := api.GetSignalNotificationRule(auth, notificationRuleID)
	defer closeDatadogResponseBody(httpResponse)
	if httpResponse != nil && httpResponse.StatusCode == http.StatusNotFound {
		response, httpResponse, err = api.GetVulnerabilityNotificationRule(auth, notificationRuleID)
		defer closeDatadogResponseBody(httpResponse)
	}
	if err != nil {
		return datadogV2.NotificationRule{}, err
	}
	notificationRule := response.GetData()
	if notificationRule.GetId() == "" {
		if response.UnparsedObject != nil {
			if rawData, ok := response.UnparsedObject["data"]; ok {
				return securityNotificationRuleFromRawData(rawData)
			}
			return datadogV2.NotificationRule{}, fmt.Errorf("security notification rule raw response missing data")
		}
		return datadogV2.NotificationRule{}, fmt.Errorf("security notification rule %q not found", notificationRuleID)
	}

	return notificationRule, nil
}

func listSecurityNotificationRules(auth context.Context, api *datadogV2.SecurityMonitoringApi) ([]datadogV2.NotificationRule, error) {
	notificationRules := []datadogV2.NotificationRule{}

	signalRules, httpResponse, err := api.GetSignalNotificationRules(auth)
	closeDatadogResponseBody(httpResponse)
	if err != nil {
		return nil, err
	}
	rules, err := securityNotificationRulesFromRawData(signalRules)
	if err != nil {
		return nil, err
	}
	notificationRules = append(notificationRules, rules...)

	vulnerabilityRules, httpResponse, err := api.GetVulnerabilityNotificationRules(auth)
	closeDatadogResponseBody(httpResponse)
	if err != nil {
		return nil, err
	}
	rules, err = securityNotificationRulesFromRawData(vulnerabilityRules)
	if err != nil {
		return nil, err
	}
	notificationRules = append(notificationRules, rules...)

	return notificationRules, nil
}

func securityNotificationRulesFromRawData(rawData interface{}) ([]datadogV2.NotificationRule, error) {
	rawResponse, ok := rawData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("security notification rules raw response is not an object")
	}
	rawRules, ok := rawResponse["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("security notification rules raw response missing data list")
	}

	notificationRules := []datadogV2.NotificationRule{}
	for index, rawRule := range rawRules {
		notificationRule, err := securityNotificationRuleFromRawData(rawRule)
		if err != nil {
			return nil, fmt.Errorf("parse security notification rule raw data[%d]: %w", index, err)
		}
		notificationRules = append(notificationRules, notificationRule)
	}
	return notificationRules, nil
}

func securityNotificationRuleFromRawData(rawData interface{}) (datadogV2.NotificationRule, error) {
	rawRule, ok := rawData.(map[string]interface{})
	if !ok {
		return datadogV2.NotificationRule{}, fmt.Errorf("raw notification rule is not an object")
	}
	if rawType, ok := rawRule["type"].(string); ok && rawType != string(datadogV2.NOTIFICATIONRULESTYPE_NOTIFICATION_RULES) {
		return datadogV2.NotificationRule{}, fmt.Errorf("unexpected notification rule type %q", rawType)
	}
	rawID, ok := rawRule["id"].(string)
	if !ok || rawID == "" {
		return datadogV2.NotificationRule{}, fmt.Errorf("raw notification rule missing id")
	}

	notificationRule := datadogV2.NewNotificationRuleWithDefaults()
	notificationRule.SetId(rawID)
	notificationRule.SetType(datadogV2.NOTIFICATIONRULESTYPE_NOTIFICATION_RULES)
	return *notificationRule, nil
}
