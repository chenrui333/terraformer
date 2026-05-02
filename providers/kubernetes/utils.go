// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package kubernetes

import (
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
	"k8s.io/client-go/kubernetes"
)

type kubernetesResourceID struct {
	group   string
	version string
	kind    string
}

var preferredTerraformResourceNames = map[kubernetesResourceID][]string{
	{group: "apiregistration.k8s.io", version: "v1", kind: "APIService"}:                                {"kubernetes_api_service_v1", "kubernetes_api_service"},
	{group: "apiregistration.k8s.io", version: "v1beta1", kind: "APIService"}:                           {"kubernetes_api_service"},
	{group: "apps", version: "v1", kind: "DaemonSet"}:                                                   {"kubernetes_daemon_set_v1", "kubernetes_daemonset"},
	{group: "apps", version: "v1", kind: "Deployment"}:                                                  {"kubernetes_deployment_v1", "kubernetes_deployment"},
	{group: "apps", version: "v1", kind: "StatefulSet"}:                                                 {"kubernetes_stateful_set_v1", "kubernetes_stateful_set"},
	{group: "apps", version: "v1beta1", kind: "DaemonSet"}:                                              {"kubernetes_daemonset"},
	{group: "apps", version: "v1beta2", kind: "DaemonSet"}:                                              {"kubernetes_daemonset"},
	{group: "autoscaling", version: "v1", kind: "HorizontalPodAutoscaler"}:                              {"kubernetes_horizontal_pod_autoscaler_v1", "kubernetes_horizontal_pod_autoscaler"},
	{group: "autoscaling", version: "v2", kind: "HorizontalPodAutoscaler"}:                              {"kubernetes_horizontal_pod_autoscaler_v2", "kubernetes_horizontal_pod_autoscaler"},
	{group: "autoscaling", version: "v2beta2", kind: "HorizontalPodAutoscaler"}:                         {"kubernetes_horizontal_pod_autoscaler_v2beta2", "kubernetes_horizontal_pod_autoscaler"},
	{group: "batch", version: "v1", kind: "CronJob"}:                                                    {"kubernetes_cron_job_v1", "kubernetes_cron_job"},
	{group: "batch", version: "v1", kind: "Job"}:                                                        {"kubernetes_job_v1", "kubernetes_job"},
	{group: "batch", version: "v1beta1", kind: "CronJob"}:                                               {"kubernetes_cron_job"},
	{group: "certificates.k8s.io", version: "v1", kind: "CertificateSigningRequest"}:                    {"kubernetes_certificate_signing_request_v1", "kubernetes_certificate_signing_request"},
	{group: "certificates.k8s.io", version: "v1beta1", kind: "CertificateSigningRequest"}:               {"kubernetes_certificate_signing_request"},
	{group: "discovery.k8s.io", version: "v1", kind: "EndpointSlice"}:                                   {"kubernetes_endpoint_slice_v1"},
	{group: "extensions", version: "v1beta1", kind: "DaemonSet"}:                                        {"kubernetes_daemonset"},
	{group: "extensions", version: "v1beta1", kind: "Ingress"}:                                          {"kubernetes_ingress"},
	{group: "rbac.authorization.k8s.io", version: "v1", kind: "ClusterRole"}:                            {"kubernetes_cluster_role_v1", "kubernetes_cluster_role"},
	{group: "rbac.authorization.k8s.io", version: "v1", kind: "ClusterRoleBinding"}:                     {"kubernetes_cluster_role_binding_v1", "kubernetes_cluster_role_binding"},
	{group: "rbac.authorization.k8s.io", version: "v1", kind: "Role"}:                                   {"kubernetes_role_v1", "kubernetes_role"},
	{group: "rbac.authorization.k8s.io", version: "v1", kind: "RoleBinding"}:                            {"kubernetes_role_binding_v1", "kubernetes_role_binding"},
	{group: "networking.k8s.io", version: "v1", kind: "Ingress"}:                                        {"kubernetes_ingress_v1", "kubernetes_ingress"},
	{group: "networking.k8s.io", version: "v1", kind: "IngressClass"}:                                   {"kubernetes_ingress_class_v1", "kubernetes_ingress_class"},
	{group: "networking.k8s.io", version: "v1", kind: "NetworkPolicy"}:                                  {"kubernetes_network_policy_v1", "kubernetes_network_policy"},
	{group: "networking.k8s.io", version: "v1beta1", kind: "Ingress"}:                                   {"kubernetes_ingress"},
	{group: "networking.k8s.io", version: "v1beta1", kind: "IngressClass"}:                              {"kubernetes_ingress_class"},
	{group: "networking.k8s.io", version: "v1beta1", kind: "NetworkPolicy"}:                             {"kubernetes_network_policy"},
	{group: "node.k8s.io", version: "v1", kind: "RuntimeClass"}:                                         {"kubernetes_runtime_class_v1"},
	{group: "policy", version: "v1", kind: "PodDisruptionBudget"}:                                       {"kubernetes_pod_disruption_budget_v1", "kubernetes_pod_disruption_budget"},
	{group: "policy", version: "v1beta1", kind: "PodDisruptionBudget"}:                                  {"kubernetes_pod_disruption_budget"},
	{group: "policy", version: "v1beta1", kind: "PodSecurityPolicy"}:                                    {"kubernetes_pod_security_policy"},
	{group: "scheduling.k8s.io", version: "v1", kind: "PriorityClass"}:                                  {"kubernetes_priority_class_v1", "kubernetes_priority_class"},
	{group: "scheduling.k8s.io", version: "v1beta1", kind: "PriorityClass"}:                             {"kubernetes_priority_class"},
	{group: "storage.k8s.io", version: "v1", kind: "CSIDriver"}:                                         {"kubernetes_csi_driver_v1", "kubernetes_csi_driver"},
	{group: "storage.k8s.io", version: "v1beta1", kind: "CSIDriver"}:                                    {"kubernetes_csi_driver"},
	{group: "storage.k8s.io", version: "v1", kind: "StorageClass"}:                                      {"kubernetes_storage_class_v1", "kubernetes_storage_class"},
	{version: "v1", kind: "ConfigMap"}:                                                                  {"kubernetes_config_map_v1", "kubernetes_config_map"},
	{version: "v1", kind: "Endpoints"}:                                                                  {"kubernetes_endpoints_v1", "kubernetes_endpoints"},
	{version: "v1", kind: "LimitRange"}:                                                                 {"kubernetes_limit_range_v1", "kubernetes_limit_range"},
	{version: "v1", kind: "Namespace"}:                                                                  {"kubernetes_namespace_v1", "kubernetes_namespace"},
	{version: "v1", kind: "PersistentVolume"}:                                                           {"kubernetes_persistent_volume_v1", "kubernetes_persistent_volume"},
	{version: "v1", kind: "PersistentVolumeClaim"}:                                                      {"kubernetes_persistent_volume_claim_v1", "kubernetes_persistent_volume_claim"},
	{version: "v1", kind: "Pod"}:                                                                        {"kubernetes_pod_v1", "kubernetes_pod"},
	{version: "v1", kind: "ReplicationController"}:                                                      {"kubernetes_replication_controller_v1", "kubernetes_replication_controller"},
	{version: "v1", kind: "ResourceQuota"}:                                                              {"kubernetes_resource_quota_v1", "kubernetes_resource_quota"},
	{version: "v1", kind: "Secret"}:                                                                     {"kubernetes_secret_v1", "kubernetes_secret"},
	{version: "v1", kind: "Service"}:                                                                    {"kubernetes_service_v1", "kubernetes_service"},
	{version: "v1", kind: "ServiceAccount"}:                                                             {"kubernetes_service_account_v1", "kubernetes_service_account"},
	{group: "admissionregistration.k8s.io", version: "v1", kind: "MutatingWebhookConfiguration"}:        {"kubernetes_mutating_webhook_configuration_v1", "kubernetes_mutating_webhook_configuration"},
	{group: "admissionregistration.k8s.io", version: "v1", kind: "ValidatingAdmissionPolicy"}:           {"kubernetes_validating_admission_policy_v1"},
	{group: "admissionregistration.k8s.io", version: "v1", kind: "ValidatingWebhookConfiguration"}:      {"kubernetes_validating_webhook_configuration_v1", "kubernetes_validating_webhook_configuration"},
	{group: "admissionregistration.k8s.io", version: "v1beta1", kind: "MutatingWebhookConfiguration"}:   {"kubernetes_mutating_webhook_configuration"},
	{group: "admissionregistration.k8s.io", version: "v1beta1", kind: "ValidatingWebhookConfiguration"}: {"kubernetes_validating_webhook_configuration"},
}

