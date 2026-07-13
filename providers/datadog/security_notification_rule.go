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

type securityNotificationRuleAPI interface {
	GetSignalNotificationRule(context.Context, string) (datadogV2.NotificationRuleResponse, *http.Response, error)
	GetSignalNotificationRules(context.Context) (datadogV2.NotificationRulesListResponse, *http.Response, error)
	GetVulnerabilityNotificationRule(context.Context, string) (datadogV2.NotificationRuleResponse, *http.Response, error)
	GetVulnerabilityNotificationRules(context.Context) (datadogV2.NotificationRulesListResponse, *http.Response, error)
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
	g.Resources = nil
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

func (g *SecurityNotificationRuleGenerator) filteredResources(auth context.Context, api securityNotificationRuleAPI) ([]terraformutils.Resource, bool, error) {
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

func getSecurityNotificationRule(auth context.Context, api securityNotificationRuleAPI, notificationRuleID string) (datadogV2.NotificationRule, error) {
	response, httpResponse, err := api.GetSignalNotificationRule(auth, notificationRuleID)
	statusCode := responseStatusCode(httpResponse)
	closeDatadogResponseBody(httpResponse)
	if statusCode != http.StatusNotFound {
		if err != nil {
			return datadogV2.NotificationRule{}, fmt.Errorf("get signal notification rule %q: %w", notificationRuleID, err)
		}
		notificationRule, parseErr := securityNotificationRuleFromResponse(response)
		if parseErr != nil {
			return datadogV2.NotificationRule{}, fmt.Errorf("parse signal notification rule %q response: %w", notificationRuleID, parseErr)
		}
		return notificationRule, nil
	}

	response, httpResponse, err = api.GetVulnerabilityNotificationRule(auth, notificationRuleID)
	closeDatadogResponseBody(httpResponse)
	if err != nil {
		return datadogV2.NotificationRule{}, fmt.Errorf(
			"get vulnerability notification rule %q after signal lookup returned 404: %w",
			notificationRuleID,
			err,
		)
	}
	notificationRule, parseErr := securityNotificationRuleFromResponse(response)
	if parseErr != nil {
		return datadogV2.NotificationRule{}, fmt.Errorf("parse vulnerability notification rule %q response: %w", notificationRuleID, parseErr)
	}

	return notificationRule, nil
}

func responseStatusCode(response *http.Response) int {
	if response == nil {
		return 0
	}
	return response.StatusCode
}

func securityNotificationRuleFromResponse(response datadogV2.NotificationRuleResponse) (datadogV2.NotificationRule, error) {
	if response.UnparsedObject != nil {
		rawData, ok := response.UnparsedObject["data"]
		if !ok {
			return datadogV2.NotificationRule{}, fmt.Errorf("raw response missing data")
		}
		return securityNotificationRuleFromRawData(rawData)
	}

	notificationRule, ok := response.GetDataOk()
	if !ok {
		return datadogV2.NotificationRule{}, fmt.Errorf("decoded response missing data")
	}
	if err := validateSecurityNotificationRule(*notificationRule); err != nil {
		return datadogV2.NotificationRule{}, err
	}
	return *notificationRule, nil
}

func listSecurityNotificationRules(auth context.Context, api securityNotificationRuleAPI) ([]datadogV2.NotificationRule, error) {
	notificationRules := []datadogV2.NotificationRule{}

	signalRules, httpResponse, err := api.GetSignalNotificationRules(auth)
	closeDatadogResponseBody(httpResponse)
	if err != nil {
		return nil, fmt.Errorf("list signal notification rules: %w", err)
	}
	rules, err := securityNotificationRulesFromRawData(signalRules)
	if err != nil {
		return nil, fmt.Errorf("parse signal notification rules response: %w", err)
	}
	notificationRules = append(notificationRules, rules...)

	vulnerabilityRules, httpResponse, err := api.GetVulnerabilityNotificationRules(auth)
	closeDatadogResponseBody(httpResponse)
	if err != nil {
		return nil, fmt.Errorf("list vulnerability notification rules: %w", err)
	}
	rules, err = securityNotificationRulesFromRawData(vulnerabilityRules)
	if err != nil {
		return nil, fmt.Errorf("parse vulnerability notification rules response: %w", err)
	}
	notificationRules = append(notificationRules, rules...)

	return notificationRules, nil
}

func securityNotificationRulesFromRawData(rawData interface{}) ([]datadogV2.NotificationRule, error) {
	if response, ok := rawData.(datadogV2.NotificationRulesListResponse); ok {
		if response.UnparsedObject == nil {
			return response.GetData(), nil
		}
		rawData = response.UnparsedObject
	}

	rawResponse, ok := rawData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("security notification rules raw response is not an object")
	}
	rawRuleData, ok := rawResponse["data"]
	if !ok {
		return nil, fmt.Errorf("security notification rules raw response missing data")
	}
	if rawRuleData == nil {
		return nil, fmt.Errorf("security notification rules raw response data is null")
	}
	rawRules, ok := rawRuleData.([]interface{})
	if !ok {
		return nil, fmt.Errorf("security notification rules raw response data is not a list")
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
	rawTypeValue, ok := rawRule["type"]
	if !ok {
		return datadogV2.NotificationRule{}, fmt.Errorf("raw notification rule missing type")
	}
	rawType, ok := rawTypeValue.(string)
	if !ok || rawType == "" {
		return datadogV2.NotificationRule{}, fmt.Errorf("raw notification rule type is not a non-empty string")
	}
	if rawType != string(datadogV2.NOTIFICATIONRULESTYPE_NOTIFICATION_RULES) {
		return datadogV2.NotificationRule{}, fmt.Errorf("unexpected notification rule type %q", rawType)
	}
	rawIDValue, ok := rawRule["id"]
	if !ok {
		return datadogV2.NotificationRule{}, fmt.Errorf("raw notification rule missing id")
	}
	rawID, ok := rawIDValue.(string)
	if !ok || rawID == "" {
		return datadogV2.NotificationRule{}, fmt.Errorf("raw notification rule id is not a non-empty string")
	}

	notificationRule := datadogV2.NewNotificationRuleWithDefaults()
	notificationRule.SetId(rawID)
	notificationRule.SetType(datadogV2.NOTIFICATIONRULESTYPE_NOTIFICATION_RULES)
	return *notificationRule, nil
}

func validateSecurityNotificationRule(notificationRule datadogV2.NotificationRule) error {
	if notificationRule.GetId() == "" {
		return fmt.Errorf("decoded notification rule missing id")
	}
	if notificationRule.GetType() == "" {
		return fmt.Errorf("decoded notification rule missing type")
	}
	if notificationRule.GetType() != datadogV2.NOTIFICATIONRULESTYPE_NOTIFICATION_RULES {
		return fmt.Errorf("unexpected notification rule type %q", notificationRule.GetType())
	}
	return nil
}
