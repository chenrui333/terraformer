// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestTerraformResourceNameCandidates(t *testing.T) {
	tests := []struct {
		name    string
		group   string
		version string
		kind    string
		want    []string
	}{
		{
			name:    "prefers service v1 name for core v1 resource",
			version: "v1",
			kind:    "Service",
			want:    []string{"kubernetes_service_v1", "kubernetes_service"},
		},
		{
			name:    "prefers modern daemon set name before legacy provider spelling for apps v1",
			group:   "apps",
			version: "v1",
			kind:    "DaemonSet",
			want:    []string{"kubernetes_daemon_set_v1", "kubernetes_daemonset", "kubernetes_daemon_set"},
		},
		{
			name:    "prefers legacy daemonset name for extensions beta",
			group:   "extensions",
			version: "v1beta1",
			kind:    "DaemonSet",
			want:    []string{"kubernetes_daemonset", "kubernetes_daemon_set"},
		},
		{
			name:    "prefers legacy ingress name for networking beta",
			group:   "networking.k8s.io",
			version: "v1beta1",
			kind:    "Ingress",
			want:    []string{"kubernetes_ingress"},
		},
		{
			name:    "prefers legacy pod disruption budget name for policy beta",
			group:   "policy",
			version: "v1beta1",
			kind:    "PodDisruptionBudget",
			want:    []string{"kubernetes_pod_disruption_budget"},
		},
		{
			name:    "uses legacy pod security policy name for policy beta",
			group:   "policy",
			version: "v1beta1",
			kind:    "PodSecurityPolicy",
			want:    []string{"kubernetes_pod_security_policy"},
		},
		{
			name:    "prefers autoscaling v2 hpa name",
			group:   "autoscaling",
			version: "v2",
			kind:    "HorizontalPodAutoscaler",
			want:    []string{"kubernetes_horizontal_pod_autoscaler_v2", "kubernetes_horizontal_pod_autoscaler"},
		},
		{
			name:    "prefers autoscaling v2beta2 hpa name",
			group:   "autoscaling",
			version: "v2beta2",
			kind:    "HorizontalPodAutoscaler",
			want:    []string{"kubernetes_horizontal_pod_autoscaler_v2beta2", "kubernetes_horizontal_pod_autoscaler"},
		},
		{
			name:    "prefers certificate signing request v1 name",
			group:   "certificates.k8s.io",
			version: "v1",
			kind:    "CertificateSigningRequest",
			want:    []string{"kubernetes_certificate_signing_request_v1", "kubernetes_certificate_signing_request"},
		},
		{
			name:    "prefers default service account v1 name",
			version: "v1",
			kind:    "DefaultServiceAccount",
			want:    []string{"kubernetes_default_service_account_v1", "kubernetes_default_service_account"},
		},
		{
			name:    "prefers csi driver v1 name",
			group:   "storage.k8s.io",
			version: "v1",
			kind:    "CSIDriver",
			want:    []string{"kubernetes_csi_driver_v1", "kubernetes_csi_driver"},
		},
		{
			name:    "prefers validating admission policy v1 name",
			group:   "admissionregistration.k8s.io",
			version: "v1",
			kind:    "ValidatingAdmissionPolicy",
			want:    []string{"kubernetes_validating_admission_policy_v1", "kubernetes_validating_admission_policy"},
		},
		{
			name:    "uses v1-only endpoint slice name",
			group:   "discovery.k8s.io",
			version: "v1",
			kind:    "EndpointSlice",
			want:    []string{"kubernetes_endpoint_slice_v1", "kubernetes_endpoint_slice"},
		},
		{
			name:    "uses v1-only runtime class name",
			group:   "node.k8s.io",
			version: "v1",
			kind:    "RuntimeClass",
			want:    []string{"kubernetes_runtime_class_v1", "kubernetes_runtime_class"},
		},
		{
			name:    "prefers api service v1 name",
			group:   "apiregistration.k8s.io",
			version: "v1",
			kind:    "APIService",
			want:    []string{"kubernetes_api_service_v1", "kubernetes_api_service"},
		},
		{
			name:    "prefers legacy api service name for beta API",
			group:   "apiregistration.k8s.io",
			version: "v1beta1",
			kind:    "APIService",
			want:    []string{"kubernetes_api_service"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := terraformResourceNameCandidates(tt.group, tt.version, tt.kind)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("terraformResourceNameCandidates(%q, %q, %q) = %#v, want %#v", tt.group, tt.version, tt.kind, got, tt.want)
			}
		})
	}
}

