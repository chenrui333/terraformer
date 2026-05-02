// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package kubernetes

import (
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
	"k8s.io/client-go/kubernetes"
)

var preferredTerraformResourceNames = map[string][]string{
	"CronJob":             {"kubernetes_cron_job_v1", "kubernetes_cron_job"},
	"DaemonSet":           {"kubernetes_daemon_set_v1", "kubernetes_daemonset"},
	"EndpointSlice":       {"kubernetes_endpoint_slice_v1"},
	"Ingress":             {"kubernetes_ingress_v1", "kubernetes_ingress"},
	"IngressClass":        {"kubernetes_ingress_class_v1", "kubernetes_ingress_class"},
	"Job":                 {"kubernetes_job_v1", "kubernetes_job"},
	"NetworkPolicy":       {"kubernetes_network_policy_v1", "kubernetes_network_policy"},
	"PodDisruptionBudget": {"kubernetes_pod_disruption_budget_v1", "kubernetes_pod_disruption_budget"},
	"RuntimeClass":        {"kubernetes_runtime_class_v1"},
}

func extractClientSetFuncGroupName(group, version string) string {
	v := strings.Title(version)
	if len(group) > 0 {
		return strings.Title(strings.Split(group, ".")[0]) + v
	}
	return "Core" + v
}

func extractClientSetFuncTypeName(kind string) string {
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

func terraformResourceNameCandidates(kind string) []string {
	candidates := append([]string{}, preferredTerraformResourceNames[kind]...)
	defaultName := extractTfResourceName(kind)
	for _, name := range candidates {
		if name == defaultName {
			return candidates
		}
	}
	return append(candidates, defaultName)
}

func selectTerraformResourceName(kind string, hasResourceType func(string) bool) (string, bool) {
	for _, name := range terraformResourceNameCandidates(kind) {
		if hasResourceType(name) {
			return name, true
		}
	}
	return "", false
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
