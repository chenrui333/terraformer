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
      version = ">= 3.86.0"
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

# Import based on multiple resource IDs
 ./terraformer import datadog --resources=monitor --filter=monitor=id1:id2:id4
```

Tag filters are order specific. For example, if your monitor has tags (in the order) `atag: atagvalue`, `foo:bar` but you filter for `--filter="Name=tags;Value='foo:bar'" --filter="Name=tags;Value='atag: atagvalue'"`, the monitor would not be imported.

## Supported Datadog resources

*   `apm_retention_filter`
    * `datadog_apm_retention_filter`
*   `apm_retention_filter_order`
    * `datadog_apm_retention_filter_order`
        * **_NOTE:_** Importing a single retention filter order by ID accepts any value because the Datadog provider stores it as `filtersOrderID`
*   `dashboard`
    * `datadog_dashboard`
*   `dashboard_json`
    * `datadog_dashboard_json`
*   `dashboard_list`
    * `datadog_dashboard_list`
*   `cloud_inventory_sync_config`
    * `datadog_cloud_inventory_sync_config`
        * **_NOTE:_** Requires DataDog/datadog provider 3.86.0 or newer.
        * **_NOTE:_** Importing resource requires resource ID's to be passed via [Filter][1] option
*   `downtime`
    * `datadog_downtime`
*   `integration_aws`
    * `datadog_integration_aws`
*   `integration_aws_lambda_arn`
    * `datadog_integration_aws_lambda_arn`
*   `integration_aws_log_collection`
    * `datadog_integration_aws_log_collection`
*   `integration_azure`
    * `datadog_integration_azure`
        * **_NOTE:_** Sensitive field `client_secret` is not generated and needs to be manually set
*   `integration_gcp`
    * `datadog_integration_gcp`
        * **_NOTE:_** Sensitive fields `private_key, private_key_id, client_id` is not generated and needs to be manually set
*   `integration_pagerduty`
    * `datadog_integration_pagerduty`
*   `integration_pagerduty_service_object`
    * `datadog_integration_pagerduty_service_object`
*   `integration_slack_channel`
    * `datadog_integration_slack_channel`
        * **_NOTE:_** Importing resource requires resource ID or `account_name` to be passed via [Filter][1] option
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
*   `rum_metric`
    * `datadog_rum_metric`
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
*   `team`
    * `datadog_team`
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
*   `user`
    * `datadog_user`

[1]: https://github.com/chenrui333/terraformer/blob/main/README.md#filtering