func TestSelectTerraformResourceName(t *testing.T) {
	tests := []struct {
		name           string
		group          string
		version        string
		kind           string
		supportedTypes map[string]struct{}
		want           string
		wantOK         bool
	}{
		{
			name:    "selects default name",
			version: "v1",
			kind:    "Service",
			supportedTypes: map[string]struct{}{
				"kubernetes_service": {},
			},
			want:   "kubernetes_service",
			wantOK: true,
		},
		{
			name:    "prefers modern mapped name for v1 API",
			group:   "apps",
			version: "v1",
			kind:    "DaemonSet",
			supportedTypes: map[string]struct{}{
				"kubernetes_daemonset":     {},
				"kubernetes_daemon_set_v1": {},
			},
			want:   "kubernetes_daemon_set_v1",
			wantOK: true,
		},
		{
			name:    "falls back to legacy mapped name for v1 API",
			group:   "apps",
			version: "v1",
			kind:    "DaemonSet",
			supportedTypes: map[string]struct{}{
				"kubernetes_daemonset": {},
			},
			want:   "kubernetes_daemonset",
			wantOK: true,
		},
		{
			name:    "prefers legacy mapped name for beta API even when v1 type exists",
			group:   "networking.k8s.io",
			version: "v1beta1",
			kind:    "Ingress",
			supportedTypes: map[string]struct{}{
				"kubernetes_ingress":    {},
				"kubernetes_ingress_v1": {},
			},
			want:   "kubernetes_ingress",
			wantOK: true,
		},
		{
			name:    "selects v1 mapped name for v1 API",
			group:   "networking.k8s.io",
			version: "v1",
			kind:    "Ingress",
			supportedTypes: map[string]struct{}{
				"kubernetes_ingress":    {},
				"kubernetes_ingress_v1": {},
			},
			want:   "kubernetes_ingress_v1",
			wantOK: true,
		},
		{
			name:    "prefers legacy pod disruption budget for beta API",
			group:   "policy",
			version: "v1beta1",
			kind:    "PodDisruptionBudget",
			supportedTypes: map[string]struct{}{
				"kubernetes_pod_disruption_budget":    {},
				"kubernetes_pod_disruption_budget_v1": {},
			},
			want:   "kubernetes_pod_disruption_budget",
			wantOK: true,
		},
		{
			name:    "selects v1 pod disruption budget for v1 API",
			group:   "policy",
			version: "v1",
			kind:    "PodDisruptionBudget",
			supportedTypes: map[string]struct{}{
				"kubernetes_pod_disruption_budget":    {},
				"kubernetes_pod_disruption_budget_v1": {},
			},
			want:   "kubernetes_pod_disruption_budget_v1",
			wantOK: true,
		},
		{
			name:    "selects pod security policy for policy beta API",
			group:   "policy",
			version: "v1beta1",
			kind:    "PodSecurityPolicy",
			supportedTypes: map[string]struct{}{
				"kubernetes_pod_security_policy": {},
			},
			want:   "kubernetes_pod_security_policy",
			wantOK: true,
		},
		{
			name:    "prefers legacy cron job for beta API",
			group:   "batch",
			version: "v1beta1",
			kind:    "CronJob",
			supportedTypes: map[string]struct{}{
				"kubernetes_cron_job":    {},
				"kubernetes_cron_job_v1": {},
			},
			want:   "kubernetes_cron_job",
			wantOK: true,
		},
		{
			name:    "selects autoscaling v2 hpa for v2 API",
			group:   "autoscaling",
			version: "v2",
			kind:    "HorizontalPodAutoscaler",
			supportedTypes: map[string]struct{}{
				"kubernetes_horizontal_pod_autoscaler":    {},
				"kubernetes_horizontal_pod_autoscaler_v2": {},
			},
			want:   "kubernetes_horizontal_pod_autoscaler_v2",
			wantOK: true,
		},
		{
			name:    "selects autoscaling v2beta2 hpa for v2beta2 API",
			group:   "autoscaling",
			version: "v2beta2",
			kind:    "HorizontalPodAutoscaler",
			supportedTypes: map[string]struct{}{
				"kubernetes_horizontal_pod_autoscaler":         {},
				"kubernetes_horizontal_pod_autoscaler_v2beta2": {},
			},
			want:   "kubernetes_horizontal_pod_autoscaler_v2beta2",
			wantOK: true,
		},
		{
			name:    "selects certificate signing request v1 for certificates v1 API",
			group:   "certificates.k8s.io",
			version: "v1",
			kind:    "CertificateSigningRequest",
			supportedTypes: map[string]struct{}{
				"kubernetes_certificate_signing_request":    {},
				"kubernetes_certificate_signing_request_v1": {},
			},
			want:   "kubernetes_certificate_signing_request_v1",
			wantOK: true,
		},
		{
			name:    "selects default service account v1",
			version: "v1",
			kind:    "DefaultServiceAccount",
			supportedTypes: map[string]struct{}{
				"kubernetes_default_service_account":    {},
				"kubernetes_default_service_account_v1": {},
			},
			want:   "kubernetes_default_service_account_v1",
			wantOK: true,
		},
		{
			name:    "prefers legacy certificate signing request for certificates beta API",
			group:   "certificates.k8s.io",
			version: "v1beta1",
			kind:    "CertificateSigningRequest",
			supportedTypes: map[string]struct{}{
				"kubernetes_certificate_signing_request":    {},
				"kubernetes_certificate_signing_request_v1": {},
			},
			want:   "kubernetes_certificate_signing_request",
			wantOK: true,
		},
		{
			name:    "selects csi driver v1 for storage v1 API",
			group:   "storage.k8s.io",
			version: "v1",
			kind:    "CSIDriver",
			supportedTypes: map[string]struct{}{
				"kubernetes_csi_driver":    {},
				"kubernetes_csi_driver_v1": {},
			},
			want:   "kubernetes_csi_driver_v1",
			wantOK: true,
		},
		{
			name:    "prefers legacy csi driver for storage beta API",
			group:   "storage.k8s.io",
			version: "v1beta1",
			kind:    "CSIDriver",
			supportedTypes: map[string]struct{}{
				"kubernetes_csi_driver":    {},
				"kubernetes_csi_driver_v1": {},
			},
			want:   "kubernetes_csi_driver",
			wantOK: true,
		},
		{
			name:    "selects priority class v1 for scheduling v1 API",
			group:   "scheduling.k8s.io",
			version: "v1",
			kind:    "PriorityClass",
			supportedTypes: map[string]struct{}{
				"kubernetes_priority_class":    {},
				"kubernetes_priority_class_v1": {},
			},
			want:   "kubernetes_priority_class_v1",
			wantOK: true,
		},
		{
			name:    "selects admission webhook v1 for admissionregistration v1 API",
			group:   "admissionregistration.k8s.io",
			version: "v1",
			kind:    "MutatingWebhookConfiguration",
			supportedTypes: map[string]struct{}{
				"kubernetes_mutating_webhook_configuration":    {},
				"kubernetes_mutating_webhook_configuration_v1": {},
			},
			want:   "kubernetes_mutating_webhook_configuration_v1",
			wantOK: true,
		},
		{
			name:    "selects validating admission policy v1",
			group:   "admissionregistration.k8s.io",
			version: "v1",
			kind:    "ValidatingAdmissionPolicy",
			supportedTypes: map[string]struct{}{
				"kubernetes_validating_admission_policy_v1": {},
			},
			want:   "kubernetes_validating_admission_policy_v1",
			wantOK: true,
		},
		{
			name:    "selects api service v1 for apiregistration v1 API",
			group:   "apiregistration.k8s.io",
			version: "v1",
			kind:    "APIService",
			supportedTypes: map[string]struct{}{
				"kubernetes_api_service":    {},
				"kubernetes_api_service_v1": {},
			},
			want:   "kubernetes_api_service_v1",
			wantOK: true,
		},
		{
			name:    "prefers legacy api service for apiregistration beta API",
			group:   "apiregistration.k8s.io",
			version: "v1beta1",
			kind:    "APIService",
			supportedTypes: map[string]struct{}{
				"kubernetes_api_service":    {},
				"kubernetes_api_service_v1": {},
			},
			want:   "kubernetes_api_service",
			wantOK: true,
		},
		{
			name:    "returns false when provider has no matching resource",
			group:   "discovery.k8s.io",
			version: "v1",
			kind:    "EndpointSlice",
			supportedTypes: map[string]struct{}{
				"kubernetes_service": {},
			},
			want:   "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := selectTerraformResourceName(tt.group, tt.version, tt.kind, func(name string) bool {
				_, exists := tt.supportedTypes[name]
				return exists
			})
			if ok != tt.wantOK {
				t.Fatalf("selectTerraformResourceName(%q, %q, %q) ok = %t, want %t", tt.group, tt.version, tt.kind, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("selectTerraformResourceName(%q, %q, %q) = %q, want %q", tt.group, tt.version, tt.kind, got, tt.want)
			}
		})
	}
}

