### Use with Kubernetes

Example:

```
 terraformer import kubernetes --resources=deployments,services,storageclasses
 terraformer import kubernetes --resources=deployments,services,storageclasses --filter=deployment=name1:name2:name3
```

Terraformer discovers Kubernetes API resources from the active cluster and imports resources that are available through either the typed Kubernetes client or an explicit dynamic-client import path, and the installed Terraform Kubernetes provider schema.
This provider's native Kubernetes resource policy is scoped to Kubernetes 1.34 through 1.36; historical alpha APIs that only existed before that support window are not added as native manifest selectors.
Discovered CRDs, other untyped API extensions, and selected native APIs without a first-class Terraform Kubernetes provider type can be imported through `kubernetes_manifest` when the API resource is manageable and the installed provider supports that resource.
Manifest-backed resources use a group/version-qualified resource selector such as `example.com/v1/widgets`, `apps/v1/replicasets`, `v1/podtemplates`, or `admissionregistration.k8s.io/v1/validatingadmissionpolicybindings` to avoid collisions between API resources that share the same plural name.
For selected native manifest-backed resources, beta and alpha variants are supported when the cluster advertises them and they satisfy the required management verbs. For example, use selectors like `certificates.k8s.io/v1beta1/clustertrustbundles`, `scheduling.k8s.io/v1alpha2/workloads`, or `storagemigration.k8s.io/v1alpha1/storageversionmigrations` on clusters that still serve those versions.
Native Kubernetes API groups only use `kubernetes_manifest` when the resource kind is explicitly selected for manifest-backed import; unselected native kinds are skipped instead of falling through the generic CRD import path. Run with `--verbose` to log native API resources skipped by this policy or by generated-resource safeguards.
Terraformer intentionally skips `PodCertificateRequest` (`podcertificaterequests`) even when served, because kubelets generate these runtime certificate request objects and their specs include pod, node, service account, and proof material that should not become Terraform-owned configuration.
Terraformer also skips native runtime or controller-generated APIs such as `ResourceSlice`, `PodScheduling`/`PodSchedulingContext`, `IPAddress`, `ControllerRevision`, `LeaseCandidate`, `CSINode`, `CSIStorageCapacity`, and `VolumeAttachment`, even when an older served version is not recognized by the pinned typed client.
Structured unsupported-resource metadata for these Kubernetes import-policy skips is tracked in `providers/kubernetes/unsupported_resources.json`.
When `labels` or `annotations` are selected with full resource imports, Terraformer keeps the full resource import and skips overlapping metadata-only resources to avoid duplicate Terraform ownership of the same object metadata.
When `env` is selected with full workload imports, Terraformer keeps the full workload import and skips overlapping environment-only resources to avoid duplicate Terraform ownership of the same container environment.
When `configmaps` and `configmapdata` are selected together, Terraformer keeps the full `configmaps` import and skips overlapping data-only resources to avoid duplicate Terraform ownership of the same ConfigMap data.
When `secrets` and `secretdata` are selected together, Terraformer keeps the full `secrets` import and skips overlapping data-only resources to avoid duplicate Terraform ownership of the same Secret data.
Because `kubernetes_secret_v1_data` only accepts string data, `secretdata` skips Secrets containing non-UTF-8 payloads instead of emitting lossy configuration.

#### Kubernetes 1.34-1.36 support window

This provider's native API policy is scoped to Kubernetes 1.34 through 1.36.
Every API resource discovered from the cluster is classified into exactly one
behavior class:

| Behavior class | Description | Example |
|---|---|---|
| First-class Terraform resource | Imported via the typed or dynamic Kubernetes client with a dedicated provider resource type (e.g. `kubernetes_deployment_v1`). | `apps/v1 Deployment`, `v1 Service`, `storage.k8s.io/v1 StorageClass` |
| Explicit native `kubernetes_manifest` selector | A native Kubernetes API that is explicitly allowlisted for manifest-backed import because no first-class provider type covers it. | `admissionregistration.k8s.io/v1 MutatingAdmissionPolicy`, `resource.k8s.io/v1 DeviceClass`, `apps/v1 ReplicaSet` |
| CRD/custom-resource manifest fallback | Any non-native API group (CRDs, operator resources, third-party extensions) is imported through `kubernetes_manifest` automatically. | `example.com/v1 Widget`, `serving.knative.dev/v1 Service` |
| Runtime/controller-generated skip | Native APIs whose objects are created by kubelets, controllers, or drivers and should not become Terraform-owned configuration. | `resource.k8s.io/v1 ResourceSlice`, `certificates.k8s.io/v1beta1 PodCertificateRequest`, `storage.k8s.io/v1 VolumeAttachment` |
| Policy skip | Native APIs that are manageable but intentionally not imported: either no Terraform provider type exists, or the API is outside the explicit manifest selector allowlist. | `events.k8s.io/v1 Event`, `coordination.k8s.io/v1 Lease`, `resource.k8s.io/v1alpha2 ResourceClass` |

