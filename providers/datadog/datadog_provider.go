// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/zclconf/go-cty/cty"
)

type DatadogProvider struct { //nolint
	terraformutils.Provider
	apiKey        string
	appKey        string
	apiURL        string
	validate      bool
	auth          context.Context
	datadogClient *datadog.APIClient
}

// Init check env params and initialize API Client
func (p *DatadogProvider) Init(args []string) error {
	p.apiKey = ""
	p.appKey = ""
	p.apiURL = ""
	p.validate = false
	p.auth = nil
	p.datadogClient = nil

	apiKeyArg := optionalInitArg(args, 0)
	appKeyArg := optionalInitArg(args, 1)
	apiURLArg := optionalInitArg(args, 2)
	validateArg := optionalInitArg(args, 3)
	var apiKey, appKey, apiURL string
	validate := true

	switch {
	case validateArg != "":
		parsedValidate, validateErr := strconv.ParseBool(validateArg)
		if validateErr != nil {
			return fmt.Errorf(`invalid validate arg : %w`, validateErr)
		}
		validate = parsedValidate
	case os.Getenv("DATADOG_VALIDATE") != "":
		parsedValidate, validateErr := strconv.ParseBool(os.Getenv("DATADOG_VALIDATE"))
		if validateErr != nil {
			return fmt.Errorf(`invalid DATADOG_VALIDATE env var : %w`, validateErr)
		}
		validate = parsedValidate
	}

	if apiKeyArg != "" {
		apiKey = apiKeyArg
	} else {
		if envAPIKey := os.Getenv("DATADOG_API_KEY"); envAPIKey != "" {
			apiKey = envAPIKey
		} else if validate {
			return errors.New("api-key requirement")
		}
	}

	if appKeyArg != "" {
		appKey = appKeyArg
	} else {
		if envAppKey := os.Getenv("DATADOG_APP_KEY"); envAppKey != "" {
			appKey = envAppKey
		} else if validate {
			return errors.New("app-key requirement")
		}
	}

	if apiURLArg != "" {
		apiURL = apiURLArg
	} else if v := os.Getenv("DATADOG_HOST"); v != "" {
		apiURL = v
	}

	// Initialize the Datadog V1 API client
	auth := context.WithValue(
		context.Background(),
		datadog.ContextAPIKeys,
		map[string]datadog.APIKey{
			"apiKeyAuth": {
				Key: apiKey,
			},
			"appKeyAuth": {
				Key: appKey,
			},
		},
	)
	if apiURL != "" {
		parsedAPIURL, parseErr := url.Parse(apiURL)
		if parseErr != nil {
			return fmt.Errorf(`invalid API Url : %w`, parseErr)
		}
		if parsedAPIURL.Host == "" || parsedAPIURL.Scheme == "" {
			return fmt.Errorf(`missing protocol or host : %v`, apiURL)
		}
		// If api url is passed, set and use the api name and protocol on ServerIndex{1}
		auth = context.WithValue(auth, datadog.ContextServerIndex, 1)
		auth = context.WithValue(auth, datadog.ContextServerVariables, map[string]string{
			"name":     parsedAPIURL.Host,
			"protocol": parsedAPIURL.Scheme,
		})
	}
	configV1 := datadog.NewConfiguration()
	datadogClient := datadog.NewAPIClient(configV1)

	p.apiKey = apiKey
	p.appKey = appKey
	p.apiURL = apiURL
	p.validate = validate
	p.auth = auth
	p.datadogClient = datadogClient

	return nil
}

func optionalInitArg(args []string, index int) string {
	if len(args) > index {
		return args[index]
	}
	return ""
}

// GetName return string of provider name for Datadog
func (p *DatadogProvider) GetName() string {
	return "datadog"
}

// GetConfig return map of provider config for Datadog
func (p *DatadogProvider) GetConfig() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"api_key":  cty.StringVal(p.apiKey),
		"app_key":  cty.StringVal(p.appKey),
		"api_url":  cty.StringVal(p.apiURL),
		"validate": cty.BoolVal(p.validate),
	})
}

// InitService ...
func (p *DatadogProvider) InitService(serviceName string, verbose bool) error {
	if !terraformutils.SelectProviderService(&p.Provider, p.GetSupportedService(), serviceName, verbose, p.GetName()) {
		return errors.New(p.GetName() + ": " + serviceName + " not supported service")
	}
	p.Service.SetArgs(map[string]interface{}{
		"api-key":       p.apiKey,
		"app-key":       p.appKey,
		"api-url":       p.apiURL,
		"validate":      p.validate,
		"auth":          p.auth,
		"datadogClient": p.datadogClient,
	})
	return nil
}