func TestSelectImportResourceName(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	manageableVerbs := []string{"create", "delete", "get", "list", "patch", "update"}
	tests := []struct {
		name           string
		group          string
		version        string
		resource       metav1.APIResource
		supportedTypes map[string]struct{}
		want           string
		wantDynamic    bool
		wantOK         bool
	}{
		{
			name:    "selects first-class typed resource",
			version: "v1",
			resource: metav1.APIResource{
				Name:  "services",
				Kind:  "Service",
				Verbs: manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				"kubernetes_service_v1": {},
			},
			want:   "kubernetes_service_v1",
			wantOK: true,
		},
		{
			name:    "selects explicit dynamic first-class resource",
			group:   "apiregistration.k8s.io",
			version: "v1",
			resource: metav1.APIResource{
				Name:  "apiservices",
				Kind:  "APIService",
				Verbs: manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				"kubernetes_api_service_v1": {},
			},
			want:        "kubernetes_api_service_v1",
			wantDynamic: true,
			wantOK:      true,
		},
		{
			name:    "falls back to manifest for untyped API extension",
			group:   "example.com",
			version: "v1",
			resource: metav1.APIResource{
				Name:  "widgets",
				Kind:  "Widget",
				Verbs: manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			want:        manifestTerraformResourceName,
			wantDynamic: true,
			wantOK:      true,
		},
		{
			name:    "skips native typed resource without first-class provider type",
			version: "v1",
			resource: metav1.APIResource{
				Name:  "nodes",
				Kind:  "Node",
				Verbs: manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			wantOK: false,
		},
		{
			name:    "skips untyped resource without manifest provider type",
			group:   "example.com",
			version: "v1",
			resource: metav1.APIResource{
				Name:  "widgets",
				Kind:  "Widget",
				Verbs: manageableVerbs,
			},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, dynamic, ok := selectImportResourceName(clientset, tt.group, tt.version, tt.resource, func(name string) bool {
				_, exists := tt.supportedTypes[name]
				return exists
			})
			if ok != tt.wantOK {
				t.Fatalf("selectImportResourceName() ok = %t, want %t", ok, tt.wantOK)
			}
			if dynamic != tt.wantDynamic {
				t.Fatalf("selectImportResourceName() dynamic = %t, want %t", dynamic, tt.wantDynamic)
			}
			if got != tt.want {
				t.Fatalf("selectImportResourceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSelectTerraformResourceNameStableV1Aliases(t *testing.T) {
	tests := []struct {
		name       string
		group      string
		kind       string
		modernType string
		legacyType string
	}{
		{name: "config map", kind: "ConfigMap", modernType: "kubernetes_config_map_v1", legacyType: "kubernetes_config_map"},
		{name: "deployment", group: "apps", kind: "Deployment", modernType: "kubernetes_deployment_v1", legacyType: "kubernetes_deployment"},
		{name: "default service account", kind: "DefaultServiceAccount", modernType: "kubernetes_default_service_account_v1", legacyType: "kubernetes_default_service_account"},
		{name: "endpoints", kind: "Endpoints", modernType: "kubernetes_endpoints_v1", legacyType: "kubernetes_endpoints"},
		{name: "limit range", kind: "LimitRange", modernType: "kubernetes_limit_range_v1", legacyType: "kubernetes_limit_range"},
		{name: "namespace", kind: "Namespace", modernType: "kubernetes_namespace_v1", legacyType: "kubernetes_namespace"},
		{name: "persistent volume", kind: "PersistentVolume", modernType: "kubernetes_persistent_volume_v1", legacyType: "kubernetes_persistent_volume"},
		{name: "persistent volume claim", kind: "PersistentVolumeClaim", modernType: "kubernetes_persistent_volume_claim_v1", legacyType: "kubernetes_persistent_volume_claim"},
		{name: "pod", kind: "Pod", modernType: "kubernetes_pod_v1", legacyType: "kubernetes_pod"},
		{name: "replication controller", kind: "ReplicationController", modernType: "kubernetes_replication_controller_v1", legacyType: "kubernetes_replication_controller"},
		{name: "resource quota", kind: "ResourceQuota", modernType: "kubernetes_resource_quota_v1", legacyType: "kubernetes_resource_quota"},
		{name: "secret", kind: "Secret", modernType: "kubernetes_secret_v1", legacyType: "kubernetes_secret"},
		{name: "service", kind: "Service", modernType: "kubernetes_service_v1", legacyType: "kubernetes_service"},
		{name: "service account", kind: "ServiceAccount", modernType: "kubernetes_service_account_v1", legacyType: "kubernetes_service_account"},
		{name: "stateful set", group: "apps", kind: "StatefulSet", modernType: "kubernetes_stateful_set_v1", legacyType: "kubernetes_stateful_set"},
		{name: "storage class", group: "storage.k8s.io", kind: "StorageClass", modernType: "kubernetes_storage_class_v1", legacyType: "kubernetes_storage_class"},
		{name: "cluster role", group: "rbac.authorization.k8s.io", kind: "ClusterRole", modernType: "kubernetes_cluster_role_v1", legacyType: "kubernetes_cluster_role"},
		{name: "cluster role binding", group: "rbac.authorization.k8s.io", kind: "ClusterRoleBinding", modernType: "kubernetes_cluster_role_binding_v1", legacyType: "kubernetes_cluster_role_binding"},
		{name: "role", group: "rbac.authorization.k8s.io", kind: "Role", modernType: "kubernetes_role_v1", legacyType: "kubernetes_role"},
		{name: "role binding", group: "rbac.authorization.k8s.io", kind: "RoleBinding", modernType: "kubernetes_role_binding_v1", legacyType: "kubernetes_role_binding"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			supportedTypes := map[string]struct{}{
				tt.legacyType: {},
				tt.modernType: {},
			}
			got, ok := selectTerraformResourceName(tt.group, "v1", tt.kind, func(name string) bool {
				_, exists := supportedTypes[name]
				return exists
			})
			if !ok {
				t.Fatalf("selectTerraformResourceName(%q, %q, %q) did not find a type", tt.group, "v1", tt.kind)
			}
			if got != tt.modernType {
				t.Fatalf("selectTerraformResourceName(%q, %q, %q) = %q, want %q", tt.group, "v1", tt.kind, got, tt.modernType)
			}

			got, ok = selectTerraformResourceName(tt.group, "v1", tt.kind, func(name string) bool {
				return name == tt.legacyType
			})
			if !ok {
				t.Fatalf("selectTerraformResourceName(%q, %q, %q) did not find legacy fallback", tt.group, "v1", tt.kind)
			}
			if got != tt.legacyType {
				t.Fatalf("selectTerraformResourceName(%q, %q, %q) fallback = %q, want %q", tt.group, "v1", tt.kind, got, tt.legacyType)
			}
		})
	}
}

func TestSupportsDynamicClientResource(t *testing.T) {
	tests := []struct {
		name    string
		group   string
		version string
		kind    string
		want    bool
	}{
		{name: "api service v1", group: "apiregistration.k8s.io", version: "v1", kind: "APIService", want: true},
		{name: "api service beta", group: "apiregistration.k8s.io", version: "v1beta1", kind: "APIService", want: true},
		{name: "autoscaling v2beta2 hpa", group: "autoscaling", version: "v2beta2", kind: "HorizontalPodAutoscaler", want: true},
		{name: "pod security policy beta", group: "policy", version: "v1beta1", kind: "PodSecurityPolicy", want: true},
		{name: "typed core resource", version: "v1", kind: "Service", want: false},
		{name: "unknown resource", group: "example.com", version: "v1", kind: "Widget", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := supportsDynamicClientResource(tt.group, tt.version, tt.kind)
			if got != tt.want {
				t.Fatalf("supportsDynamicClientResource(%q, %q, %q) = %t, want %t", tt.group, tt.version, tt.kind, got, tt.want)
			}
		})
	}
}

func TestSupportsManifestResource(t *testing.T) {
	manageableVerbs := []string{"create", "delete", "get", "list", "patch", "update"}
	tests := []struct {
		name           string
		resource       metav1.APIResource
		supportedTypes map[string]struct{}
		want           bool
	}{
		{
			name: "supports manageable custom resource",
			resource: metav1.APIResource{
				Name:  "widgets",
				Kind:  "Widget",
				Verbs: manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			want: true,
		},
		{
			name: "requires manifest provider type",
			resource: metav1.APIResource{
				Name:  "widgets",
				Kind:  "Widget",
				Verbs: manageableVerbs,
			},
			want: false,
		},
		{
			name: "rejects subresources",
			resource: metav1.APIResource{
				Name:  "widgets/status",
				Kind:  "Widget",
				Verbs: manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			want: false,
		},
		{
			name: "requires manageable verbs",
			resource: metav1.APIResource{
				Name:  "podmetrics",
				Kind:  "PodMetrics",
				Verbs: []string{"get", "list"},
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			want: false,
		},
		{
			name: "requires kind",
			resource: metav1.APIResource{
				Name:  "widgets",
				Verbs: manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := supportsManifestResource(tt.resource, func(name string) bool {
				_, exists := tt.supportedTypes[name]
				return exists
			})
			if got != tt.want {
				t.Fatalf("supportsManifestResource() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestSupportsTypedClientResource(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	tests := []struct {
		name    string
		group   string
		version string
		kind    string
		want    bool
	}{
		{name: "core service", version: "v1", kind: "Service", want: true},
		{name: "core endpoints", version: "v1", kind: "Endpoints", want: true},
		{name: "apps daemon set", group: "apps", version: "v1", kind: "DaemonSet", want: true},
		{name: "autoscaling hpa", group: "autoscaling", version: "v2", kind: "HorizontalPodAutoscaler", want: true},
		{name: "autoscaling v2beta2 hpa is not exposed by kubernetes clientset", group: "autoscaling", version: "v2beta2", kind: "HorizontalPodAutoscaler", want: false},
		{name: "certificate signing request", group: "certificates.k8s.io", version: "v1", kind: "CertificateSigningRequest", want: true},
		{name: "storage csi driver", group: "storage.k8s.io", version: "v1", kind: "CSIDriver", want: true},
		{name: "scheduling priority class", group: "scheduling.k8s.io", version: "v1", kind: "PriorityClass", want: true},
		{name: "admission validating policy", group: "admissionregistration.k8s.io", version: "v1", kind: "ValidatingAdmissionPolicy", want: true},
		{name: "admission webhook", group: "admissionregistration.k8s.io", version: "v1", kind: "MutatingWebhookConfiguration", want: true},
		{name: "discovery endpoint slice", group: "discovery.k8s.io", version: "v1", kind: "EndpointSlice", want: true},
		{name: "node runtime class", group: "node.k8s.io", version: "v1", kind: "RuntimeClass", want: true},
		{name: "api service is not exposed by kubernetes clientset", group: "apiregistration.k8s.io", version: "v1", kind: "APIService", want: false},
		{name: "pod security policy is not exposed by kubernetes clientset", group: "policy", version: "v1beta1", kind: "PodSecurityPolicy", want: false},
		{name: "unknown group", group: "example.com", version: "v1", kind: "Widget", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := supportsTypedClientResource(clientset, tt.group, tt.version, tt.kind)
			if got != tt.want {
				t.Fatalf("supportsTypedClientResource(%q, %q, %q) = %t, want %t", tt.group, tt.version, tt.kind, got, tt.want)
			}
		})
	}
}
