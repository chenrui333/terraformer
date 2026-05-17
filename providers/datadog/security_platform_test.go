// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/zclconf/go-cty/cty"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestCloudConfigurationRuleCreateResourcesOnlyCloudConfiguration(t *testing.T) {
	cloudRule := securityMonitoringStandardRule("cloud-rule", datadogV2.SECURITYMONITORINGRULETYPEREAD_CLOUD_CONFIGURATION, false)
	defaultCloudRule := securityMonitoringStandardRule("default-cloud-rule", datadogV2.SECURITYMONITORINGRULETYPEREAD_CLOUD_CONFIGURATION, true)
	logRule := securityMonitoringStandardRule("log-rule", datadogV2.SECURITYMONITORINGRULETYPEREAD_LOG_DETECTION, false)

	generator := &CloudConfigurationRuleGenerator{}
	resources := generator.createResources([]datadogV2.SecurityMonitoringRuleResponse{
		datadogV2.SecurityMonitoringStandardRuleResponseAsSecurityMonitoringRuleResponse(cloudRule),
		datadogV2.SecurityMonitoringStandardRuleResponseAsSecurityMonitoringRuleResponse(defaultCloudRule),
		datadogV2.SecurityMonitoringStandardRuleResponseAsSecurityMonitoringRuleResponse(logRule),
	})

	if len(resources) != 1 {
		t.Fatalf("resource count = %d, want %d", len(resources), 1)
	}
	if resources[0].InstanceState.ID != "cloud-rule" {
		t.Fatalf("resource ID = %q, want %q", resources[0].InstanceState.ID, "cloud-rule")
	}
	if resources[0].ResourceName != "tfer--cloud_configuration_rule_cloud-rule" {
		t.Fatalf("resource name = %q, want %q", resources[0].ResourceName, "tfer--cloud_configuration_rule_cloud-rule")
	}
	if resources[0].InstanceInfo.Type != "datadog_cloud_configuration_rule" {
		t.Fatalf("resource type = %q, want %q", resources[0].InstanceInfo.Type, "datadog_cloud_configuration_rule")
	}
}

func TestSecurityMonitoringRuleCreateResourcesSkipsCloudConfiguration(t *testing.T) {
	cloudRule := securityMonitoringStandardRule("cloud-rule", datadogV2.SECURITYMONITORINGRULETYPEREAD_CLOUD_CONFIGURATION, false)
	logRule := securityMonitoringStandardRule("log-rule", datadogV2.SECURITYMONITORINGRULETYPEREAD_LOG_DETECTION, false)

	generator := &SecurityMonitoringRuleGenerator{}
	resources := generator.createResources([]datadogV2.SecurityMonitoringRuleResponse{
		datadogV2.SecurityMonitoringStandardRuleResponseAsSecurityMonitoringRuleResponse(cloudRule),
		datadogV2.SecurityMonitoringStandardRuleResponseAsSecurityMonitoringRuleResponse(logRule),
	})

	if len(resources) != 1 {
		t.Fatalf("resource count = %d, want %d", len(resources), 1)
	}
	if resources[0].InstanceState.ID != "log-rule" {
		t.Fatalf("resource ID = %q, want %q", resources[0].InstanceState.ID, "log-rule")
	}
}

func TestCloudConfigurationRuleAllowEmptyValuesPreservesFilterQuery(t *testing.T) {
	allowEmptyValues := allowEmptyValueRegexps(CloudConfigurationRuleAllowEmptyValues)
	parser := terraformutils.NewFlatmapParser(map[string]string{
		"filter.#":        "1",
		"filter.0.action": "require",
		"filter.0.query":  "",
	}, nil, allowEmptyValues)
	cloudConfigurationRuleType := cty.Object(map[string]cty.Type{
		"filter": cty.List(cty.Object(map[string]cty.Type{
			"action": cty.String,
			"query":  cty.String,
		})),
	})

	result, err := parser.Parse(cloudConfigurationRuleType)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	filters := result["filter"].([]interface{})
	filter := filters[0].(map[string]interface{})
	if filter["query"] != "" {
		t.Fatalf("filter query = %v, want empty string", filter["query"])
	}
}

