### Use with Kubernetes

Example:

```
 terraformer import kubernetes --resources=deployments,services,storageclasses
 terraformer import kubernetes --resources=deployments,services,storageclasses --filter=deployment=name1:name2:name3
```

Terraformer discovers Kubernetes API resources from the active cluster and imports resources that are available through either the typed Kubernetes client or an explicit dynamic-client import path, and the installed Terraform Kubernetes provider schema.
Discovered CRDs, other untyped API extensions, and selected native APIs without a first-class Terraform Kubernetes provider type can be imported through `kubernetes_manifest` when the API resource is manageable and the installed provider supports that resource.
Manifest-backed resources use a group/version-qualified resource selector such as `example.com/v1/widgets`, `apps/v1/replicasets`, `v1/podtemplates`, or `admissionregistration.k8s.io/v1/validatingadmissionpolicybindings` to avoid collisions between API resources that share the same plural name.
For selected native manifest-backed resources, beta and alpha variants are supported when the cluster advertises them and they satisfy the required management verbs. For example, use selectors like `certificates.k8s.io/v1beta1/clustertrustbundles`, `scheduling.k8s.io/v1alpha2/workloads`, or `storagemigration.k8s.io/v1alpha1/storageversionmigrations` on clusters that still serve those versions.
Terraformer intentionally skips `PodCertificateRequest` (`podcertificaterequests`) even when served, because kubelets generate these runtime certificate request objects and their specs include pod, node, service account, and proof material that should not become Terraform-owned configuration.
When `labels` or `annotations` are selected with full resource imports, Terraformer keeps the full resource import and skips overlapping metadata-only resources to avoid duplicate Terraform ownership of the same object metadata.
When `env` is selected with full workload imports, Terraformer keeps the full workload import and skips overlapping environment-only resources to avoid duplicate Terraform ownership of the same container environment.
When `configmaps` and `configmapdata` are selected together, Terraformer keeps the full `configmaps` import and skips overlapping data-only resources to avoid duplicate Terraform ownership of the same ConfigMap data.
When `secrets` and `secretdata` are selected together, Terraformer keeps the full `secrets` import and skips overlapping data-only resources to avoid duplicate Terraform ownership of the same Secret data.
Because `kubernetes_secret_v1_data` only accepts string data, `secretdata` skips Secrets containing non-UTF-8 payloads instead of emitting lossy configuration.

Common supported resources include:

*   `annotations`
    * `kubernetes_annotations`
*   `admissionregistration.k8s.io/v1/mutatingadmissionpolicies`
    * `kubernetes_manifest`
*   `admissionregistration.k8s.io/v1/mutatingadmissionpolicybindings`
    * `kubernetes_manifest`
*   `admissionregistration.k8s.io/v1/validatingadmissionpolicybindings`
    * `kubernetes_manifest`
*   `apiservices`
    * `kubernetes_api_service_v1`
*   `apps/v1/replicasets`
    * `kubernetes_manifest`
*   `certificatesigningrequests`
    * `kubernetes_certificate_signing_request_v1`
*   `certificates.k8s.io/v1beta1/clustertrustbundles`
    * `kubernetes_manifest`
*   `certificates.k8s.io/v1alpha1/clustertrustbundles`
    * `kubernetes_manifest`
*   `clusterroles`
    * `kubernetes_cluster_role_v1`
*   `clusterrolebindings`
    * `kubernetes_cluster_role_binding_v1`
*   `configmaps`
    * `kubernetes_config_map_v1`
*   `configmapdata`
    * `kubernetes_config_map_v1_data`
*   `cronjobs`
    * `kubernetes_cron_job_v1`
*   `csidrivers`
    * `kubernetes_csi_driver_v1`
*   `daemonsets`
    * `kubernetes_daemon_set_v1`
*   `defaultserviceaccounts`
    * `kubernetes_default_service_account_v1`
*   `deployments`
    * `kubernetes_deployment_v1`
*   `endpoints`
    * `kubernetes_endpoints_v1`
*   `endpointslices`
    * `kubernetes_endpoint_slice_v1`
*   `env`
    * `kubernetes_env`
*   discovered CRDs and custom resources
    * `kubernetes_manifest`
*   `flowcontrol.apiserver.k8s.io/v1/flowschemas`
    * `kubernetes_manifest`
