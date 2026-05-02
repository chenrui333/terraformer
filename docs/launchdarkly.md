### Use with LaunchDarkly

Example:

```
export LAUNCHDARKLY_ACCESS_TOKEN=[LAUNCHDARKLY_ACCESS_TOKEN]
./terraformer import launchdarkly -r auditLogSubscription,customRole,destination,environment,featureFlag,flagTemplates,flagTrigger,metric,relayProxyConfiguration,segment,team,teamMember,webhook
```

Use `project` separately when you want LaunchDarkly environments managed as nested
`launchdarkly_project` blocks. Avoid importing `project` and `environment` together,
because that can generate both nested and standalone resources for the same
environments.

List of supported LaunchDarkly resources:

*   `auditLogSubscription`
    * `launchdarkly_audit_log_subscription`
*   `customRole`
    * `launchdarkly_custom_role`
*   `destination`
    * `launchdarkly_destination`
*   `project`
    * `launchdarkly_project`
*   `environment`
    * `launchdarkly_environment`
*   `featureFlag`
    * `launchdarkly_feature_flag`
    * `launchdarkly_feature_flag_environment`
*   `flagTemplates`
    * `launchdarkly_flag_templates`
*   `flagTrigger`
    * `launchdarkly_flag_trigger`
*   `metric`
    * `launchdarkly_metric`
*   `relayProxyConfiguration`
    * `launchdarkly_relay_proxy_configuration`
*   `segment`
    * `launchdarkly_segment`
*   `team`
    * `launchdarkly_team`
*   `teamMember`
    * `launchdarkly_team_member`
*   `webhook`
    * `launchdarkly_webhook`
