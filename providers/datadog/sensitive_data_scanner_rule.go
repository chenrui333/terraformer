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
	// SensitiveDataScannerRuleAllowEmptyValues ...
	SensitiveDataScannerRuleAllowEmptyValues = []string{"included_keyword_configuration.*.keywords"}
)

// SensitiveDataScannerRuleGenerator ...
type SensitiveDataScannerRuleGenerator struct {
	DatadogService
}

func (g *SensitiveDataScannerRuleGenerator) createResources(rules []datadogV2.SensitiveDataScannerRuleIncludedItem) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, rule := range rules {
		resource, err := g.createResource(rule)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *SensitiveDataScannerRuleGenerator) createResource(rule datadogV2.SensitiveDataScannerRuleIncludedItem) (terraformutils.Resource, error) {
	ruleID := rule.GetId()
	if ruleID == "" {
		return terraformutils.Resource{}, fmt.Errorf("sensitive data scanner rule missing id")
	}
	groupID := sensitiveDataScannerRuleGroupID(rule)
	if groupID == "" {
		return terraformutils.Resource{}, fmt.Errorf("sensitive data scanner rule %q missing group relationship id", ruleID)
	}

	attributes := map[string]string{
		"group_id": groupID,
	}
	if standardPatternID := sensitiveDataScannerRuleStandardPatternID(rule); standardPatternID != "" {
		attributes["standard_pattern_id"] = standardPatternID
	}

	return terraformutils.NewResource(
		ruleID,
		fmt.Sprintf("sensitive_data_scanner_rule_%s", ruleID),
		"datadog_sensitive_data_scanner_rule",
		"datadog",
		attributes,
		SensitiveDataScannerRuleAllowEmptyValues,
		map[string]interface{}{},
	), nil
}

func (g *SensitiveDataScannerRuleGenerator) PostConvertHook() error {
	for i := range g.Resources {
		resource := &g.Resources[i]
		if resource.Item == nil {
			resource.Item = map[string]interface{}{}
		}
		preserveSensitiveDataScannerRuleEmptyKeywords(resource)
	}
	return nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each sensitive_data_scanner_rule create 1 TerraformResource.
func (g *SensitiveDataScannerRuleGenerator) InitResources() error {
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
	resources, err = g.createResources(sensitiveDataScannerRulesFromConfig(config))
	if err != nil {
		return err
	}

	g.Resources = resources
	return nil
}

func (g *SensitiveDataScannerRuleGenerator) filteredResources(auth context.Context, api *datadogV2.SensitiveDataScannerApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false
	var config *datadogV2.SensitiveDataScannerGetConfigResponse

	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("sensitive_data_scanner_rule") {
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
			rule, ok := sensitiveDataScannerRuleByID(*config, value)
			if !ok {
				return nil, true, fmt.Errorf("sensitive data scanner rule %q not found", value)
			}
			resource, err := g.createResource(rule)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, filtered, nil
}

func sensitiveDataScannerRulesFromConfig(config datadogV2.SensitiveDataScannerGetConfigResponse) []datadogV2.SensitiveDataScannerRuleIncludedItem {
	rules := []datadogV2.SensitiveDataScannerRuleIncludedItem{}
	for _, included := range config.GetIncluded() {
		if included.SensitiveDataScannerRuleIncludedItem == nil {
			continue
		}
		rules = append(rules, *included.SensitiveDataScannerRuleIncludedItem)
	}
	return rules
}

func sensitiveDataScannerRuleByID(config datadogV2.SensitiveDataScannerGetConfigResponse, ruleID string) (datadogV2.SensitiveDataScannerRuleIncludedItem, bool) {
	for _, rule := range sensitiveDataScannerRulesFromConfig(config) {
		if rule.GetId() == ruleID {
			return rule, true
		}
	}
	return datadogV2.SensitiveDataScannerRuleIncludedItem{}, false
}

func sensitiveDataScannerRuleGroupID(rule datadogV2.SensitiveDataScannerRuleIncludedItem) string {
	relationships, ok := rule.GetRelationshipsOk()
	if !ok {
		return ""
	}
	group, ok := relationships.GetGroupOk()
	if !ok {
		return ""
	}
	groupData, ok := group.GetDataOk()
	if !ok {
		return ""
	}
	return groupData.GetId()
}

func sensitiveDataScannerRuleStandardPatternID(rule datadogV2.SensitiveDataScannerRuleIncludedItem) string {
	relationships, ok := rule.GetRelationshipsOk()
	if !ok {
		return ""
	}
	standardPattern, ok := relationships.GetStandardPatternOk()
	if !ok {
		return ""
	}
	standardPatternData, ok := standardPattern.GetDataOk()
	if !ok {
		return ""
	}
	return standardPatternData.GetId()
}

func preserveSensitiveDataScannerRuleEmptyKeywords(resource *terraformutils.Resource) {
	if resource == nil || resource.InstanceState == nil || resource.InstanceState.Attributes == nil {
		return
	}
	count, err := strconv.Atoi(resource.InstanceState.Attributes["included_keyword_configuration.#"])
	if err != nil || count == 0 {
		return
	}

	configs, _ := resource.Item["included_keyword_configuration"].([]interface{})
	for len(configs) < count {
		configs = append(configs, map[string]interface{}{})
	}

	changed := false
	for index := 0; index < count; index++ {
		keywordsKey := fmt.Sprintf("included_keyword_configuration.%d.keywords.#", index)
		if resource.InstanceState.Attributes[keywordsKey] != "0" {
			continue
		}
		config, ok := configs[index].(map[string]interface{})
		if !ok {
			config = map[string]interface{}{}
			configs[index] = config
		}
		if _, ok := config["keywords"]; ok {
			continue
		}
		config["keywords"] = []interface{}{}
		changed = true
	}
	if changed {
		resource.Item["included_keyword_configuration"] = configs
	}
}
