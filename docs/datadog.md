# Use Terraformer with [Datadog](https://www.datadoghq.com/)

This provider uses the [terraform-provider-datadog](https://registry.terraform.io/providers/DataDog/datadog/latest).

##  Usage
### 1. Installation
First you will need to install Terraformer with the Datadog provider. See the [README](https://github.com/chenrui333/terraformer#installation).

### 2. Set up a template Terraform workspace
Before you can use Terraformer, you need to create a template workspace so that Terraformer
can access the [DataDog/datadog](https://registry.terraform.io/providers/DataDog/datadog/latest) provider.

To do this, create a new directory with a basic `provider.tf` file:
```hcl
terraform {
  required_providers {
    datadog = {
      source  = "DataDog/datadog"
      version = ">= 4.9.0"
    }
  }
}

provider "datadog" {
  # Configuration options
}
```

then run:
```bash
$ terraform init
````

You should see the output: `Terraform has been successfully initialized!`

### 3. Run Terraformer

```bash
export DATADOG_API_KEY=Datadog API key. More information on this at https://docs.datadoghq.com/account_management/api-app-keys/ 
export DATADOG_HOST=Datadog API host i.e. https://api.datadoghq.eu which can be found at https://docs.datadoghq.com/getting_started/site/#access-the-datadog-site
export DATADOG_APP_KEY=Datadog APP key. More information on this at https://docs.datadoghq.com/account_management/api-app-keys/ 

./terraformer import datadog --resources=* 
```

You can also specify only certain kinds of resources to import as well, i.e. `--resources=dashboard`.

### 4. Inspect the imported Terraform files

You should now see a `generated/` subdirectory with generated files.

You can now initialize and use your new generated resources:
```bash
$ terraform init
$ terraform plan # No changes. Your infrastructure matches the configuration.
```

### Filtering Resources

You can use the `filter` argument to restrict the import of Terraform resources.

Filtering based on Tags follows the convention `--filter="Name=tags;Value='your tag'"`.

```bash
# Import monitors based on multiple tags
./terraformer import datadog --resources=monitor --filter="Name=tags;Value='foo:bar'" --filter="Name=tags;Value='env:production'"

# Import monitor where tag doesn't include colon
./terraformer import datadog --resources=monitor --filter="Name=tags;Value=anExampleTag"
```

Filtering based on resource ID:

```bash
# Import dashboard based on the dashboard ID
./terraformer import datadog --resources=dashboard --filter=dashboard=some-id

# Import dashboard_v2 based on the dashboard ID
./terraformer import datadog --resources=dashboard_v2 --filter=dashboard_v2=some-id

# Import based on multiple resource IDs
 ./terraformer import datadog --resources=monitor --filter=monitor=id1:id2:id4
```

Tag filters are order specific. For example, if your monitor has tags (in the order) `atag: atagvalue`, `foo:bar` but you filter for `--filter="Name=tags;Value='foo:bar'" --filter="Name=tags;Value='atag: atagvalue'"`, the monitor would not be imported.

## Supported Datadog resources

*   `agentless_scanning_aws_scan_options`
    * `datadog_agentless_scanning_aws_scan_options`
*   `agentless_scanning_azure_scan_options`
    * `datadog_agentless_scanning_azure_scan_options`
        * **_NOTE:_** Requires DataDog/datadog provider 4.9.0 or newer.
*   `agentless_scanning_gcp_scan_options`
    * `datadog_agentless_scanning_gcp_scan_options`
*   `api_key`
    * `datadog_api_key`
*   `application_key`
    * `datadog_application_key`
*   `apm_retention_filter`
    * `datadog_apm_retention_filter`
*   `apm_retention_filter_order`
    * `datadog_apm_retention_filter_order`
        * **_NOTE:_** Importing a single retention filter order by ID accepts any value because the Datadog provider stores it as `filtersOrderID`, for example `--filter=apm_retention_filter_order=any-value`
*   `appsec_waf_custom_rule`
    * `datadog_appsec_waf_custom_rule`
*   `appsec_waf_exclusion_filter`
    * `datadog_appsec_waf_exclusion_filter`
*   `authn_mapping`
    * `datadog_authn_mapping`
*   `aws_cur_config`
    * `datadog_aws_cur_config`
        * **_NOTE:_** Requires DataDog/datadog provider 3.39.0 or newer.
*   `azure_uc_config`
    * `datadog_azure_uc_config`
        * **_NOTE:_** Requires DataDog/datadog provider 3.39.0 or newer.
*   `dashboard`
    * `datadog_dashboard`
*   `dashboard_json`
    * `datadog_dashboard_json`
*   `dashboard_list`
    * `datadog_dashboard_list`
*   `dashboard_v2`
    * `datadog_dashboard_v2`
        * **_NOTE:_** Requires DataDog/datadog provider 4.9.0 or newer.
        * **_NOTE:_** Discovers the same dashboard IDs as `dashboard` and `dashboard_json`; select one dashboard resource representation for each imported dashboard to avoid duplicate Terraform ownership.
*   `cloud_inventory_sync_config`
    * `datadog_cloud_inventory_sync_config`
        * **_NOTE:_** Requires DataDog/datadog provider 3.86.0 or newer.
        * **_NOTE:_** Importing resource requires resource ID's to be passed via [Filter][1] option
*   `cost_budget`
    * `datadog_cost_budget`
        * **_NOTE:_** Requires DataDog/datadog provider 3.39.0 or newer.
*   `csm_threats_agent_rule`
    * `datadog_csm_threats_agent_rule`
        * **_NOTE:_** For policy-scoped rules, filter IDs use `policy_id:rule_id` format, for example `--filter="csm_threats_agent_rule='policy-abc:rule-123'"`; unscoped rules accept bare rule IDs
*   `csm_threats_policy`
    * `datadog_csm_threats_policy`
*   `custom_allocation_rule`
    * `datadog_custom_allocation_rule`
        * **_NOTE:_** Requires DataDog/datadog provider 3.39.0 or newer.
*   `domain_allowlist`
    * `datadog_domain_allowlist`
        * **_NOTE:_** Singleton resource. Only one domain allowlist per org.
*   `downtime`
    * `datadog_downtime`
*   `gcp_uc_config`
    * `datadog_gcp_uc_config`
        * **_NOTE:_** Requires DataDog/datadog provider 3.39.0 or newer.
*   `integration_aws`
    * `datadog_integration_aws`
*   `integration_aws_lambda_arn`
    * `datadog_integration_aws_lambda_arn`
*   `integration_aws_log_collection`
    * `datadog_integration_aws_log_collection`
*   `integration_azure`
    * `datadog_integration_azure`
        * **_NOTE:_** Sensitive field `client_secret` is not generated and needs to be manually set
*   `integration_confluent_resource`
    * `datadog_integration_confluent_resource`
        * **_NOTE:_** Import ID is composite `account_id:resource_id`. Discovery lists resources across all Confluent accounts.
*   `integration_fastly_service`
    * `datadog_integration_fastly_service`
        * **_NOTE:_** Import ID is composite `account_id:service_id`. Discovery lists services across all Fastly accounts.
*   `integration_gcp`
    * `datadog_integration_gcp`
        * **_NOTE:_** Sensitive fields `private_key, private_key_id, client_id` is not generated and needs to be manually set
*   `integration_ms_teams_tenant_based_handle`
    * `datadog_integration_ms_teams_tenant_based_handle`
*   `integration_pagerduty`
    * `datadog_integration_pagerduty`
*   `integration_pagerduty_service_object`
    * `datadog_integration_pagerduty_service_object`
*   `integration_slack_channel`
    * `datadog_integration_slack_channel`
        * **_NOTE:_** Importing resource requires resource ID or `account_name` to be passed via [Filter][1] option
*   `ip_allowlist`
    * `datadog_ip_allowlist`
        * **_NOTE:_** Singleton resource. Only one IP allowlist per org.
*   `logs_archive`
    * `datadog_logs_archive`
*   `logs_archive_order`
    * `datadog_logs_archive_order`
*   `logs_custom_pipeline`
    * `datadog_logs_custom_pipeline`
*   `logs_index`
    * `datadog_logs_index`
*   `logs_index_order`
    * `datadog_logs_index_order`
*   `logs_integration_pipeline`
    * `datadog_logs_integration_pipeline`
*   `logs_pipeline_order`
    * `datadog_logs_pipeline_order`
*   `logs_restriction_query`
    * `datadog_logs_restriction_query`
*   `metric_metadata`
    * `datadog_metric_metadata`
        * **_NOTE:_** Importing resource requires resource ID's to be passed via [Filter][1] option
*   `metric_tag_configuration`
    * `datadog_metric_tag_configuration`
*   `monitor`
    * `datadog_monitor`
*   `monitor_config_policy`
    * `datadog_monitor_config_policy`
*   `monitor_json`
    * `datadog_monitor_json`
*   `monitor_notification_rule`
    * `datadog_monitor_notification_rule`
        * **_NOTE:_** Requires DataDog/datadog provider 3.83.0 or newer.
*   `on_call_escalation_policy`
    * `datadog_on_call_escalation_policy`
        * **_NOTE:_** The Datadog API does not expose a list endpoint for On-Call escalation policies; pass IDs explicitly, for example `--filter=on_call_escalation_policy=policy-id`
*   `on_call_schedule`
    * `datadog_on_call_schedule`
        * **_NOTE:_** The Datadog API does not expose a list endpoint for On-Call schedules; pass IDs explicitly, for example `--filter=on_call_schedule=schedule-id`
*   `on_call_team_routing_rules`
    * `datadog_on_call_team_routing_rules`
        * **_NOTE:_** On-Call team routing rules are keyed by Datadog team ID, for example `--filter=on_call_team_routing_rules=team-id`
*   `on_call_user_notification_channel`
    * `datadog_on_call_user_notification_channel`
        * **_NOTE:_** Importing a single On-Call user notification channel by ID requires quoting the `user_id:channel_id` filter value, for example `--filter="on_call_user_notification_channel='user-id:channel-id'"`
        * **_NOTE:_** To import channels for one user, filter by `user_id`, for example `--filter="Type=on_call_user_notification_channel;Name=user_id;Value=user-id"`
        * **_NOTE:_** Push notification channels are skipped because the Datadog provider resource supports email and phone channels.
*   `on_call_user_notification_rule`
    * `datadog_on_call_user_notification_rule`
        * **_NOTE:_** Importing a single On-Call user notification rule by ID requires quoting the `user_id:rule_id` filter value, for example `--filter="on_call_user_notification_rule='user-id:rule-id'"`
        * **_NOTE:_** To import notification rules for one user, filter by `user_id`, for example `--filter="Type=on_call_user_notification_rule;Name=user_id;Value=user-id"`
*   `org_connection`
    * `datadog_org_connection`
*   `org_group`
    * `datadog_org_group`
        * **_NOTE:_** Requires DataDog/datadog provider 4.8.0 or newer.
*   `org_group_membership`
    * `datadog_org_group_membership`
        * **_NOTE:_** Requires DataDog/datadog provider 4.8.0 or newer.
*   `org_group_policy`
    * `datadog_org_group_policy`
        * **_NOTE:_** Requires DataDog/datadog provider 4.8.0 or newer. Policies are discovered per org group.
*   `organization_settings`
    * `datadog_organization_settings`
        * **_NOTE:_** Singleton-like. Lists org(s) via V1 API and imports each by public ID.
*   `powerpack`
    * `datadog_powerpack`
        * **_NOTE:_** Discovers the same powerpack IDs as `powerpack_v2`; select one powerpack resource representation for each imported powerpack to avoid duplicate Terraform ownership.
*   `powerpack_v2`
    * `datadog_powerpack_v2`
        * **_NOTE:_** Requires DataDog/datadog provider 4.9.0 or newer.
        * **_NOTE:_** Discovers the same powerpack IDs as `powerpack`; select one powerpack resource representation for each imported powerpack to avoid duplicate Terraform ownership.
*   `rum_application`
    * `datadog_rum_application`
*   `rum_metric`
    * `datadog_rum_metric`
*   `rum_retention_filter`
    * `datadog_rum_retention_filter`
        * **_NOTE:_** Importing a single RUM retention filter by ID requires `application_id:retention_filter_id`, for example `--filter="rum_retention_filter='app-id:filter-id'"`
*   `rum_retention_filters_order`
    * `datadog_rum_retention_filters_order`
        * **_NOTE:_** Importing a single RUM retention filters order by ID uses the RUM application ID, for example `--filter=rum_retention_filters_order=app-id`
*   `role`
    * `datadog_role`
*   `security_monitoring_default_rule`
    * `datadog_security_monitoring_default_rule`
*   `security_monitoring_filter`
    * `datadog_security_monitoring_filter`
*   `security_monitoring_rule`
    * `datadog_security_monitoring_rule`
*   `security_monitoring_suppression`
    * `datadog_security_monitoring_suppression`
        * **_NOTE:_** Requires DataDog/datadog provider 3.36.0 or newer.
*   `sensitive_data_scanner_group`
    * `datadog_sensitive_data_scanner_group`
        * **_NOTE:_** Requires DataDog/datadog provider 3.90.0 or newer.
*   `sensitive_data_scanner_group_order`
    * `datadog_sensitive_data_scanner_group_order`
        * **_NOTE:_** Requires DataDog/datadog provider 3.90.0 or newer.
*   `sensitive_data_scanner_rule`
    * `datadog_sensitive_data_scanner_rule`
        * **_NOTE:_** Requires DataDog/datadog provider 3.90.0 or newer.
*   `service_account`
    * `datadog_service_account`
*   `service_account_application_key`
    * `datadog_service_account_application_key`
        * **_NOTE:_** Importing requires `service_account_id` filter or composite `service_account_id:key_id` ID filter, for example `--filter="Type=service_account_application_key;Name=service_account_id;Value=sa-id"` or `--filter="service_account_application_key='sa-id:key-id'"`
*   `service_level_objective`
    * `datadog_service_level_objective`
*   `slo_correction`
    * `datadog_slo_correction`
*   `spans_metric`
    * `datadog_spans_metric`
*   `synthetics_global_variable`
    * `datadog_synthetics_global_variable`
        * **_NOTE:_** Importing resource requires resource ID's to be passed via [Filter][1] option
*   `synthetics_private_location`
    * `datadog_synthetics_private_location`
*   `synthetics_test`
    * `datadog_synthetics_test`
*   `tag_pipeline_ruleset`
    * `datadog_tag_pipeline_ruleset`
        * **_NOTE:_** Requires DataDog/datadog provider 3.39.0 or newer.
*   `team`
    * `datadog_team`
*   `team_connection`
    * `datadog_team_connection`
        * **_NOTE:_** Requires DataDog/datadog provider 4.5.0 or newer.
        * **_NOTE:_** Team connections can be filtered by `source`
*   `team_hierarchy_links`
    * `datadog_team_hierarchy_links`
        * **_NOTE:_** Team hierarchy links can be filtered by `parent_team_id` or `sub_team_id`
*   `team_link`
    * `datadog_team_link`
        * **_NOTE:_** Importing a single team link by ID requires quoting the `team_id:link_id` filter value, for example `--filter="team_link='team-id:link-id'"`; links can also be filtered by `team_id`
*   `team_membership`
    * `datadog_team_membership`
        * **_NOTE:_** Importing a single membership by ID requires quoting the `team_id:user_id` filter value, for example `--filter="team_membership='team-id:user-id'"`; memberships can also be filtered by `team_id`
*   `team_notification_rule`
    * `datadog_team_notification_rule`
        * **_NOTE:_** Requires DataDog/datadog provider 3.85.0 or newer.
        * **_NOTE:_** Importing a single notification rule by ID requires quoting the `team_id:rule_id` filter value, for example `--filter="team_notification_rule='team-id:rule-id'"`; notification rules can also be filtered by `team_id`
*   `team_permission_setting`
    * `datadog_team_permission_setting`
        * **_NOTE:_** Requires DataDog/datadog provider 3.90.0 or newer.
        * **_NOTE:_** Importing a single permission setting by ID requires quoting the `team_id:action` filter value, for example `--filter="team_permission_setting='team-id:manage_membership'"`; permission settings can also be filtered by `team_id`
*   `team_sync`
    * `datadog_team_sync`
        * **_NOTE:_** Requires DataDog/datadog provider 4.5.0 or newer.
        * **_NOTE:_** The Datadog provider currently supports the GitHub team sync source
*   `user`
    * `datadog_user`

## Unsupported / Deferred Resources

The following Terraform provider resources have been evaluated and cannot be safely imported by Terraformer:

| Resource | Reason |
|----------|--------|
| `datadog_integration_aws_account` | Wildcard `--resources=*` conflicts with legacy `integration_aws` generator; required empty blocks (`lambda_forwarder`, `namespace_filters`, `xray_services`) are dropped by Terraformer's flatmap parser before `AllowEmptyValues` is consulted. Revisit after legacy generator is removed. |
| `datadog_integration_aws_event_bridge` | List API returns full event source names (with assigned suffix); provider's required `event_generator_name` is the user-supplied prefix only, and there is no safe way to derive it. |
| `datadog_integration_cloudflare_account` | `api_key` is required and sensitive; read API does not return it. |
| `datadog_integration_confluent_account` | `api_secret` is required and sensitive; read API does not return it. |
| `datadog_integration_fastly_account` | `api_key` is required and sensitive; read API does not return it. |
| `datadog_integration_ms_teams_workflows_webhook_handle` | `url` is required and sensitive; read API does not return it. |
| `datadog_integration_opsgenie_service_object` | `opsgenie_api_key` is required and sensitive; Datadog API explicitly never returns it. |
| `datadog_secure_embed_dashboard` | Deferred because Datadog exposes secure embeds by `dashboard_id:token` only; the API and provider import path require the token and do not provide a list/token discovery endpoint. |

[1]: https://github.com/chenrui333/terraformer/blob/main/README.md#filtering
