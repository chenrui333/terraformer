### Use with Cloudflare

Example using a Cloudflare API Key and corresponding email:
```bash
export CLOUDFLARE_API_KEY=[CLOUDFLARE_API_KEY]
export CLOUDFLARE_EMAIL=[CLOUDFLARE_EMAIL]
export CLOUDFLARE_ACCOUNT_ID=[CLOUDFLARE_ACCOUNT_ID]
./terraformer import cloudflare --resources=dns,firewall,ruleset,access,storage,settings
```

or using a Cloudflare API Token:

```bash
export CLOUDFLARE_API_TOKEN=[CLOUDFLARE_API_TOKEN]
export CLOUDFLARE_ACCOUNT_ID=[CLOUDFLARE_ACCOUNT_ID]
./terraformer import cloudflare --resources=dns,firewall,ruleset,access,storage,settings
```

List of supported Cloudflare services:

* `access`
  * `cloudflare_zero_trust_access_application`
  * `cloudflare_zero_trust_access_custom_page`
  * `cloudflare_zero_trust_access_group`
  * `cloudflare_zero_trust_access_identity_provider`
  * `cloudflare_zero_trust_access_infrastructure_target`
  * `cloudflare_zero_trust_access_mtls_certificate`
  * `cloudflare_zero_trust_access_policy`
  * `cloudflare_zero_trust_access_service_token`
  * `cloudflare_zero_trust_access_tag`
* `account_member`
  * `cloudflare_account_member`
* `certificates`
  * `cloudflare_custom_hostname`
  * `cloudflare_mtls_certificate`
* `dns`
  * `cloudflare_dns_record`
  * `cloudflare_zone`
* `email_routing`
  * `cloudflare_email_routing_address`
  * `cloudflare_email_routing_catch_all`
  * `cloudflare_email_routing_dns`
  * `cloudflare_email_routing_rule`
  * `cloudflare_email_routing_settings`

  Avoid importing `email_routing` with `dns` when you intend to manage Email Routing DNS records
  through `cloudflare_email_routing_dns`.
* `firewall`
  * `cloudflare_access_rule`
  * `cloudflare_filter`
  * `cloudflare_firewall_rule`
  * `cloudflare_rate_limit`
  * `cloudflare_zone_lockdown`
* `lists`
  * `cloudflare_list`
* `load_balancing`
  * `cloudflare_healthcheck`
  * `cloudflare_load_balancer`
  * `cloudflare_load_balancer_monitor`
  * `cloudflare_load_balancer_pool`
* `logpush`
  * `cloudflare_logpush_job`
* `magic_wan`
  * `cloudflare_magic_wan_gre_tunnel`
  * `cloudflare_magic_wan_ipsec_tunnel`
  * `cloudflare_magic_wan_static_route`
* `media_platform`
  * `cloudflare_image_variant`
  * `cloudflare_pipeline`
  * `cloudflare_pipeline_stream`

  `media_platform` requires `CLOUDFLARE_ACCOUNT_ID`. Pipeline streams with schema
  field types outside the Terraform provider's scalar/json validator are skipped.
* `network_edge`
  * `cloudflare_address_map`
  * `cloudflare_magic_network_monitoring_rule`
  * `cloudflare_magic_transit_site`
  * `cloudflare_magic_transit_site_acl`
  * `cloudflare_magic_transit_site_lan`
  * `cloudflare_magic_transit_site_wan`
  * `cloudflare_regional_hostname`
  * `cloudflare_spectrum_application`
  * `cloudflare_web3_hostname`
* `notifications`
  * `cloudflare_notification_policy`
  * `cloudflare_notification_policy_webhooks`
* `page_rule`
  * `cloudflare_page_rule`
* `pages`
  * `cloudflare_pages_domain`
  * `cloudflare_pages_project`
* `ruleset`
  * `cloudflare_ruleset`