func TestSecurityMonitoringCriticalAssetCreateResource(t *testing.T) {
	criticalAsset := datadogV2.NewSecurityMonitoringCriticalAssetWithDefaults()
	criticalAsset.SetId("critical-asset-id")

	generator := &SecurityMonitoringCriticalAssetGenerator{}
	resource, err := generator.createResource(*criticalAsset)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "critical-asset-id" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "critical-asset-id")
	}
	if resource.ResourceName != "tfer--security_monitoring_critical_asset_critical-asset-id" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--security_monitoring_critical_asset_critical-asset-id")
	}
	if resource.InstanceInfo.Type != "datadog_security_monitoring_critical_asset" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_security_monitoring_critical_asset")
	}
}

func TestSecurityMonitoringCriticalAssetCreateResourceMissingID(t *testing.T) {
	generator := &SecurityMonitoringCriticalAssetGenerator{}
	_, err := generator.createResource(datadogV2.SecurityMonitoringCriticalAsset{})
	if err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestSecurityMonitoringCriticalAssetAllowEmptyValuesPreservesQueries(t *testing.T) {
	allowEmptyValues := allowEmptyValueRegexps(SecurityMonitoringCriticalAssetAllowEmptyValues)
	parser := terraformutils.NewFlatmapParser(map[string]string{
		"query":      "",
		"rule_query": "",
	}, nil, allowEmptyValues)
	criticalAssetType := cty.Object(map[string]cty.Type{
		"query":      cty.String,
		"rule_query": cty.String,
	})

	result, err := parser.Parse(criticalAssetType)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if result["query"] != "" {
		t.Fatalf("query = %v, want empty string", result["query"])
	}
	if result["rule_query"] != "" {
		t.Fatalf("rule_query = %v, want empty string", result["rule_query"])
	}
}

func TestSecurityMonitoringCriticalAssetsFromRawData(t *testing.T) {
	criticalAssets := securityMonitoringCriticalAssetsFromRawData([]interface{}{
		map[string]interface{}{
			"id":   "critical-asset-1",
			"type": "critical_assets",
		},
		map[string]interface{}{
			"id":   "critical-asset-2",
			"type": "critical_assets",
		},
		map[string]interface{}{
			"id":   "ignored",
			"type": "other_type",
		},
		map[string]interface{}{
			"type": "critical_assets",
		},
	})

	if len(criticalAssets) != 2 {
		t.Fatalf("critical asset count = %d, want %d", len(criticalAssets), 2)
	}
	if criticalAssets[0].GetId() != "critical-asset-1" {
		t.Fatalf("first critical asset ID = %q, want %q", criticalAssets[0].GetId(), "critical-asset-1")
	}
	if criticalAssets[1].GetId() != "critical-asset-2" {
		t.Fatalf("second critical asset ID = %q, want %q", criticalAssets[1].GetId(), "critical-asset-2")
	}
}

func TestSecurityNotificationRuleCreateResource(t *testing.T) {
	notificationRule := datadogV2.NewNotificationRuleWithDefaults()
	notificationRule.SetId("notification-rule-id")

	generator := &SecurityNotificationRuleGenerator{}
	resource, err := generator.createResource(*notificationRule)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "notification-rule-id" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "notification-rule-id")
	}
	if resource.ResourceName != "tfer--security_notification_rule_notification-rule-id" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--security_notification_rule_notification-rule-id")
	}
	if resource.InstanceInfo.Type != "datadog_security_notification_rule" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_security_notification_rule")
	}
}

