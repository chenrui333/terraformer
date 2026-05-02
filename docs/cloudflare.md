### Use with Cloudflare

Example using a Cloudflare API Key and corresponding email:
```bash
export CLOUDFLARE_API_KEY=[CLOUDFLARE_API_KEY]
export CLOUDFLARE_EMAIL=[CLOUDFLARE_EMAIL]
export CLOUDFLARE_ACCOUNT_ID=[CLOUDFLARE_ACCOUNT_ID]
./terraformer import cloudflare --resources=dns,firewall,ruleset,access,storage
```

or using a Cloudflare API Token:

```bash
export CLOUDFLARE_API_TOKEN=[CLOUDFLARE_API_TOKEN]
export CLOUDFLARE_ACCOUNT_ID=[CLOUDFLARE_ACCOUNT_ID]
./terraformer import cloudflare --resources=dns,firewall,ruleset,access,storage
```

List of supported Cloudflare services:

* `access`
  * `cloudflare_zero_trust_access_application`
  * `cloudflare_zero_trust_access_custom_page`
  * `cloudflare_zero_trust_access_group`
  * `cloudflare_zero_trust_access_identity_provider`
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
* `storage`
  * `cloudflare_d1_database`
  * `cloudflare_queue`
  * `cloudflare_r2_bucket`
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
  * `cloudflare_workers_route`