* `security`
  * `cloudflare_api_shield`
  * `cloudflare_api_shield_operation`
  * `cloudflare_cloud_connector_rules`
  * `cloudflare_custom_page_asset`
  * `cloudflare_custom_pages`
  * `cloudflare_email_security_block_sender`
  * `cloudflare_email_security_impersonation_registry`
  * `cloudflare_leaked_credential_check_rule`
  * `cloudflare_page_shield_policy`
  * `cloudflare_schema_validation_schemas`
  * `cloudflare_token_validation_config`
  * `cloudflare_token_validation_rules`
  * `cloudflare_user_agent_blocking_rule`
  * `cloudflare_vulnerability_scanner_credential_set`
  * `cloudflare_vulnerability_scanner_target_environment`

  `security` requires `CLOUDFLARE_ACCOUNT_ID` for account-scoped Email Security
  and custom page resources. Default Cloudflare custom pages are skipped; only
  customized pages are imported.
* `settings`
  * `cloudflare_account_dns_settings_internal_view`
  * `cloudflare_argo_smart_routing`
  * `cloudflare_argo_tiered_caching`
  * `cloudflare_authenticated_origin_pulls_settings`
  * `cloudflare_custom_hostname_fallback_origin`
  * `cloudflare_dns_firewall`
  * `cloudflare_dns_zone_transfers_acl`
  * `cloudflare_dns_zone_transfers_incoming`
  * `cloudflare_dns_zone_transfers_outgoing`
  * `cloudflare_dns_zone_transfers_peer`
  * `cloudflare_leaked_credential_check`
  * `cloudflare_logpull_retention`
  * `cloudflare_managed_transforms`
  * `cloudflare_regional_tiered_cache`
  * `cloudflare_tiered_cache`
  * `cloudflare_total_tls`
  * `cloudflare_universal_ssl_setting`
  * `cloudflare_url_normalization_settings`
  * `cloudflare_waiting_room_settings`
  * `cloudflare_zone_cache_reserve`
  * `cloudflare_zone_cache_variants`
  * `cloudflare_zone_dnssec`
  * `cloudflare_zone_hold`
  * `cloudflare_zone_setting`

  Account-scoped settings and DNS transfer resources require `CLOUDFLARE_ACCOUNT_ID`.
  Zone singleton settings are imported only when Terraformer can see durable, explicit
  user-owned configuration. Cloudflare defaults are skipped so generated Terraform does not
  claim ownership of unset account or zone settings. Generic zone settings use a conservative
  allowlist and require Cloudflare modification metadata before import.
* `storage`
  * `cloudflare_d1_database`
  * `cloudflare_queue`
  * `cloudflare_queue_consumer`
  * `cloudflare_r2_bucket`
  * `cloudflare_r2_bucket_cors`
  * `cloudflare_r2_bucket_event_notification`
  * `cloudflare_r2_bucket_lifecycle`
  * `cloudflare_r2_bucket_lock`
  * `cloudflare_r2_custom_domain`
  * `cloudflare_r2_data_catalog`
  * `cloudflare_workers_kv_namespace`
* `turnstile`
  * `cloudflare_turnstile_widget`
* `tunnel`
  * `cloudflare_zero_trust_tunnel_cloudflared`
  * `cloudflare_zero_trust_tunnel_cloudflared_virtual_network`
* `waiting_room`
  * `cloudflare_waiting_room`
  * `cloudflare_waiting_room_event`
  * `cloudflare_waiting_room_rules`
* `web_analytics`
  * `cloudflare_web_analytics_site`
* `workers`
  * `cloudflare_worker`
  * `cloudflare_workers_cron_trigger`
  * `cloudflare_workers_custom_domain`
  * `cloudflare_workers_for_platforms_dispatch_namespace`
  * `cloudflare_workers_route`
* `zero_trust_gateway`
  * `cloudflare_zero_trust_dns_location`
  * `cloudflare_zero_trust_gateway_certificate`
  * `cloudflare_zero_trust_gateway_logging`
  * `cloudflare_zero_trust_gateway_pacfile`
  * `cloudflare_zero_trust_gateway_policy`
  * `cloudflare_zero_trust_gateway_proxy_endpoint`
  * `cloudflare_zero_trust_gateway_settings`
  * `cloudflare_zero_trust_list`
  * `cloudflare_zero_trust_network_hostname_route`

Unsupported and deferred Cloudflare import decisions are tracked in
[unsupported_resources.json](../providers/cloudflare/unsupported_resources.json).
