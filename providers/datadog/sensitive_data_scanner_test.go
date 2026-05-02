// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"regexp"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/zclconf/go-cty/cty"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestSensitiveDataScannerGroupAllowEmptyValuesPreservesFilterQuery(t *testing.T) {
	allowEmptyValues := compileSensitiveDataScannerAllowEmptyValues(SensitiveDataScannerGroupAllowEmptyValues)

	parser := terraformutils.NewFlatmapParser(map[string]string{
		"filter.#":       "1",
		"filter.0.query": "",
	}, nil, allowEmptyValues)
	groupType := cty.Object(map[string]cty.Type{
		"filter": cty.List(cty.Object(map[string]cty.Type{
			"query": cty.String,
		})),
	})

	result, err := parser.Parse(groupType)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	filters, ok := result["filter"].([]interface{})
	if !ok {
		t.Fatalf("filter = %T, want []interface{}", result["filter"])
	}
	if len(filters) != 1 {
		t.Fatalf("filter length = %d, want %d", len(filters), 1)
	}
	filter, ok := filters[0].(map[string]interface{})
	if !ok {
		t.Fatalf("filter[0] = %T, want map[string]interface{}", filters[0])
	}
	if filter["query"] != "" {
		t.Fatalf("filter[0].query = %v, want empty string", filter["query"])
	}
}

func TestSensitiveDataScannerGroupCreateResource(t *testing.T) {
	group := datadogV2.NewSensitiveDataScannerGroupIncludedItemWithDefaults()
	group.SetId("group-id")

	generator := &SensitiveDataScannerGroupGenerator{}
	resource, err := generator.createResource(*group)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "group-id" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "group-id")
	}
	if resource.ResourceName != "tfer--sensitive_data_scanner_group_group-id" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--sensitive_data_scanner_group_group-id")
	}
	if resource.InstanceInfo.Type != "datadog_sensitive_data_scanner_group" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_sensitive_data_scanner_group")
	}
}

func TestSensitiveDataScannerGroupCreateResourceMissingID(t *testing.T) {
	generator := &SensitiveDataScannerGroupGenerator{}
	_, err := generator.createResource(datadogV2.SensitiveDataScannerGroupIncludedItem{})
	if err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestSensitiveDataScannerGroupPostConvertHookPreservesEmptyProductList(t *testing.T) {
	resource := terraformutils.NewSimpleResource(
		"group-id",
		"sensitive_data_scanner_group_group-id",
		"datadog_sensitive_data_scanner_group",
		"datadog",
		SensitiveDataScannerGroupAllowEmptyValues,
	)
	resource.InstanceState.Attributes = map[string]string{
		"product_list.#": "0",
	}
	resource.Item = map[string]interface{}{}

	generator := &SensitiveDataScannerGroupGenerator{}
	generator.Resources = []terraformutils.Resource{resource}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}

	productList, ok := generator.Resources[0].Item["product_list"].([]interface{})
	if !ok {
		t.Fatalf("product_list = %T, want []interface{}", generator.Resources[0].Item["product_list"])
	}
	if len(productList) != 0 {
		t.Fatalf("product_list length = %d, want %d", len(productList), 0)
	}
}

func TestSensitiveDataScannerRuleCreateResourceSeedsRelationships(t *testing.T) {
	rule := newSensitiveDataScannerRule("rule-id", "group-id", "standard-pattern-id")

	generator := &SensitiveDataScannerRuleGenerator{}
	resource, err := generator.createResource(rule)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "rule-id" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "rule-id")
	}
	if resource.ResourceName != "tfer--sensitive_data_scanner_rule_rule-id" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--sensitive_data_scanner_rule_rule-id")
	}
	if resource.InstanceInfo.Type != "datadog_sensitive_data_scanner_rule" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_sensitive_data_scanner_rule")
	}
	if resource.InstanceState.Attributes["group_id"] != "group-id" {
		t.Fatalf("group_id = %q, want %q", resource.InstanceState.Attributes["group_id"], "group-id")
	}
	if resource.InstanceState.Attributes["standard_pattern_id"] != "standard-pattern-id" {
		t.Fatalf("standard_pattern_id = %q, want %q", resource.InstanceState.Attributes["standard_pattern_id"], "standard-pattern-id")
	}
}

