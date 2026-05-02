### Use with Kubernetes

Example:

```
 terraformer import kubernetes --resources=deployments,services,storageclasses
 terraformer import kubernetes --resources=deployments,services,storageclasses --filter=deployment=name1:name2:name3
```

Terraformer discovers Kubernetes API resources from the active cluster and imports resources that are available through either the typed Kubernetes client or an explicit dynamic-client import path, and the installed Terraform Kubernetes provider schema.

Common supported resources include:

*   `apiservices`
    * `kubernetes_api_service_v1`
*   `certificatesigningrequests`
    * `kubernetes_certificate_signing_request_v1`
*   `clusterroles`
    * `kubernetes_cluster_role_v1`
*   `clusterrolebindings`
    * `kubernetes_cluster_role_binding_v1`
*   `configmaps`
    * `kubernetes_config_map_v1`
*   `cronjobs`
    * `kubernetes_cron_job_v1`
*   `csidrivers`
    * `kubernetes_csi_driver_v1`
*   `daemonsets`
    * `kubernetes_daemon_set_v1`
*   `deployments`
    * `kubernetes_deployment_v1`
*   `endpoints`
    * `kubernetes_endpoints_v1`
*   `endpointslices`
    * `kubernetes_endpoint_slice_v1`
*   `horizontalpodautoscalers`
    * `kubernetes_horizontal_pod_autoscaler_v2`
    * `kubernetes_horizontal_pod_autoscaler_v2beta2`
*   `ingressclasses`
    * `kubernetes_ingress_class_v1`
*   `ingresses`
    * `kubernetes_ingress_v1`
*   `jobs`
    * `kubernetes_job_v1`
*   `limitranges`
    * `kubernetes_limit_range_v1`
*   `mutatingwebhookconfigurations`
    * `kubernetes_mutating_webhook_configuration_v1`
*   `namespaces`
    * `kubernetes_namespace_v1`
*   `networkpolicies`
    * `kubernetes_network_policy_v1`
*   `persistentvolumes`
    * `kubernetes_persistent_volume_v1`
*   `persistentvolumeclaims`
    * `kubernetes_persistent_volume_claim_v1`
*   `pods`
    * `kubernetes_pod_v1`
*   `podsecuritypolicies`
    * `kubernetes_pod_security_policy`
*   `poddisruptionbudgets`
    * `kubernetes_pod_disruption_budget_v1`
*   `priorityclasses`
    * `kubernetes_priority_class_v1`
*   `replicationcontrollers`
    * `kubernetes_replication_controller_v1`
*   `resourcequotas`
    * `kubernetes_resource_quota_v1`
*   `roles`
    * `kubernetes_role_v1`
*   `rolebindings`
    * `kubernetes_role_binding_v1`
*   `runtimeclasses`
    * `kubernetes_runtime_class_v1`
*   `secrets`
    * `kubernetes_secret_v1`
*   `services`
    * `kubernetes_service_v1`
*   `serviceaccounts`
    * `kubernetes_service_account_v1`
*   `statefulsets`
    * `kubernetes_stateful_set_v1`
*   `storageclasses`
    * `kubernetes_storage_class_v1`
*   `validatingadmissionpolicies`
    * `kubernetes_validating_admission_policy_v1`
*   `validatingwebhookconfigurations`
    * `kubernetes_validating_webhook_configuration_v1`
    
#### Known issues

* Terraform Kubernetes provider is rejecting resources with ":" characters in their names (as they don't meet DNS-1123), while it's allowed for certain types in Kubernetes, e.g. ClusterRoleBinding.
* Because Terraform flatmap uses "." to detect the keys for unflattening the maps, some keys with "." in their names are being considered as the maps.
* Since the library assumes empty strings to be empty values (not "0"), there are some issues with optional integer keys that are restricted to be positive.