func TestSecurityNotificationRuleCreateResourceMissingID(t *testing.T) {
	generator := &SecurityNotificationRuleGenerator{}
	_, err := generator.createResource(datadogV2.NotificationRule{})
	if err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestSecurityNotificationRulesFromRawData(t *testing.T) {
	notificationRules := securityNotificationRulesFromRawData(map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{
				"id":   "signal-rule",
				"type": "notification_rules",
			},
			map[string]interface{}{
				"id":   "vulnerability-rule",
				"type": "notification_rules",
			},
			map[string]interface{}{
				"id":   "ignored",
				"type": "other_type",
			},
			map[string]interface{}{
				"type": "notification_rules",
			},
		},
	})

	if len(notificationRules) != 2 {
		t.Fatalf("notification rule count = %d, want %d", len(notificationRules), 2)
	}
	if notificationRules[0].GetId() != "signal-rule" {
		t.Fatalf("first notification rule ID = %q, want %q", notificationRules[0].GetId(), "signal-rule")
	}
	if notificationRules[1].GetId() != "vulnerability-rule" {
		t.Fatalf("second notification rule ID = %q, want %q", notificationRules[1].GetId(), "vulnerability-rule")
	}
}

func TestSecurityNotificationRuleInitResourcesListsSignalAndVulnerabilityRules(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v2/security/signals/notification_rules":
			fmt.Fprint(w, securityNotificationRuleListResponseJSON("signal-rule"))
		case "/api/v2/security/vulnerabilities/notification_rules":
			fmt.Fprint(w, securityNotificationRuleListResponseJSON("vulnerability-rule"))
		default:
			t.Fatalf("unexpected request path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	generator := newSecurityNotificationRuleTestGenerator(server, nil)
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}

	if len(generator.Resources) != 2 {
		t.Fatalf("resource count = %d, want %d", len(generator.Resources), 2)
	}
	if generator.Resources[0].InstanceState.ID != "signal-rule" {
		t.Fatalf("first resource ID = %q, want %q", generator.Resources[0].InstanceState.ID, "signal-rule")
	}
	if generator.Resources[1].InstanceState.ID != "vulnerability-rule" {
		t.Fatalf("second resource ID = %q, want %q", generator.Resources[1].InstanceState.ID, "vulnerability-rule")
	}
}

func TestSecurityMonitoringRulesHasNextPage(t *testing.T) {
	if !securityMonitoringRulesHasNextPage(datadogV2.SecurityMonitoringListRulesResponse{}, 0, 2, 2) {
		t.Fatal("expected next page when page item count equals page size and response has no metadata")
	}
	if securityMonitoringRulesHasNextPage(datadogV2.SecurityMonitoringListRulesResponse{}, 0, 2, 1) {
		t.Fatal("did not expect next page when page item count is less than page size and response has no metadata")
	}
}

func securityMonitoringStandardRule(ruleID string, ruleType datadogV2.SecurityMonitoringRuleTypeRead, isDefault bool) *datadogV2.SecurityMonitoringStandardRuleResponse {
	rule := datadogV2.NewSecurityMonitoringStandardRuleResponseWithDefaults()
	rule.SetId(ruleID)
	rule.SetType(ruleType)
	rule.SetIsDefault(isDefault)
	rule.SetIsEnabled(true)
	return rule
}

func allowEmptyValueRegexps(patterns []string) []*regexp.Regexp {
	allowEmptyValues := []*regexp.Regexp{}
	for _, pattern := range patterns {
		allowEmptyValues = append(allowEmptyValues, regexp.MustCompile(pattern))
	}
	return allowEmptyValues
}

func securityNotificationRuleListResponseJSON(ids ...string) string {
	rules := ""
	for index, id := range ids {
		if index > 0 {
			rules += ","
		}
		rules += fmt.Sprintf("{\"id\":%q,\"type\":\"notification_rules\"}", id)
	}
	return fmt.Sprintf("{\"data\":[%s]}", rules)
}

func newSecurityNotificationRuleTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *SecurityNotificationRuleGenerator {
	return &SecurityNotificationRuleGenerator{
		DatadogService: DatadogService{
			Service: terraformutils.Service{
				Args: map[string]interface{}{
					"auth":          context.Background(),
					"datadogClient": newTeamRelationshipTestClient(server),
				},
				Filter: filter,
			},
		},
	}
}