func TestSensitiveDataScannerRulePostConvertHookPreservesEmptyKeywords(t *testing.T) {
	resource := terraformutils.NewSimpleResource(
		"rule-id",
		"sensitive_data_scanner_rule_rule-id",
		"datadog_sensitive_data_scanner_rule",
		"datadog",
		SensitiveDataScannerRuleAllowEmptyValues,
	)
	resource.InstanceState.Attributes = map[string]string{
		"included_keyword_configuration.#":            "1",
		"included_keyword_configuration.0.keywords.#": "0",
	}
	resource.Item = map[string]interface{}{
		"included_keyword_configuration": []interface{}{
			map[string]interface{}{
				"character_count": 30,
			},
		},
	}

	generator := &SensitiveDataScannerRuleGenerator{}
	generator.Resources = []terraformutils.Resource{resource}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}

	configs, ok := generator.Resources[0].Item["included_keyword_configuration"].([]interface{})
	if !ok {
		t.Fatalf("included_keyword_configuration = %T, want []interface{}", generator.Resources[0].Item["included_keyword_configuration"])
	}
	if len(configs) != 1 {
		t.Fatalf("included_keyword_configuration length = %d, want %d", len(configs), 1)
	}
	config, ok := configs[0].(map[string]interface{})
	if !ok {
		t.Fatalf("included_keyword_configuration[0] = %T, want map[string]interface{}", configs[0])
	}
	keywords, ok := config["keywords"].([]interface{})
	if !ok {
		t.Fatalf("keywords = %T, want []interface{}", config["keywords"])
	}
	if len(keywords) != 0 {
		t.Fatalf("keywords length = %d, want %d", len(keywords), 0)
	}
}

func TestSensitiveDataScannerRuleCreateResourceMissingGroupID(t *testing.T) {
	rule := datadogV2.NewSensitiveDataScannerRuleIncludedItemWithDefaults()
	rule.SetId("rule-id")

	generator := &SensitiveDataScannerRuleGenerator{}
	_, err := generator.createResource(*rule)
	if err == nil {
		t.Fatal("createResource returned nil error, want missing group relationship error")
	}
}

func TestSensitiveDataScannerGroupOrderCreateResource(t *testing.T) {
	config := newSensitiveDataScannerConfig("config-id", []string{"group-1", "group-2"})

	generator := &SensitiveDataScannerGroupOrderGenerator{}
	resource := generator.createResource(config)

	if resource.InstanceState.ID != "config-id" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "config-id")
	}
	if resource.ResourceName != "tfer--sensitive_data_scanner_group_order" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--sensitive_data_scanner_group_order")
	}
	if resource.InstanceInfo.Type != "datadog_sensitive_data_scanner_group_order" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_sensitive_data_scanner_group_order")
	}
	if resource.InstanceState.Attributes["group_ids.#"] != "2" {
		t.Fatalf("group_ids.# = %q, want %q", resource.InstanceState.Attributes["group_ids.#"], "2")
	}
	if resource.InstanceState.Attributes["group_ids.0"] != "group-1" {
		t.Fatalf("group_ids.0 = %q, want %q", resource.InstanceState.Attributes["group_ids.0"], "group-1")
	}
	if resource.InstanceState.Attributes["group_ids.1"] != "group-2" {
		t.Fatalf("group_ids.1 = %q, want %q", resource.InstanceState.Attributes["group_ids.1"], "group-2")
	}
}

func TestSensitiveDataScannerGroupOrderPostConvertHookPreservesEmptyGroupIDs(t *testing.T) {
	resource := terraformutils.NewResource(
		"order",
		"sensitive_data_scanner_group_order",
		"datadog_sensitive_data_scanner_group_order",
		"datadog",
		map[string]string{
			"group_ids.#": "0",
		},
		SensitiveDataScannerGroupOrderAllowEmptyValues,
		map[string]interface{}{},
	)
	resource.Item = map[string]interface{}{}

	generator := &SensitiveDataScannerGroupOrderGenerator{}
	generator.Resources = []terraformutils.Resource{resource}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}

	groupIDs, ok := generator.Resources[0].Item["group_ids"].([]interface{})
	if !ok {
		t.Fatalf("group_ids = %T, want []interface{}", generator.Resources[0].Item["group_ids"])
	}
	if len(groupIDs) != 0 {
		t.Fatalf("group_ids length = %d, want %d", len(groupIDs), 0)
	}
}