// Dynamic imports are limited to provider resources that are importable but not
// exposed through kubernetes.Interface in the pinned client-go version.
var dynamicClientResources = map[kubernetesResourceID]struct{}{
	{group: "apiregistration.k8s.io", version: "v1", kind: "APIService"}:        {},
	{group: "apiregistration.k8s.io", version: "v1beta1", kind: "APIService"}:   {},
	{group: "autoscaling", version: "v2beta2", kind: "HorizontalPodAutoscaler"}: {},
	{group: "policy", version: "v1beta1", kind: "PodSecurityPolicy"}:            {},
}

func extractClientSetFuncGroupName(group, version string) string {
	v := strings.Title(version)
	if len(group) > 0 {
		return strings.Title(strings.Split(group, ".")[0]) + v
	}
	return "Core" + v
}

func extractClientSetFuncTypeName(kind string) string {
	if kind == "Endpoints" {
		return kind
	}

	switch string(kind[len(kind)-1]) {
	case "s":
		return kind + "es"
	case "y":
		return strings.TrimSuffix(kind, "y") + "ies"
	}
	return kind + "s"
}

func extractTfResourceName(kind string) string {
	return "kubernetes_" + strcase.ToSnake(kind)
}

func terraformResourceNameCandidates(group, version, kind string) []string {
	candidates := append([]string{}, preferredTerraformResourceNames[kubernetesResourceID{group: group, version: version, kind: kind}]...)
	defaultName := extractTfResourceName(kind)
	for _, name := range candidates {
		if name == defaultName {
			return candidates
		}
	}
	return append(candidates, defaultName)
}

func selectTerraformResourceName(group, version, kind string, hasResourceType func(string) bool) (string, bool) {
	for _, name := range terraformResourceNameCandidates(group, version, kind) {
		if hasResourceType(name) {
			return name, true
		}
	}
	return "", false
}

func supportsDynamicClientResource(group, version, kind string) bool {
	_, ok := dynamicClientResources[kubernetesResourceID{group: group, version: version, kind: kind}]
	return ok
}

func supportsTypedClientResource(clientset kubernetes.Interface, group, version, kind string) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()

	groupMethod := reflect.ValueOf(clientset).MethodByName(extractClientSetFuncGroupName(group, version))
	if !groupMethod.IsValid() {
		return false
	}

	groupValues := groupMethod.Call([]reflect.Value{})
	if len(groupValues) == 0 || !groupValues[0].IsValid() {
		return false
	}

	resourceMethod := groupValues[0].MethodByName(extractClientSetFuncTypeName(kind))
	return resourceMethod.IsValid()
}
