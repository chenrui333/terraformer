### Use with Vultr

Example:

```
export VULTR_API_KEY=[VULTR_API_KEY]
./terraformer import vultr -r instance
```

List of supported Vultr resources:

*   `bare_metal_server`
    * `vultr_bare_metal_server`
*   `block_storage`
    * `vultr_block_storage`
*   `dns_domain`
    * `vultr_dns_domain`
    * `vultr_dns_record`
*   `firewall_group`
    * `vultr_firewall_group`
    * `vultr_firewall_rule`
*   `instance`
    * `vultr_instance`
*   `reserved_ip`
    * `vultr_reserved_ip`
*   `snapshot`
    * `vultr_snapshot`
*   `ssh_key`
    * `vultr_ssh_key`
*   `startup_script`
    * `vultr_startup_script`
*   `user`
    * `vultr_user`
*   `vpc`
    * `vultr_vpc`