func TestSensitiveDataScannerConfigExtraction(t *testing.T) {
	group := datadogV2.NewSensitiveDataScannerGroupIncludedItemWithDefaults()
	group.SetId("group-id")
	rule := newSensitiveDataScannerRule("rule-id", "group-id", "")
	config := newSensitiveDataScannerConfig("config-id", []string{"group-id"})
	config.SetIncluded([]datadogV2.SensitiveDataScannerGetConfigIncludedItem{
		datadogV2.SensitiveDataScannerGroupIncludedItemAsSensitiveDataScannerGetConfigIncludedItem(group),
		datadogV2.SensitiveDataScannerRuleIncludedItemAsSensitiveDataScannerGetConfigIncludedItem(&rule),
	})

	groups := sensitiveDataScannerGroupsFromConfig(config)
	if len(groups) != 1 {
		t.Fatalf("group count = %d, want %d", len(groups), 1)
	}
	if groups[0].GetId() != "group-id" {
		t.Fatalf("group ID = %q, want %q", groups[0].GetId(), "group-id")
	}

	rules := sensitiveDataScannerRulesFromConfig(config)
	if len(rules) != 1 {
		t.Fatalf("rule count = %d, want %d", len(rules), 1)
	}
	if rules[0].GetId() != "rule-id" {
		t.Fatalf("rule ID = %q, want %q", rules[0].GetId(), "rule-id")
	}
}

func compileSensitiveDataScannerAllowEmptyValues(patterns []string) []*regexp.Regexp {
	allowEmptyValues := []*regexp.Regexp{}
	for _, pattern := range patterns {
		allowEmptyValues = append(allowEmptyValues, regexp.MustCompile(pattern))
	}
	return allowEmptyValues
}

func newSensitiveDataScannerConfig(configID string, groupIDs []string) datadogV2.SensitiveDataScannerGetConfigResponse {
	config := datadogV2.NewSensitiveDataScannerGetConfigResponseWithDefaults()
	data := datadogV2.NewSensitiveDataScannerGetConfigResponseDataWithDefaults()
	data.SetId(configID)

	items := []datadogV2.SensitiveDataScannerGroupItem{}
	for _, groupID := range groupIDs {
		item := datadogV2.NewSensitiveDataScannerGroupItemWithDefaults()
		item.SetId(groupID)
		items = append(items, *item)
	}
	groups := datadogV2.NewSensitiveDataScannerGroupListWithDefaults()
	groups.SetData(items)
	relationships := datadogV2.NewSensitiveDataScannerConfigurationRelationshipsWithDefaults()
	relationships.SetGroups(*groups)
	data.SetRelationships(*relationships)
	config.SetData(*data)

	return *config
}

func newSensitiveDataScannerRule(ruleID, groupID, standardPatternID string) datadogV2.SensitiveDataScannerRuleIncludedItem {
	rule := datadogV2.NewSensitiveDataScannerRuleIncludedItemWithDefaults()
	rule.SetId(ruleID)

	relationships := datadogV2.NewSensitiveDataScannerRuleRelationshipsWithDefaults()
	groupData := datadogV2.NewSensitiveDataScannerGroupDataWithDefaults()
	group := datadogV2.NewSensitiveDataScannerGroupWithDefaults()
	group.SetId(groupID)
	groupData.SetData(*group)
	relationships.SetGroup(*groupData)

	if standardPatternID != "" {
		standardPatternData := datadogV2.NewSensitiveDataScannerStandardPatternDataWithDefaults()
		standardPattern := datadogV2.NewSensitiveDataScannerStandardPatternWithDefaults()
		standardPattern.SetId(standardPatternID)
		standardPatternData.SetData(*standardPattern)
		relationships.SetStandardPattern(*standardPatternData)
	}

	rule.SetRelationships(*relationships)
	return *rule
}
