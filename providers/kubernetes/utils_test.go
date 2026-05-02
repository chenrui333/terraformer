// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"reflect"
	"testing"

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
			name:    "uses default name for existing core resource",
			version: "v1",
			kind:    "Service",
			want:    []string{"kubernetes_service"},
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
		{name: "apps daemon set", group: "apps", version: "v1", kind: "DaemonSet", want: true},
		{name: "discovery endpoint slice", group: "discovery.k8s.io", version: "v1", kind: "EndpointSlice", want: true},
		{name: "node runtime class", group: "node.k8s.io", version: "v1", kind: "RuntimeClass", want: true},
		{name: "api service is not exposed by kubernetes clientset", group: "apiregistration.k8s.io", version: "v1", kind: "APIService", want: false},
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