**Why non-declarative resources are skipped:**
Resources like `PodCertificateRequest`, `ResourceSlice`, `PodSchedulingContext`,
`IPAddress`, `ControllerRevision`, `LeaseCandidate`, `CSINode`, and
`VolumeAttachment` are generated at runtime by kubelets, schedulers, CSI
drivers, or controllers. Their specs contain ephemeral proof material, node
bindings, or scheduling state that would create noisy, non-idempotent Terraform
plans. Importing them would not produce meaningful infrastructure-as-code.

**Why token/request/action-style resources are not imported:**
Resources like `TokenRequest`, `TokenReview`, `SubjectAccessReview`,
`SelfSubjectAccessReview`, and `LocalSubjectAccessReview` are action-style APIs
that create ephemeral tokens or perform one-shot authorization checks. They do
not represent persistent cluster state and cannot be meaningfully managed as
Terraform resources. These are skipped because the Terraform provider does not
expose them as resource types and the native manifest policy does not allowlist
them.

**`--verbose` skip logging:**
Run with `--verbose` to see which native APIs are skipped and why. Two reason
strings are emitted:
- `"runtime/controller-generated native API is not importable as Terraform-managed configuration"` — for explicitly skipped generated resources.
- `"native API is outside the explicit manifest import policy"` — for manageable native APIs without typed client support that are not in the manifest allowlist.

Native APIs with typed client support but no registered Terraform provider type
(e.g. `Event`, `Lease`) are silently skipped without a verbose reason.

These behaviors are enforced by `TestKubernetes134To136APIDiscoveryMatrix`,
`TestVerboseSkipLoggingForNativeAPIs`, and `TestCRDManifestFallbackNotBroken`
in `providers/kubernetes/utils_test.go`.

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
    * Standalone ReplicaSets only; Deployment-owned ReplicaSets are ignored.
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
*   `resource.k8s.io/v1beta2/deviceclasses`
    * `kubernetes_manifest`
*   `resource.k8s.io/v1beta2/resourceclaims`
    * `kubernetes_manifest`
*   `resource.k8s.io/v1beta2/resourceclaimtemplates`
    * `kubernetes_manifest`
*   `resource.k8s.io/v1beta2/devicetaintrules`
    * `kubernetes_manifest`
*   `resource.k8s.io/v1beta1/deviceclasses`
    * `kubernetes_manifest`
*   `resource.k8s.io/v1beta1/resourceclaims`
    * `kubernetes_manifest`
*   `resource.k8s.io/v1beta1/resourceclaimtemplates`
    * `kubernetes_manifest`
*   `resource.k8s.io/v1alpha3/deviceclasses`
    * `kubernetes_manifest`
*   `resource.k8s.io/v1alpha3/devicetaintrules`
    * `kubernetes_manifest`
*   `resource.k8s.io/v1alpha3/resourceclaims`
    * `kubernetes_manifest`
*   `resource.k8s.io/v1alpha3/resourceclaimtemplates`
    * `kubernetes_manifest`
*   `resourcequotas`
    * `kubernetes_resource_quota_v1`
*   `roles`
    * `kubernetes_role_v1`
*   `rolebindings`
    * `kubernetes_role_binding_v1`
*   `runtimeclasses`
    * `kubernetes_runtime_class_v1`
*   `scheduling.k8s.io/v1alpha2/podgroups`
    * `kubernetes_manifest`
*   `scheduling.k8s.io/v1alpha2/workloads`
    * `kubernetes_manifest`
*   `scheduling.k8s.io/v1alpha1/podgroups`
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
