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
			name:    "falls back to manifest for colliding custom kind",
			group:   "serving.knative.dev",
			version: "v1",
			resource: metav1.APIResource{
				Name:  "services",
				Kind:  "Service",
				Verbs: manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				"kubernetes_service":          {},
				manifestTerraformResourceName: {},
			},
			want:        manifestTerraformResourceName,
			wantDynamic: true,
			wantOK:      true,
		},
		{
			name:    "falls back to manifest for custom group sharing native prefix",
			group:   "apps.example.com",
			version: "v1",
			resource: metav1.APIResource{
				Name:  "deployments",
				Kind:  "Deployment",
				Verbs: manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				"kubernetes_deployment":       {},
				manifestTerraformResourceName: {},
			},
			want:        manifestTerraformResourceName,
			wantDynamic: true,
			wantOK:      true,
		},
		{
			name:    "skips unallowlisted native API group instead of generic manifest fallback",
			group:   "events.k8s.io",
			version: "v1beta1",
			resource: metav1.APIResource{
				Name:  "events",
				Kind:  "Event",
				Verbs: manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			wantOK: false,
		},
		{
			name:    "falls back to manifest for native admission policy binding",
			group:   "admissionregistration.k8s.io",
			version: "v1",
			resource: metav1.APIResource{
				Name:  "validatingadmissionpolicybindings",
				Kind:  "ValidatingAdmissionPolicyBinding",
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
			name:    "falls back to manifest for standalone replica set",
			group:   "apps",
			version: "v1",
			resource: metav1.APIResource{
				Name:       "replicasets",
				Kind:       "ReplicaSet",
				Namespaced: true,
				Verbs:      manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			want:        manifestTerraformResourceName,
			wantDynamic: true,
			wantOK:      true,
		},
		{
			name:    "falls back to manifest for pod template",
			version: "v1",
			resource: metav1.APIResource{
				Name:       "podtemplates",
				Kind:       "PodTemplate",
				Namespaced: true,
				Verbs:      manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			want:        manifestTerraformResourceName,
			wantDynamic: true,
			wantOK:      true,
		},
		{
			name:    "falls back to manifest for flow control resource",
			group:   "flowcontrol.apiserver.k8s.io",
			version: "v1",
			resource: metav1.APIResource{
				Name:  "flowschemas",
				Kind:  "FlowSchema",
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
			name:    "falls back to manifest for service cidr",
			group:   "networking.k8s.io",
			version: "v1",
			resource: metav1.APIResource{
				Name:  "servicecidrs",
				Kind:  "ServiceCIDR",
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
			name:    "falls back to manifest for beta service cidr",
			group:   "networking.k8s.io",
			version: "v1beta1",
			resource: metav1.APIResource{
				Name:  "servicecidrs",
				Kind:  "ServiceCIDR",
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
			name:    "falls back to manifest for dynamic resource allocation resource",
			group:   "resource.k8s.io",
			version: "v1",
			resource: metav1.APIResource{
				Name:       "resourceclaims",
				Kind:       "ResourceClaim",
				Namespaced: true,
				Verbs:      manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			want:        manifestTerraformResourceName,
			wantDynamic: true,
			wantOK:      true,
		},
		{
			name:    "falls back to manifest for alpha dynamic resource allocation resource",
			group:   "resource.k8s.io",
			version: "v1alpha3",
			resource: metav1.APIResource{
				Name:       "resourceclaims",
				Kind:       "ResourceClaim",
				Namespaced: true,
				Verbs:      manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			want:        manifestTerraformResourceName,
			wantDynamic: true,
			wantOK:      true,
		},
		{
			name:    "skips legacy dynamic resource allocation resource outside supported policy",
			group:   "resource.k8s.io",
			version: "v1alpha2",
			resource: metav1.APIResource{
				Name:  "resourceclasses",
				Kind:  "ResourceClass",
				Verbs: manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			wantOK: false,
		},
		{
			name:    "falls back to manifest for cluster trust bundle",
			group:   "certificates.k8s.io",
			version: "v1beta1",
			resource: metav1.APIResource{
				Name:  "clustertrustbundles",
				Kind:  "ClusterTrustBundle",
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
			name:    "skips kubelet-generated pod certificate requests",
			group:   "certificates.k8s.io",
			version: "v1beta1",
			resource: metav1.APIResource{
				Name:       "podcertificaterequests",
				Kind:       "PodCertificateRequest",
				Namespaced: true,
				Verbs:      manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			wantOK: false,
		},
		{
			name:    "skips kubelet-generated alpha pod certificate requests",
			group:   "certificates.k8s.io",
			version: "v1alpha1",
			resource: metav1.APIResource{
				Name:       "podcertificaterequests",
				Kind:       "PodCertificateRequest",
				Namespaced: true,
				Verbs:      manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			wantOK: false,
		},
		{
			name:    "skips driver-generated alpha resource slices",
			group:   "resource.k8s.io",
			version: "v1alpha3",
			resource: metav1.APIResource{
				Name:  "resourceslices",
				Kind:  "ResourceSlice",
				Verbs: manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			wantOK: false,
		},
		{
			name:    "skips historical alpha resource slices",
			group:   "resource.k8s.io",
			version: "v1alpha2",
			resource: metav1.APIResource{
				Name:  "resourceslices",
				Kind:  "ResourceSlice",
				Verbs: manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			wantOK: false,
		},
		{
			name:    "skips historical alpha pod scheduling contexts",
			group:   "resource.k8s.io",
			version: "v1alpha3",
			resource: metav1.APIResource{
				Name:       "podschedulingcontexts",
				Kind:       "PodSchedulingContext",
				Namespaced: true,
				Verbs:      manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			wantOK: false,
		},
		{
			name:    "skips older pod scheduling contexts",
			group:   "resource.k8s.io",
			version: "v1alpha2",
			resource: metav1.APIResource{
				Name:       "podschedulingcontexts",
				Kind:       "PodSchedulingContext",
				Namespaced: true,
				Verbs:      manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			wantOK: false,
		},
		{
			name:    "skips original alpha pod schedulings",
			group:   "resource.k8s.io",
			version: "v1alpha1",
			resource: metav1.APIResource{
				Name:       "podschedulings",
				Kind:       "PodScheduling",
				Namespaced: true,
				Verbs:      manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			wantOK: false,
		},
		{
			name:    "skips allocator-managed beta ip addresses",
			group:   "networking.k8s.io",
			version: "v1beta1",
			resource: metav1.APIResource{
				Name:  "ipaddresses",
				Kind:  "IPAddress",
				Verbs: manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			wantOK: false,
		},
		{
			name:    "skips historical alpha lease candidates",
			group:   "coordination.k8s.io",
			version: "v1alpha1",
			resource: metav1.APIResource{
				Name:       "leasecandidates",
				Kind:       "LeaseCandidate",
				Namespaced: true,
				Verbs:      manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			wantOK: false,
		},
		{
			name:    "falls back to manifest for scheduling pod group",
			group:   "scheduling.k8s.io",
			version: "v1alpha1",
			resource: metav1.APIResource{
				Name:       "podgroups",
				Kind:       "PodGroup",
				Namespaced: true,
				Verbs:      manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			want:        manifestTerraformResourceName,
			wantDynamic: true,
			wantOK:      true,
		},
		{
			name:    "falls back to manifest for scheduling workload",
			group:   "scheduling.k8s.io",
			version: "v1alpha1",
			resource: metav1.APIResource{
				Name:       "workloads",
				Kind:       "Workload",
				Namespaced: true,
				Verbs:      manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			want:        manifestTerraformResourceName,
			wantDynamic: true,
			wantOK:      true,
		},
		{
			name:    "falls back to manifest for preferred scheduling pod group",
			group:   "scheduling.k8s.io",
			version: "v1alpha2",
			resource: metav1.APIResource{
				Name:       "podgroups",
				Kind:       "PodGroup",
				Namespaced: true,
				Verbs:      manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			want:        manifestTerraformResourceName,
			wantDynamic: true,
			wantOK:      true,
		},
		{
			name:    "falls back to manifest for preferred scheduling workload",
			group:   "scheduling.k8s.io",
			version: "v1alpha2",
			resource: metav1.APIResource{
				Name:       "workloads",
				Kind:       "Workload",
				Namespaced: true,
				Verbs:      manageableVerbs,
			},
			supportedTypes: map[string]struct{}{
				manifestTerraformResourceName: {},
			},
			want:        manifestTerraformResourceName,
			wantDynamic: true,
			wantOK:      true,
		},
		{
			name:    "falls back to manifest for storage version migration",
			group:   "storagemigration.k8s.io",
			version: "v1beta1",
			resource: metav1.APIResource{
				Name:  "storageversionmigrations",
				Kind:  "StorageVersionMigration",
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
			name:    "falls back to manifest for alpha storage version migration",
			group:   "storagemigration.k8s.io",
			version: "v1alpha1",
			resource: metav1.APIResource{
				Name:  "storageversionmigrations",
				Kind:  "StorageVersionMigration",
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
			name:    "falls back to manifest for volume attributes class",
			group:   "storage.k8s.io",
			version: "v1",
			resource: metav1.APIResource{
				Name:  "volumeattributesclasses",
				Kind:  "VolumeAttributesClass",
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

func TestSkipsImportResource(t *testing.T) {
	tests := []struct {
		name    string
		group   string
		version string
		kind    string
		want    bool
	}{
		{name: "pod certificate request", group: "certificates.k8s.io", version: "v1beta1", kind: "PodCertificateRequest", want: true},
		{name: "old pod certificate request", group: "certificates.k8s.io", version: "v1alpha1", kind: "PodCertificateRequest", want: true},
		{name: "resource slice", group: "resource.k8s.io", version: "v1", kind: "ResourceSlice", want: true},
		{name: "old resource slice", group: "resource.k8s.io", version: "v1alpha3", kind: "ResourceSlice", want: true},
		{name: "historical resource slice", group: "resource.k8s.io", version: "v1alpha2", kind: "ResourceSlice", want: true},
		{name: "pod scheduling context", group: "resource.k8s.io", version: "v1alpha3", kind: "PodSchedulingContext", want: true},
		{name: "old pod scheduling context", group: "resource.k8s.io", version: "v1alpha2", kind: "PodSchedulingContext", want: true},
		{name: "original pod scheduling", group: "resource.k8s.io", version: "v1alpha1", kind: "PodScheduling", want: true},
		{name: "resource pool status request", group: "resource.k8s.io", version: "v1alpha3", kind: "ResourcePoolStatusRequest", want: true},
		{name: "ip address", group: "networking.k8s.io", version: "v1", kind: "IPAddress", want: true},
		{name: "beta ip address", group: "networking.k8s.io", version: "v1beta1", kind: "IPAddress", want: true},
		{name: "controller revision", group: "apps", version: "v1", kind: "ControllerRevision", want: true},
		{name: "lease candidate", group: "coordination.k8s.io", version: "v1alpha2", kind: "LeaseCandidate", want: true},
		{name: "historical lease candidate", group: "coordination.k8s.io", version: "v1alpha1", kind: "LeaseCandidate", want: true},
		{name: "storage version", group: "internal.apiserver.k8s.io", version: "v1alpha1", kind: "StorageVersion", want: true},
		{name: "csi node", group: "storage.k8s.io", version: "v1", kind: "CSINode", want: true},
		{name: "csi storage capacity", group: "storage.k8s.io", version: "v1alpha1", kind: "CSIStorageCapacity", want: true},
		{name: "volume attachment", group: "storage.k8s.io", version: "v1", kind: "VolumeAttachment", want: true},
		{name: "custom resource is not skipped", group: "example.com", version: "v1", kind: "Widget", want: false},
		{name: "declarative native manifest is not skipped", group: "resource.k8s.io", version: "v1alpha3", kind: "ResourceClaim", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := skipsImportResource(tt.group, tt.version, tt.kind)
			if got != tt.want {
				t.Fatalf("skipsImportResource(%q, %q, %q) = %t, want %t", tt.group, tt.version, tt.kind, got, tt.want)
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

func TestIsNativeAPIGroup(t *testing.T) {
	tests := []struct {
		name  string
		group string
		want  bool
	}{
		{name: "core", want: true},
		{name: "resource API", group: "resource.k8s.io", want: true},
		{name: "custom group sharing native prefix", group: "apps.example.com", want: false},
		{name: "custom group", group: "example.com", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNativeAPIGroup(tt.group)
			if got != tt.want {
				t.Fatalf("isNativeAPIGroup(%q) = %t, want %t", tt.group, got, tt.want)
			}
		})
	}
}

func TestImportSkipPolicyReason(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	manageableVerbs := []string{"create", "delete", "get", "list", "patch", "update"}
	hasManifestType := func(name string) bool {
		return name == manifestTerraformResourceName
	}

	tests := []struct {
		name     string
		group    string
		version  string
		resource metav1.APIResource
		want     string
	}{
		{
			name:    "generated native API",
			group:   "resource.k8s.io",
			version: "v1",
			resource: metav1.APIResource{
				Name:  "resourceslices",
				Kind:  "ResourceSlice",
				Verbs: manageableVerbs,
			},
			want: "runtime/controller-generated native API is not importable as Terraform-managed configuration",
		},
		{
			name:    "legacy native API outside supported manifest policy",
			group:   "resource.k8s.io",
			version: "v1alpha2",
			resource: metav1.APIResource{
				Name:  "resourceclasses",
				Kind:  "ResourceClass",
				Verbs: manageableVerbs,
			},
			want: "native API is outside the explicit manifest import policy",
		},
		{
			name:    "custom resource keeps generic manifest fallback",
			group:   "example.com",
			version: "v1",
			resource: metav1.APIResource{
				Name:  "widgets",
				Kind:  "Widget",
				Verbs: manageableVerbs,
			},
		},
		{
			name:    "allowlisted native manifest resource",
			group:   "resource.k8s.io",
			version: "v1",
			resource: metav1.APIResource{
				Name:       "resourceclaims",
				Kind:       "ResourceClaim",
				Namespaced: true,
				Verbs:      manageableVerbs,
			},
		},
		{
			name:    "native resource without manifest type",
			group:   "resource.k8s.io",
			version: "v1alpha2",
			resource: metav1.APIResource{
				Name:  "resourceclasses",
				Kind:  "ResourceClass",
				Verbs: manageableVerbs,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasResourceType := hasManifestType
			if tt.name == "native resource without manifest type" {
				hasResourceType = func(string) bool { return false }
			}
			got := importSkipPolicyReason(clientset, tt.group, tt.version, tt.resource, hasResourceType)
			if got != tt.want {
				t.Fatalf("importSkipPolicyReason() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSupportsNativeManifestResource(t *testing.T) {
	tests := []struct {
		name    string
		group   string
		version string
		kind    string
		want    bool
	}{
		{name: "mutating admission policy v1", group: "admissionregistration.k8s.io", version: "v1", kind: "MutatingAdmissionPolicy", want: true},
		{name: "validating admission policy binding v1", group: "admissionregistration.k8s.io", version: "v1", kind: "ValidatingAdmissionPolicyBinding", want: true},
		{name: "validating admission policy beta", group: "admissionregistration.k8s.io", version: "v1beta1", kind: "ValidatingAdmissionPolicy", want: true},
		{name: "mutating admission policy binding alpha", group: "admissionregistration.k8s.io", version: "v1alpha1", kind: "MutatingAdmissionPolicyBinding", want: true},
		{name: "standalone replica set", group: "apps", version: "v1", kind: "ReplicaSet", want: true},
		{name: "cluster trust bundle beta", group: "certificates.k8s.io", version: "v1beta1", kind: "ClusterTrustBundle", want: true},
		{name: "flow schema v1", group: "flowcontrol.apiserver.k8s.io", version: "v1", kind: "FlowSchema", want: true},
		{name: "priority level configuration v1beta3", group: "flowcontrol.apiserver.k8s.io", version: "v1beta3", kind: "PriorityLevelConfiguration", want: true},
		{name: "service cidr v1", group: "networking.k8s.io", version: "v1", kind: "ServiceCIDR", want: true},
		{name: "service cidr beta", group: "networking.k8s.io", version: "v1beta1", kind: "ServiceCIDR", want: true},
		{name: "device class v1", group: "resource.k8s.io", version: "v1", kind: "DeviceClass", want: true},
		{name: "resource claim v1beta2", group: "resource.k8s.io", version: "v1beta2", kind: "ResourceClaim", want: true},
		{name: "resource claim template v1beta1", group: "resource.k8s.io", version: "v1beta1", kind: "ResourceClaimTemplate", want: true},
		{name: "device class alpha", group: "resource.k8s.io", version: "v1alpha3", kind: "DeviceClass", want: true},
		{name: "device taint rule alpha", group: "resource.k8s.io", version: "v1alpha3", kind: "DeviceTaintRule", want: true},
		{name: "resource claim alpha", group: "resource.k8s.io", version: "v1alpha3", kind: "ResourceClaim", want: true},
		{name: "resource claim template alpha", group: "resource.k8s.io", version: "v1alpha3", kind: "ResourceClaimTemplate", want: true},
		{name: "legacy resource class is outside supported policy", group: "resource.k8s.io", version: "v1alpha2", kind: "ResourceClass", want: false},
		{name: "scheduling pod group preferred alpha", group: "scheduling.k8s.io", version: "v1alpha2", kind: "PodGroup", want: true},
		{name: "scheduling workload preferred alpha", group: "scheduling.k8s.io", version: "v1alpha2", kind: "Workload", want: true},
		{name: "scheduling pod group alpha", group: "scheduling.k8s.io", version: "v1alpha1", kind: "PodGroup", want: true},
		{name: "scheduling workload alpha", group: "scheduling.k8s.io", version: "v1alpha1", kind: "Workload", want: true},
		{name: "volume attributes class v1", group: "storage.k8s.io", version: "v1", kind: "VolumeAttributesClass", want: true},
		{name: "storage version migration beta", group: "storagemigration.k8s.io", version: "v1beta1", kind: "StorageVersionMigration", want: true},
		{name: "storage version migration alpha", group: "storagemigration.k8s.io", version: "v1alpha1", kind: "StorageVersionMigration", want: true},
		{name: "pod template", version: "v1", kind: "PodTemplate", want: true},
		{name: "webhook has first-class provider resource", group: "admissionregistration.k8s.io", version: "v1", kind: "ValidatingWebhookConfiguration", want: false},
		{name: "other native resource is not manifest-backed", group: "coordination.k8s.io", version: "v1", kind: "Lease", want: false},
		{name: "event is not manifest-backed", group: "events.k8s.io", version: "v1", kind: "Event", want: false},
		{name: "token review is not manifest-backed", group: "authentication.k8s.io", version: "v1", kind: "TokenReview", want: false},
		{name: "pod certificate request is kubelet-generated", group: "certificates.k8s.io", version: "v1beta1", kind: "PodCertificateRequest", want: false},
		{name: "resource slice is generated by drivers", group: "resource.k8s.io", version: "v1", kind: "ResourceSlice", want: false},
		{name: "pod scheduling context is generated scheduling state", group: "resource.k8s.io", version: "v1alpha2", kind: "PodSchedulingContext", want: false},
		{name: "pod scheduling is generated scheduling state", group: "resource.k8s.io", version: "v1alpha1", kind: "PodScheduling", want: false},
		{name: "volume attachment is controller-managed", group: "storage.k8s.io", version: "v1", kind: "VolumeAttachment", want: false},
		{name: "ip address is allocator-managed", group: "networking.k8s.io", version: "v1", kind: "IPAddress", want: false},
		{name: "custom resource is handled by general manifest fallback", group: "example.com", version: "v1", kind: "Widget", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := supportsNativeManifestResource(tt.group, tt.version, tt.kind)
			if got != tt.want {
				t.Fatalf("supportsNativeManifestResource(%q, %q, %q) = %t, want %t", tt.group, tt.version, tt.kind, got, tt.want)
			}
		})
	}
}

// TestKubernetes134To136APIDiscoveryMatrix is a comprehensive fixture that
// classifies every relevant Kubernetes 1.34-1.36 API resource into exactly one
// behavior class. This test is the definition of done for the native API policy.
func TestKubernetes134To136APIDiscoveryMatrix(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	manageableVerbs := []string{"create", "delete", "get", "list", "patch", "update"}

	// allProviderTypes simulates a provider that has both first-class and
	// manifest resource types available.
	allProviderTypes := map[string]struct{}{
		"kubernetes_api_service_v1":                      {},
		"kubernetes_certificate_signing_request_v1":      {},
		"kubernetes_cluster_role_v1":                     {},
		"kubernetes_cluster_role_binding_v1":             {},
		"kubernetes_config_map_v1":                       {},
		"kubernetes_cron_job_v1":                         {},
		"kubernetes_csi_driver_v1":                       {},
		"kubernetes_daemon_set_v1":                       {},
		"kubernetes_default_service_account_v1":          {},
		"kubernetes_deployment_v1":                       {},
		"kubernetes_endpoint_slice_v1":                   {},
		"kubernetes_endpoints_v1":                        {},
		"kubernetes_horizontal_pod_autoscaler_v2":        {},
		"kubernetes_ingress_class_v1":                    {},
		"kubernetes_ingress_v1":                          {},
		"kubernetes_job_v1":                              {},
		"kubernetes_limit_range_v1":                      {},
		manifestTerraformResourceName:                    {},
		"kubernetes_mutating_webhook_configuration_v1":   {},
		"kubernetes_namespace_v1":                        {},
		"kubernetes_network_policy_v1":                   {},
		"kubernetes_node_taint":                          {},
		"kubernetes_persistent_volume_claim_v1":          {},
		"kubernetes_persistent_volume_v1":                {},
		"kubernetes_pod_disruption_budget_v1":            {},
		"kubernetes_pod_v1":                              {},
		"kubernetes_priority_class_v1":                   {},
		"kubernetes_replication_controller_v1":           {},
		"kubernetes_resource_quota_v1":                   {},
		"kubernetes_role_binding_v1":                     {},
		"kubernetes_role_v1":                             {},
		"kubernetes_runtime_class_v1":                    {},
		"kubernetes_secret_v1":                           {},
		"kubernetes_service_account_v1":                  {},
		"kubernetes_service_v1":                          {},
		"kubernetes_stateful_set_v1":                     {},
		"kubernetes_storage_class_v1":                    {},
		"kubernetes_validating_admission_policy_v1":      {},
		"kubernetes_validating_webhook_configuration_v1": {},
	}
	hasType := func(name string) bool {
		_, ok := allProviderTypes[name]
		return ok
	}

	type behaviorClass string
	const (
		firstClass     behaviorClass = "first-class Terraform resource"
		nativeManifest behaviorClass = "explicit native kubernetes_manifest selector"
		crdFallback    behaviorClass = "CRD/custom-resource manifest fallback"
		runtimeSkip    behaviorClass = "runtime/controller-generated skip"
		policySkip     behaviorClass = "native API outside manifest import policy"
	)

	tests := []struct {
		name     string
		group    string
		version  string
		resource metav1.APIResource
		class    behaviorClass
	}{
		// === First-class Terraform resources ===
		{
			name:     "core/v1 Service → first-class",
			version:  "v1",
			resource: metav1.APIResource{Name: "services", Kind: "Service", Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "core/v1 ConfigMap → first-class",
			version:  "v1",
			resource: metav1.APIResource{Name: "configmaps", Kind: "ConfigMap", Namespaced: true, Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "core/v1 Secret → first-class",
			version:  "v1",
			resource: metav1.APIResource{Name: "secrets", Kind: "Secret", Namespaced: true, Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "core/v1 Namespace → first-class",
			version:  "v1",
			resource: metav1.APIResource{Name: "namespaces", Kind: "Namespace", Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "core/v1 Pod → first-class",
			version:  "v1",
			resource: metav1.APIResource{Name: "pods", Kind: "Pod", Namespaced: true, Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "apps/v1 Deployment → first-class",
			group:    "apps",
			version:  "v1",
			resource: metav1.APIResource{Name: "deployments", Kind: "Deployment", Namespaced: true, Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "apps/v1 DaemonSet → first-class",
			group:    "apps",
			version:  "v1",
			resource: metav1.APIResource{Name: "daemonsets", Kind: "DaemonSet", Namespaced: true, Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "apps/v1 StatefulSet → first-class",
			group:    "apps",
			version:  "v1",
			resource: metav1.APIResource{Name: "statefulsets", Kind: "StatefulSet", Namespaced: true, Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "batch/v1 CronJob → first-class",
			group:    "batch",
			version:  "v1",
			resource: metav1.APIResource{Name: "cronjobs", Kind: "CronJob", Namespaced: true, Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "batch/v1 Job → first-class",
			group:    "batch",
			version:  "v1",
			resource: metav1.APIResource{Name: "jobs", Kind: "Job", Namespaced: true, Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "networking.k8s.io/v1 Ingress → first-class",
			group:    "networking.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "ingresses", Kind: "Ingress", Namespaced: true, Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "networking.k8s.io/v1 NetworkPolicy → first-class",
			group:    "networking.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "networkpolicies", Kind: "NetworkPolicy", Namespaced: true, Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "storage.k8s.io/v1 StorageClass → first-class",
			group:    "storage.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "storageclasses", Kind: "StorageClass", Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "storage.k8s.io/v1 CSIDriver → first-class",
			group:    "storage.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "csidrivers", Kind: "CSIDriver", Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "admissionregistration.k8s.io/v1 ValidatingAdmissionPolicy → first-class",
			group:    "admissionregistration.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "validatingadmissionpolicies", Kind: "ValidatingAdmissionPolicy", Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "apiregistration.k8s.io/v1 APIService → first-class (dynamic)",
			group:    "apiregistration.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "apiservices", Kind: "APIService", Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "autoscaling/v2 HorizontalPodAutoscaler → first-class",
			group:    "autoscaling",
			version:  "v2",
			resource: metav1.APIResource{Name: "horizontalpodautoscalers", Kind: "HorizontalPodAutoscaler", Namespaced: true, Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "scheduling.k8s.io/v1 PriorityClass → first-class",
			group:    "scheduling.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "priorityclasses", Kind: "PriorityClass", Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "rbac.authorization.k8s.io/v1 ClusterRole → first-class",
			group:    "rbac.authorization.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "clusterroles", Kind: "ClusterRole", Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "rbac.authorization.k8s.io/v1 ClusterRoleBinding → first-class",
			group:    "rbac.authorization.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "clusterrolebindings", Kind: "ClusterRoleBinding", Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "rbac.authorization.k8s.io/v1 Role → first-class",
			group:    "rbac.authorization.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "roles", Kind: "Role", Namespaced: true, Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "rbac.authorization.k8s.io/v1 RoleBinding → first-class",
			group:    "rbac.authorization.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "rolebindings", Kind: "RoleBinding", Namespaced: true, Verbs: manageableVerbs},
			class:    firstClass,
		},
		{
			name:     "policy/v1 PodDisruptionBudget → first-class",
			group:    "policy",
			version:  "v1",
			resource: metav1.APIResource{Name: "poddisruptionbudgets", Kind: "PodDisruptionBudget", Namespaced: true, Verbs: manageableVerbs},
			class:    firstClass,
		},

		// === Explicit native kubernetes_manifest selectors ===
		{
			name:     "admissionregistration.k8s.io/v1 MutatingAdmissionPolicy → native manifest",
			group:    "admissionregistration.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "mutatingadmissionpolicies", Kind: "MutatingAdmissionPolicy", Verbs: manageableVerbs},
			class:    nativeManifest,
		},
		{
			name:     "admissionregistration.k8s.io/v1 MutatingAdmissionPolicyBinding → native manifest",
			group:    "admissionregistration.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "mutatingadmissionpolicybindings", Kind: "MutatingAdmissionPolicyBinding", Verbs: manageableVerbs},
			class:    nativeManifest,
		},
		{
			name:     "admissionregistration.k8s.io/v1 ValidatingAdmissionPolicyBinding → native manifest",
			group:    "admissionregistration.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "validatingadmissionpolicybindings", Kind: "ValidatingAdmissionPolicyBinding", Verbs: manageableVerbs},
			class:    nativeManifest,
		},
		{
			name:     "apps/v1 ReplicaSet → native manifest",
			group:    "apps",
			version:  "v1",
			resource: metav1.APIResource{Name: "replicasets", Kind: "ReplicaSet", Namespaced: true, Verbs: manageableVerbs},
			class:    nativeManifest,
		},
		{
			name:     "certificates.k8s.io/v1beta1 ClusterTrustBundle → native manifest",
			group:    "certificates.k8s.io",
			version:  "v1beta1",
			resource: metav1.APIResource{Name: "clustertrustbundles", Kind: "ClusterTrustBundle", Verbs: manageableVerbs},
			class:    nativeManifest,
		},
		{
			name:     "flowcontrol.apiserver.k8s.io/v1 FlowSchema → native manifest",
			group:    "flowcontrol.apiserver.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "flowschemas", Kind: "FlowSchema", Verbs: manageableVerbs},
			class:    nativeManifest,
		},
		{
			name:     "flowcontrol.apiserver.k8s.io/v1 PriorityLevelConfiguration → native manifest",
			group:    "flowcontrol.apiserver.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "prioritylevelconfigurations", Kind: "PriorityLevelConfiguration", Verbs: manageableVerbs},
			class:    nativeManifest,
		},
		{
			name:     "networking.k8s.io/v1 ServiceCIDR → native manifest",
			group:    "networking.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "servicecidrs", Kind: "ServiceCIDR", Verbs: manageableVerbs},
			class:    nativeManifest,
		},
		{
			name:     "resource.k8s.io/v1 DeviceClass → native manifest",
			group:    "resource.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "deviceclasses", Kind: "DeviceClass", Verbs: manageableVerbs},
			class:    nativeManifest,
		},
		{
			name:     "resource.k8s.io/v1 ResourceClaim → native manifest",
			group:    "resource.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "resourceclaims", Kind: "ResourceClaim", Namespaced: true, Verbs: manageableVerbs},
			class:    nativeManifest,
		},
		{
			name:     "resource.k8s.io/v1 ResourceClaimTemplate → native manifest",
			group:    "resource.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "resourceclaimtemplates", Kind: "ResourceClaimTemplate", Namespaced: true, Verbs: manageableVerbs},
			class:    nativeManifest,
		},
		{
			name:     "resource.k8s.io/v1beta2 DeviceTaintRule → native manifest",
			group:    "resource.k8s.io",
			version:  "v1beta2",
			resource: metav1.APIResource{Name: "devicetaintrules", Kind: "DeviceTaintRule", Verbs: manageableVerbs},
			class:    nativeManifest,
		},
		{
			name:     "resource.k8s.io/v1alpha3 DeviceTaintRule → native manifest",
			group:    "resource.k8s.io",
			version:  "v1alpha3",
			resource: metav1.APIResource{Name: "devicetaintrules", Kind: "DeviceTaintRule", Verbs: manageableVerbs},
			class:    nativeManifest,
		},
		{
			name:     "scheduling.k8s.io/v1alpha2 PodGroup → native manifest",
			group:    "scheduling.k8s.io",
			version:  "v1alpha2",
			resource: metav1.APIResource{Name: "podgroups", Kind: "PodGroup", Namespaced: true, Verbs: manageableVerbs},
			class:    nativeManifest,
		},
		{
			name:     "scheduling.k8s.io/v1alpha2 Workload → native manifest",
			group:    "scheduling.k8s.io",
			version:  "v1alpha2",
			resource: metav1.APIResource{Name: "workloads", Kind: "Workload", Namespaced: true, Verbs: manageableVerbs},
			class:    nativeManifest,
		},
		{
			name:     "storage.k8s.io/v1 VolumeAttributesClass → native manifest",
			group:    "storage.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "volumeattributesclasses", Kind: "VolumeAttributesClass", Verbs: manageableVerbs},
			class:    nativeManifest,
		},
		{
			name:     "storagemigration.k8s.io/v1beta1 StorageVersionMigration → native manifest",
			group:    "storagemigration.k8s.io",
			version:  "v1beta1",
			resource: metav1.APIResource{Name: "storageversionmigrations", Kind: "StorageVersionMigration", Verbs: manageableVerbs},
			class:    nativeManifest,
		},
		{
			name:     "v1 PodTemplate → native manifest",
			version:  "v1",
			resource: metav1.APIResource{Name: "podtemplates", Kind: "PodTemplate", Namespaced: true, Verbs: manageableVerbs},
			class:    nativeManifest,
		},

		// === CRD/custom-resource manifest fallback ===
		{
			name:     "example.com/v1 Widget → CRD fallback",
			group:    "example.com",
			version:  "v1",
			resource: metav1.APIResource{Name: "widgets", Kind: "Widget", Namespaced: true, Verbs: manageableVerbs},
			class:    crdFallback,
		},
		{
			name:     "serving.knative.dev/v1 Service → CRD fallback",
			group:    "serving.knative.dev",
			version:  "v1",
			resource: metav1.APIResource{Name: "services", Kind: "Service", Namespaced: true, Verbs: manageableVerbs},
			class:    crdFallback,
		},
		{
			name:     "argoproj.io/v1alpha1 Application → CRD fallback",
			group:    "argoproj.io",
			version:  "v1alpha1",
			resource: metav1.APIResource{Name: "applications", Kind: "Application", Namespaced: true, Verbs: manageableVerbs},
			class:    crdFallback,
		},

		// === Runtime/controller-generated skip ===
		{
			name:     "apps/v1 ControllerRevision → runtime skip",
			group:    "apps",
			version:  "v1",
			resource: metav1.APIResource{Name: "controllerrevisions", Kind: "ControllerRevision", Namespaced: true, Verbs: manageableVerbs},
			class:    runtimeSkip,
		},
		{
			name:     "certificates.k8s.io/v1beta1 PodCertificateRequest → runtime skip",
			group:    "certificates.k8s.io",
			version:  "v1beta1",
			resource: metav1.APIResource{Name: "podcertificaterequests", Kind: "PodCertificateRequest", Namespaced: true, Verbs: manageableVerbs},
			class:    runtimeSkip,
		},
		{
			name:     "coordination.k8s.io/v1beta1 LeaseCandidate → runtime skip",
			group:    "coordination.k8s.io",
			version:  "v1beta1",
			resource: metav1.APIResource{Name: "leasecandidates", Kind: "LeaseCandidate", Namespaced: true, Verbs: manageableVerbs},
			class:    runtimeSkip,
		},
		{
			name:     "networking.k8s.io/v1 IPAddress → runtime skip",
			group:    "networking.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "ipaddresses", Kind: "IPAddress", Verbs: manageableVerbs},
			class:    runtimeSkip,
		},
		{
			name:     "resource.k8s.io/v1 ResourceSlice → runtime skip",
			group:    "resource.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "resourceslices", Kind: "ResourceSlice", Verbs: manageableVerbs},
			class:    runtimeSkip,
		},
		{
			name:     "resource.k8s.io/v1alpha3 PodSchedulingContext → runtime skip",
			group:    "resource.k8s.io",
			version:  "v1alpha3",
			resource: metav1.APIResource{Name: "podschedulingcontexts", Kind: "PodSchedulingContext", Namespaced: true, Verbs: manageableVerbs},
			class:    runtimeSkip,
		},
		{
			name:     "storage.k8s.io/v1 CSINode → runtime skip",
			group:    "storage.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "csinodes", Kind: "CSINode", Verbs: manageableVerbs},
			class:    runtimeSkip,
		},
		{
			name:     "storage.k8s.io/v1 CSIStorageCapacity → runtime skip",
			group:    "storage.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "csistoragecapacities", Kind: "CSIStorageCapacity", Namespaced: true, Verbs: manageableVerbs},
			class:    runtimeSkip,
		},
		{
			name:     "storage.k8s.io/v1 VolumeAttachment → runtime skip",
			group:    "storage.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "volumeattachments", Kind: "VolumeAttachment", Verbs: manageableVerbs},
			class:    runtimeSkip,
		},
		{
			name:     "internal.apiserver.k8s.io/v1alpha1 StorageVersion → runtime skip",
			group:    "internal.apiserver.k8s.io",
			version:  "v1alpha1",
			resource: metav1.APIResource{Name: "storageversions", Kind: "StorageVersion", Verbs: manageableVerbs},
			class:    runtimeSkip,
		},

		// === Native API outside explicit manifest import policy ===
		// These are skipped because the typed client exists in the pinned
		// client-go but no Terraform provider type is registered.
		{
			name:     "events.k8s.io/v1 Event → policy skip (no provider type)",
			group:    "events.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "events", Kind: "Event", Namespaced: true, Verbs: manageableVerbs},
			class:    policySkip,
		},
		{
			name:     "coordination.k8s.io/v1 Lease → policy skip (no provider type)",
			group:    "coordination.k8s.io",
			version:  "v1",
			resource: metav1.APIResource{Name: "leases", Kind: "Lease", Namespaced: true, Verbs: manageableVerbs},
			class:    policySkip,
		},
		// This legacy resource has no typed client and triggers the verbose
		// skip reason.
		{
			name:     "resource.k8s.io/v1alpha2 ResourceClass → policy skip (verbose)",
			group:    "resource.k8s.io",
			version:  "v1alpha2",
			resource: metav1.APIResource{Name: "resourceclasses", Kind: "ResourceClass", Verbs: manageableVerbs},
			class:    policySkip,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tfName, _, ok := selectImportResourceName(clientset, tt.group, tt.version, tt.resource, hasType)
			reason := importSkipPolicyReason(clientset, tt.group, tt.version, tt.resource, hasType)

			var got behaviorClass
			switch {
			case ok && tfName != manifestTerraformResourceName:
				got = firstClass
			case ok && tfName == manifestTerraformResourceName && isNativeAPIGroup(tt.group):
				got = nativeManifest
			case ok && tfName == manifestTerraformResourceName && !isNativeAPIGroup(tt.group):
				got = crdFallback
			case !ok && reason == "runtime/controller-generated native API is not importable as Terraform-managed configuration":
				got = runtimeSkip
			case !ok && isNativeAPIGroup(tt.group):
				// Native APIs that are not imported: either verbose policy
				// skip (no typed client, reason != "") or silent skip (typed
				// client exists but no provider type, reason == "").
				got = policySkip
			}

			if got != tt.class {
				t.Fatalf("API %s/%s/%s classified as %q, want %q (tfName=%q, ok=%t, reason=%q)",
					tt.group, tt.version, tt.resource.Kind, got, tt.class, tfName, ok, reason)
			}
		})
	}
}

// TestVerboseSkipLoggingForNativeAPIs verifies that importSkipPolicyReason
// returns the expected reason strings for the two skip categories, enabling
// --verbose logging to explain why native APIs are not imported.
func TestVerboseSkipLoggingForNativeAPIs(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	manageableVerbs := []string{"create", "delete", "get", "list", "patch", "update"}
	hasManifestType := func(name string) bool {
		return name == manifestTerraformResourceName
	}

	tests := []struct {
		name       string
		group      string
		version    string
		resource   metav1.APIResource
		wantReason string
	}{
		{
			name:       "runtime skip reason for ResourceSlice",
			group:      "resource.k8s.io",
			version:    "v1",
			resource:   metav1.APIResource{Name: "resourceslices", Kind: "ResourceSlice", Verbs: manageableVerbs},
			wantReason: "runtime/controller-generated native API is not importable as Terraform-managed configuration",
		},
		{
			name:       "runtime skip reason for PodCertificateRequest",
			group:      "certificates.k8s.io",
			version:    "v1beta1",
			resource:   metav1.APIResource{Name: "podcertificaterequests", Kind: "PodCertificateRequest", Namespaced: true, Verbs: manageableVerbs},
			wantReason: "runtime/controller-generated native API is not importable as Terraform-managed configuration",
		},
		{
			name:       "runtime skip reason for VolumeAttachment",
			group:      "storage.k8s.io",
			version:    "v1",
			resource:   metav1.APIResource{Name: "volumeattachments", Kind: "VolumeAttachment", Verbs: manageableVerbs},
			wantReason: "runtime/controller-generated native API is not importable as Terraform-managed configuration",
		},
		{
			name:       "no verbose reason for Event (typed client exists but no provider type)",
			group:      "events.k8s.io",
			version:    "v1",
			resource:   metav1.APIResource{Name: "events", Kind: "Event", Namespaced: true, Verbs: manageableVerbs},
			wantReason: "",
		},
		{
			name:       "no verbose reason for Lease (typed client exists but no provider type)",
			group:      "coordination.k8s.io",
			version:    "v1",
			resource:   metav1.APIResource{Name: "leases", Kind: "Lease", Namespaced: true, Verbs: manageableVerbs},
			wantReason: "",
		},
		{
			name:       "policy skip reason for legacy ResourceClass",
			group:      "resource.k8s.io",
			version:    "v1alpha2",
			resource:   metav1.APIResource{Name: "resourceclasses", Kind: "ResourceClass", Verbs: manageableVerbs},
			wantReason: "native API is outside the explicit manifest import policy",
		},
		{
			name:       "no skip reason for allowlisted native manifest resource",
			group:      "resource.k8s.io",
			version:    "v1",
			resource:   metav1.APIResource{Name: "resourceclaims", Kind: "ResourceClaim", Namespaced: true, Verbs: manageableVerbs},
			wantReason: "",
		},
		{
			name:       "no skip reason for CRD/custom resource",
			group:      "example.com",
			version:    "v1",
			resource:   metav1.APIResource{Name: "widgets", Kind: "Widget", Namespaced: true, Verbs: manageableVerbs},
			wantReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := importSkipPolicyReason(clientset, tt.group, tt.version, tt.resource, hasManifestType)
			if got != tt.wantReason {
				t.Fatalf("importSkipPolicyReason() = %q, want %q", got, tt.wantReason)
			}
		})
	}
}

// TestCRDManifestFallbackNotBroken verifies that CRDs and custom resources
// continue to use the generic manifest fallback path regardless of the native
// API policy.
func TestCRDManifestFallbackNotBroken(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	manageableVerbs := []string{"create", "delete", "get", "list", "patch", "update"}
	hasManifestType := func(name string) bool {
		return name == manifestTerraformResourceName
	}

	tests := []struct {
		name    string
		group   string
		version string
		kind    string
		plural  string
	}{
		{name: "typical CRD", group: "example.com", version: "v1", kind: "Widget", plural: "widgets"},
		{name: "knative service", group: "serving.knative.dev", version: "v1", kind: "Service", plural: "services"},
		{name: "argo application", group: "argoproj.io", version: "v1alpha1", kind: "Application", plural: "applications"},
		{name: "cert-manager certificate", group: "cert-manager.io", version: "v1", kind: "Certificate", plural: "certificates"},
		{name: "istio virtual service", group: "networking.istio.io", version: "v1", kind: "VirtualService", plural: "virtualservices"},
		{name: "custom group sharing native prefix", group: "apps.example.com", version: "v1", kind: "Deployment", plural: "deployments"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := metav1.APIResource{
				Name:       tt.plural,
				Kind:       tt.kind,
				Namespaced: true,
				Verbs:      manageableVerbs,
			}
			tfName, dynamic, ok := selectImportResourceName(clientset, tt.group, tt.version, resource, hasManifestType)
			if !ok {
				t.Fatalf("CRD %s/%s/%s was not imported, want manifest fallback", tt.group, tt.version, tt.kind)
			}
			if tfName != manifestTerraformResourceName {
				t.Fatalf("CRD terraform type = %q, want %q", tfName, manifestTerraformResourceName)
			}
			if !dynamic {
				t.Fatal("CRD dynamic = false, want true")
			}

			reason := importSkipPolicyReason(clientset, tt.group, tt.version, resource, hasManifestType)
			if reason != "" {
				t.Fatalf("CRD should have no skip reason, got %q", reason)
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
		{name: "apps replica set", group: "apps", version: "v1", kind: "ReplicaSet", want: true},
		{name: "core pod template", version: "v1", kind: "PodTemplate", want: true},
		{name: "custom group sharing native prefix is not typed", group: "apps.example.com", version: "v1", kind: "Deployment", want: false},
		{name: "autoscaling hpa", group: "autoscaling", version: "v2", kind: "HorizontalPodAutoscaler", want: true},
		{name: "autoscaling v2beta2 hpa is not exposed by kubernetes clientset", group: "autoscaling", version: "v2beta2", kind: "HorizontalPodAutoscaler", want: false},
		{name: "certificate signing request", group: "certificates.k8s.io", version: "v1", kind: "CertificateSigningRequest", want: true},
		{name: "pod certificate request", group: "certificates.k8s.io", version: "v1beta1", kind: "PodCertificateRequest", want: true},
		{name: "storage csi driver", group: "storage.k8s.io", version: "v1", kind: "CSIDriver", want: true},
		{name: "storage version migration", group: "storagemigration.k8s.io", version: "v1beta1", kind: "StorageVersionMigration", want: true},
		{name: "scheduling workload preferred alpha", group: "scheduling.k8s.io", version: "v1alpha2", kind: "Workload", want: true},
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
