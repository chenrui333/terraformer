### Use with Kubernetes

Example:

```
 terraformer import kubernetes --resources=deployments,services,storageclasses
 terraformer import kubernetes --resources=deployments,services,storageclasses --filter=deployment=name1:name2:name3
```

Terraformer discovers Kubernetes API resources from the active cluster and imports resources that are available through both the typed Kubernetes client and the installed Terraform Kubernetes provider schema.

Common supported resources include:

*   `clusterroles`
    * `kubernetes_cluster_role`
*   `clusterrolebindings`
    * `kubernetes_cluster_role_binding`
*   `configmaps`
    * `kubernetes_config_map`
*   `cronjobs`
    * `kubernetes_cron_job_v1`
*   `csidrivers`
    * `kubernetes_csi_driver_v1`
*   `daemonsets`
    * `kubernetes_daemon_set_v1`
*   `deployments`
    * `kubernetes_deployment`
*   `endpointslices`
    * `kubernetes_endpoint_slice_v1`
*   `horizontalpodautoscalers`
    * `kubernetes_horizontal_pod_autoscaler_v2`
*   `ingressclasses`
    * `kubernetes_ingress_class_v1`
*   `ingresses`
    * `kubernetes_ingress_v1`
*   `jobs`
    * `kubernetes_job_v1`
*   `limitranges`
    * `kubernetes_limit_range`
*   `mutatingwebhookconfigurations`
    * `kubernetes_mutating_webhook_configuration_v1`
*   `namespaces`
    * `kubernetes_namespace`
*   `networkpolicies`
    * `kubernetes_network_policy_v1`
*   `persistentvolumes`
    * `kubernetes_persistent_volume`
*   `persistentvolumeclaims`
    * `kubernetes_persistent_volume_claim`
*   `pods`
    * `kubernetes_pod`
*   `poddisruptionbudgets`
    * `kubernetes_pod_disruption_budget_v1`
*   `priorityclasses`
    * `kubernetes_priority_class_v1`
*   `replicationcontrollers`
    * `kubernetes_replication_controller`
*   `resourcequotas`
    * `kubernetes_resource_quota`
*   `roles`
    * `kubernetes_role`
*   `rolebindings`
    * `kubernetes_role_binding`
*   `runtimeclasses`
    * `kubernetes_runtime_class_v1`
*   `secrets`
    * `kubernetes_secret`
*   `services`
    * `kubernetes_service`
*   `serviceaccounts`
    * `kubernetes_service_account`
*   `statefulsets`
    * `kubernetes_stateful_set`
*   `storageclasses`
    * `kubernetes_storage_class`
*   `validatingadmissionpolicies`
    * `kubernetes_validating_admission_policy_v1`
*   `validatingwebhookconfigurations`
    * `kubernetes_validating_webhook_configuration_v1`
    
#### Known issues

* Terraform Kubernetes provider is rejecting resources with ":" characters in their names (as they don't meet DNS-1123), while it's allowed for certain types in Kubernetes, e.g. ClusterRoleBinding.
* Because Terraform flatmap uses "." to detect the keys for unflattening the maps, some keys with "." in their names are being considered as the maps.
* Since the library assumes empty strings to be empty values (not "0"), there are some issues with optional integer keys that are restricted to be positive.