// GetSupportedService return map of support service for Datadog
func (p *DatadogProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"agentless_scanning_aws_scan_options":      &AgentlessScanningAwsScanOptionsGenerator{},
		"agentless_scanning_azure_scan_options":    &AgentlessScanningAzureScanOptionsGenerator{},
		"agentless_scanning_gcp_scan_options":      &AgentlessScanningGcpScanOptionsGenerator{},
		"api_key":                                  &APIKeyGenerator{},
		"application_key":                          &ApplicationKeyGenerator{},
		"apm_retention_filter":                     &APMRetentionFilterGenerator{},
		"apm_retention_filter_order":               &APMRetentionFilterOrderGenerator{},
		"appsec_waf_custom_rule":                   &AppSecWafCustomRuleGenerator{},
		"appsec_waf_exclusion_filter":              &AppSecWafExclusionFilterGenerator{},
		"authn_mapping":                            &AuthnMappingGenerator{},
		"aws_cur_config":                           &AwsCURConfigGenerator{},
		"azure_uc_config":                          &AzureUCConfigGenerator{},
		"cloud_configuration_rule":                 &CloudConfigurationRuleGenerator{},
		"cloud_inventory_sync_config":              &CloudInventorySyncConfigGenerator{},
		"cost_budget":                              &CostBudgetGenerator{},
		"csm_threats_agent_rule":                   &CSMThreatsAgentRuleGenerator{},
		"csm_threats_policy":                       &CSMThreatsPolicyGenerator{},
		"custom_allocation_rule":                   &CustomAllocationRuleGenerator{},
		"dashboard_list":                           &DashboardListGenerator{},
		"dashboard":                                &DashboardGenerator{},
		"dashboard_json":                           &DashboardJSONGenerator{},
		"dashboard_v2":                             &DashboardV2Generator{},
		"domain_allowlist":                         &DomainAllowlistGenerator{},
		"downtime":                                 &DowntimeGenerator{},
		"downtime_schedule":                        &DowntimeScheduleGenerator{},
		"gcp_uc_config":                            &GCPUCConfigGenerator{},
		"incident_notification_rule":               &IncidentNotificationRuleGenerator{},
		"incident_notification_template":           &IncidentNotificationTemplateGenerator{},
		"incident_type":                            &IncidentTypeGenerator{},
		"ip_allowlist":                             &IPAllowlistGenerator{},
		"logs_archive":                             &LogsArchiveGenerator{},
		"logs_archive_order":                       &LogsArchiveOrderGenerator{},
		"logs_custom_pipeline":                     &LogsCustomPipelineGenerator{},
		"logs_index":                               &LogsIndexGenerator{},
		"logs_index_order":                         &LogsIndexOrderGenerator{},
		"logs_integration_pipeline":                &LogsIntegrationPipelineGenerator{},
		"logs_metric":                              &LogsMetricGenerator{},
		"logs_pipeline_order":                      &LogsPipelineOrderGenerator{},
		"logs_restriction_query":                   &LogsRestrictionQueryGenerator{},
		"integration_aws":                          &IntegrationAWSGenerator{},
		"integration_aws_lambda_arn":               &IntegrationAWSLambdaARNGenerator{},
		"integration_aws_log_collection":           &IntegrationAWSLogCollectionGenerator{},
		"integration_azure":                        &IntegrationAzureGenerator{},
		"integration_confluent_resource":           &IntegrationConfluentResourceGenerator{},
		"integration_fastly_service":               &IntegrationFastlyServiceGenerator{},
		"integration_gcp":                          &IntegrationGCPGenerator{},
		"integration_ms_teams_tenant_based_handle": &IntegrationMSTeamsTenantBasedHandleGenerator{},
		"integration_pagerduty":                    &IntegrationPagerdutyGenerator{},
		"integration_pagerduty_service_object":     &IntegrationPagerdutyServiceObjectGenerator{},
		"integration_slack_channel":                &IntegrationSlackChannelGenerator{},
		"metric_metadata":                          &MetricMetadataGenerator{},
		"metric_tag_configuration":                 &MetricTagConfigurationGenerator{},
		"monitor":                                  &MonitorGenerator{},
		"monitor_config_policy":                    &MonitorConfigPolicyGenerator{},
		"monitor_json":                             &MonitorJSONGenerator{},
		"monitor_notification_rule":                &MonitorNotificationRuleGenerator{},
		"observability_pipeline":                   &ObservabilityPipelineGenerator{},
		"on_call_escalation_policy":                &OnCallEscalationPolicyGenerator{},
		"on_call_schedule":                         &OnCallScheduleGenerator{},
		"on_call_team_routing_rules":               &OnCallTeamRoutingRulesGenerator{},
		"on_call_user_notification_channel":        &OnCallUserNotificationChannelGenerator{},
		"on_call_user_notification_rule":           &OnCallUserNotificationRuleGenerator{},
		"org_connection":                           &OrgConnectionGenerator{},
		"org_group":                                &OrgGroupGenerator{},
		"org_group_membership":                     &OrgGroupMembershipGenerator{},
		"org_group_policy":                         &OrgGroupPolicyGenerator{},
		"organization_settings":                    &OrganizationSettingsGenerator{},
		"powerpack":                                &PowerpackGenerator{},
		"powerpack_v2":                             &PowerpackV2Generator{},
		"rum_application":                          &RumApplicationGenerator{},
		"rum_metric":                               &RumMetricGenerator{},
		"rum_retention_filter":                     &RumRetentionFilterGenerator{},
		"rum_retention_filters_order":              &RumRetentionFiltersOrderGenerator{},
		"security_monitoring_default_rule":         &SecurityMonitoringDefaultRuleGenerator{},
		"security_monitoring_critical_asset":       &SecurityMonitoringCriticalAssetGenerator{},
		"security_monitoring_filter":               &SecurityMonitoringFilterGenerator{},
		"security_monitoring_rule":                 &SecurityMonitoringRuleGenerator{},
		"security_monitoring_suppression":          &SecurityMonitoringSuppressionGenerator{},
		"security_notification_rule":               &SecurityNotificationRuleGenerator{},
		"sensitive_data_scanner_group":             &SensitiveDataScannerGroupGenerator{},
		"sensitive_data_scanner_group_order":       &SensitiveDataScannerGroupOrderGenerator{},
		"sensitive_data_scanner_rule":              &SensitiveDataScannerRuleGenerator{},
		"service_account":                          &ServiceAccountGenerator{},
		"service_account_application_key":          &ServiceAccountApplicationKeyGenerator{},
		"service_definition_yaml":                  &ServiceDefinitionYAMLGenerator{},
		"service_level_objective":                  &ServiceLevelObjectiveGenerator{},
		"slo_correction":                           &SLOCorrectionGenerator{},
		"software_catalog":                         &SoftwareCatalogGenerator{},
		"spans_metric":                             &SpansMetricGenerator{},
		"synthetics_concurrency_cap":               &SyntheticsConcurrencyCapGenerator{},
		"synthetics_global_variable":               &SyntheticsGlobalVariableGenerator{},
		"synthetics_private_location":              &SyntheticsPrivateLocationGenerator{},
		"synthetics_suite":                         &SyntheticsSuiteGenerator{},
		"synthetics_test":                          &SyntheticsTestGenerator{},
		"tag_pipeline_ruleset":                     &TagPipelineRulesetGenerator{},
		"team":                                     &TeamGenerator{},
		"team_connection":                          &TeamConnectionGenerator{},
		"team_hierarchy_links":                     &TeamHierarchyLinksGenerator{},
		"team_link":                                &TeamLinkGenerator{},
		"team_membership":                          &TeamMembershipGenerator{},
		"team_notification_rule":                   &TeamNotificationRuleGenerator{},
		"team_permission_setting":                  &TeamPermissionSettingGenerator{},
		"team_sync":                                &TeamSyncGenerator{},
		"user":                                     &UserGenerator{},
		"webhook":                                  &WebhookGenerator{},
		"workflow_automation":                      &WorkflowAutomationGenerator{},
		"role":                                     &RoleGenerator{},
	}
}