*   `flowcontrol.apiserver.k8s.io/v1/prioritylevelconfigurations`
    * `kubernetes_manifest`
*   `flowcontrol.apiserver.k8s.io/v1beta3/flowschemas`
    * `kubernetes_manifest`
*   `horizontalpodautoscalers`
    * `kubernetes_horizontal_pod_autoscaler_v2`
    * `kubernetes_horizontal_pod_autoscaler_v2beta2`
*   `ingressclasses`
    * `kubernetes_ingress_class_v1`
*   `ingresses`
    * `kubernetes_ingress_v1`
*   `jobs`
    * `kubernetes_job_v1`
*   `labels`
    * `kubernetes_labels`
*   `limitranges`
    * `kubernetes_limit_range_v1`
*   `mutatingwebhookconfigurations`
    * `kubernetes_mutating_webhook_configuration_v1`
*   `namespaces`
    * `kubernetes_namespace_v1`
*   `networkpolicies`
    * `kubernetes_network_policy_v1`
*   `networking.k8s.io/v1/servicecidrs`
    * `kubernetes_manifest`
*   `networking.k8s.io/v1beta1/servicecidrs`
    * `kubernetes_manifest`
*   `nodetaints`
    * `kubernetes_node_taint`
*   `persistentvolumes`
    * `kubernetes_persistent_volume_v1`
*   `persistentvolumeclaims`
    * `kubernetes_persistent_volume_claim_v1`
*   `pods`
    * `kubernetes_pod_v1`
*   `v1/podtemplates`
    * `kubernetes_manifest`
*   `podsecuritypolicies`
    * `kubernetes_pod_security_policy`
*   `poddisruptionbudgets`
    * `kubernetes_pod_disruption_budget_v1`
*   `priorityclasses`
    * `kubernetes_priority_class_v1`
*   `replicationcontrollers`
    * `kubernetes_replication_controller_v1`
*   `resource.k8s.io/v1/deviceclasses`
    * `kubernetes_manifest`
*   `resource.k8s.io/v1/resourceclaims`
    * `kubernetes_manifest`
*   `resource.k8s.io/v1/resourceclaimtemplates`
    * `kubernetes_manifest`
*   `resource.k8s.io/v1beta2/devicetaintrules`
    * `kubernetes_manifest`
*   `resource.k8s.io/v1alpha3/devicetaintrules`
    * `kubernetes_manifest`
*   `resourcequotas`
    * `kubernetes_resource_quota_v1`
*   `roles`
    * `kubernetes_role_v1`
*   `rolebindings`
    * `kubernetes_role_binding_v1`
*   `runtimeclasses`
    * `kubernetes_runtime_class_v1`
*   `scheduling.k8s.io/v1alpha2/workloads`
    * `kubernetes_manifest`
*   `scheduling.k8s.io/v1alpha1/workloads`
    * `kubernetes_manifest`
*   `secrets`
    * `kubernetes_secret_v1`
*   `secretdata`
    * `kubernetes_secret_v1_data`
*   `services`
    * `kubernetes_service_v1`
*   `serviceaccounts`
    * `kubernetes_service_account_v1`
*   `statefulsets`
    * `kubernetes_stateful_set_v1`
*   `storageclasses`
    * `kubernetes_storage_class_v1`
*   `storage.k8s.io/v1/volumeattributesclasses`
    * `kubernetes_manifest`
*   `storage.k8s.io/v1alpha1/volumeattributesclasses`
    * `kubernetes_manifest`
*   `storagemigration.k8s.io/v1beta1/storageversionmigrations`
    * `kubernetes_manifest`
*   `storagemigration.k8s.io/v1alpha1/storageversionmigrations`
    * `kubernetes_manifest`
*   `validatingadmissionpolicies`
    * `kubernetes_validating_admission_policy_v1`
*   `validatingwebhookconfigurations`
    * `kubernetes_validating_webhook_configuration_v1`
    
#### Known issues

* Terraform Kubernetes provider is rejecting resources with ":" characters in their names (as they don't meet DNS-1123), while it's allowed for certain types in Kubernetes, e.g. ClusterRoleBinding.
* Because Terraform flatmap uses "." to detect the keys for unflattening the maps, some keys with "." in their names are being considered as the maps.
* Since the library assumes empty strings to be empty values (not "0"), there are some issues with optional integer keys that are restricted to be positive.
