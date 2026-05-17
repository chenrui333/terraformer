### Use with LaunchDarkly

Example:

```
export LAUNCHDARKLY_ACCESS_TOKEN=[LAUNCHDARKLY_ACCESS_TOKEN]
./terraformer import launchdarkly -r accessToken,aiConfig,aiConfigVariation,aiTool,auditLogSubscription,customRole,destination,environment,featureFlag,flagTemplates,flagTrigger,metric,modelConfig,relayProxyConfiguration,segment,team,teamMember,view,webhook
```

Use `project` separately when you want LaunchDarkly environments managed as nested
`launchdarkly_project` blocks. Avoid importing `project` and `environment` together,
because that can generate both nested and standalone resources for the same
environments.

Use `viewLinks` separately when you want LaunchDarkly view associations managed
as centralized `launchdarkly_view_links` blocks. Avoid importing `viewLinks`
together with `featureFlag` or `segment`, because those resources can manage the
same associations through `view_keys`.

Unsupported or deferred LaunchDarkly resource decisions are tracked in
[unsupported_resources.json](../providers/launchdarkly/unsupported_resources.json).

List of supported LaunchDarkly resources:

*   `accessToken`
    * `launchdarkly_access_token`
*   `aiConfig`
    * `launchdarkly_ai_config`
*   `aiConfigVariation`
    * `launchdarkly_ai_config_variation`
*   `aiTool`
    * `launchdarkly_ai_tool`
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
*   `modelConfig`
    * `launchdarkly_model_config`
*   `relayProxyConfiguration`
    * `launchdarkly_relay_proxy_configuration`
*   `segment`
    * `launchdarkly_segment`
*   `team`
    * `launchdarkly_team`
*   `teamMember`
    * `launchdarkly_team_member`
*   `view`
    * `launchdarkly_view`
*   `viewLinks`
    * `launchdarkly_view_links`
*   `webhook`
    * `launchdarkly_webhook`