// GetResourceConnections return map of resource connections for Datadog
func (p DatadogProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{
		"apm_retention_filter_order": {
			"apm_retention_filter": {
				"filter_ids", "id",
			},
		},
		"dashboard": {
			"monitor": {
				"widget.alert_graph_definition.alert_id", "id",
				"widget.group_definition.widget.alert_graph_definition.alert_id", "id",
				"widget.alert_value_definition.alert_id", "id",
				"widget.group_definition.widget.alert_value_definition.alert_id", "id",
			},
			"monitor_json": {
				"widget.alert_graph_definition.alert_id", "id",
				"widget.group_definition.widget.alert_graph_definition.alert_id", "id",
				"widget.alert_value_definition.alert_id", "id",
				"widget.group_definition.widget.alert_value_definition.alert_id", "id",
			},
			"service_level_objective": {
				"widget.service_level_objective_definition.slo_id", "id",
				"widget.group_definition.widget.service_level_objective_definition.slo_id", "id",
			},
		},
		"dashboard_list": {
			"dashboard": {
				"dash_item.dash_id", "id",
			},
		},
		"downtime": {
			"monitor": {
				"monitor_id", "id",
			},
			"monitor_json": {
				"monitor_id", "id",
			},
		},
		"downtime_schedule": {
			"monitor": {
				"monitor_identifier.monitor_id", "id",
			},
			"monitor_json": {
				"monitor_identifier.monitor_id", "id",
			},
		},
		"on_call_escalation_policy": {
			"on_call_schedule": {
				"step.target.schedule", "id",
			},
			"team": {
				"teams", "id",
				"step.target.team", "id",
			},
			"user": {
				"step.target.user", "id",
			},
		},
		"on_call_schedule": {
			"team": {
				"teams", "id",
			},
			"user": {
				"layer.users", "id",
			},
		},
		"on_call_team_routing_rules": {
			"on_call_escalation_policy": {
				"rule.escalation_policy", "id",
			},
			"team": {
				"id", "id",
			},
		},
		"on_call_user_notification_channel": {
			"user": {
				"user_id", "id",
			},
		},
		"on_call_user_notification_rule": {
			"on_call_user_notification_channel": {
				"channel_id", "id",
			},
			"user": {
				"user_id", "id",
			},
		},
		"rum_retention_filter": {
			"rum_application": {
				"application_id", "id",
			},
		},
		"rum_retention_filters_order": {
			"rum_application": {
				"application_id", "id",
			},
		},
		"team_connection": {
			"team": {
				"team.id", "id",
			},
		},
		"team_hierarchy_links": {
			"team": {
				"parent_team_id", "id",
				"sub_team_id", "id",
			},
		},
		"integration_aws_lambda_arn": {
			"integration_aws": {
				"account_id", "account_id",
			},
		},
		"integration_aws_log_collection": {
			"integration_aws": {
				"account_id", "account_id",
			},
		},
		"logs_archive": {
			"integration_aws": {
				"s3.account_id", "account_id",
				"s3.role_name", "role_name",
				"s3_archive.account_id", "account_id",
				"s3_archive.role_name", "role_name",
			},
			"integration_gcp": {
				"gcs.project_id", "project_id",
				"gcs.client_email", "client_email",
				"gcs_archive.project_id", "project_id",
				"gcs_archive.client_email", "client_email",
			},
			"integration_azure": {
				"azure.client_id", "client_id",
				"azure.tenant_id", "tenant_name",
				"azure_archive.client_id", "client_id",
				"azure_archive.tenant_id", "tenant_name",
			},
		},
		"logs_archive_order": {
			"logs_archive": {
				"archive_ids", "id",
			},
		},
		"logs_index_order": {
			"logs_index": {
				"indexes", "id",
			},
		},
		"logs_pipeline_order": {
			"logs_integration_pipeline": {
				"pipelines", "id",
			},
			"logs_custom_pipeline": {
				"pipelines", "id",
			},
		},
		"monitor": {
			"role": {
				"restricted_roles", "id",
			},
		},
		"service_level_objective": {
			"monitor": {
				"monitor_ids", "id",
			},
			"monitor_json": {
				"monitor_ids", "id",
			},
		},
		"sensitive_data_scanner_group_order": {
			"sensitive_data_scanner_group": {
				"group_ids", "id",
			},
		},
		"sensitive_data_scanner_rule": {
			"sensitive_data_scanner_group": {
				"group_id", "id",
			},
		},
		"synthetics_test": {
			"synthetics_private_location": {
				"locations", "id",
			},
		},
		"synthetics_suite": {
			"synthetics_test": {
				"tests.public_id", "id",
			},
		},
		"synthetics_global_variable": {
			"synthetics_test": {
				"parse_test_id", "id",
			},
		},
		"user": {
			"role": {
				"roles", "id",
			},
		},
	}
}

// GetProviderData return map of provider data for Datadog
func (p DatadogProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}
