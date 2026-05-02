// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"reflect"
	"testing"

	"k8s.io/client-go/kubernetes/fake"
)

func TestTerraformResourceNameCandidates(t *testing.T) {
	tests := []struct {
		name string
		kind string
		want []string
	}{
		{
			name: "uses default name for existing core resource",
			kind: "Service",
			want: []string{"kubernetes_service"},
		},
		{
			name: "prefers modern daemon set name before legacy provider spelling",
			kind: "DaemonSet",
			want: []string{"kubernetes_daemon_set_v1", "kubernetes_daemonset", "kubernetes_daemon_set"},
		},
		{
			name: "uses v1-only endpoint slice name",
			kind: "EndpointSlice",
			want: []string{"kubernetes_endpoint_slice_v1", "kubernetes_endpoint_slice"},
		},
		{
			name: "uses v1-only runtime class name",
			kind: "RuntimeClass",
			want: []string{"kubernetes_runtime_class_v1", "kubernetes_runtime_class"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := terraformResourceNameCandidates(tt.kind)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("terraformResourceNameCandidates(%q) = %#v, want %#v", tt.kind, got, tt.want)
			}
		})
	}
}

func TestSelectTerraformResourceName(t *testing.T) {
	tests := []struct {
		name           string
		kind           string
		supportedTypes map[string]struct{}
		want           string
		wantOK         bool
	}{
		{
			name: "selects default name",
			kind: "Service",
			supportedTypes: map[string]struct{}{
				"kubernetes_service": {},
			},
			want:   "kubernetes_service",
			wantOK: true,
		},
		{
			name: "prefers modern mapped name",
			kind: "DaemonSet",
			supportedTypes: map[string]struct{}{
				"kubernetes_daemonset":     {},
				"kubernetes_daemon_set_v1": {},
			},
			want:   "kubernetes_daemon_set_v1",
			wantOK: true,
		},
		{
			name: "falls back to legacy mapped name",
			kind: "DaemonSet",
			supportedTypes: map[string]struct{}{
				"kubernetes_daemonset": {},
			},
			want:   "kubernetes_daemonset",
			wantOK: true,
		},
		{
			name: "returns false when provider has no matching resource",
			kind: "EndpointSlice",
			supportedTypes: map[string]struct{}{
				"kubernetes_service": {},
			},
			want:   "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := selectTerraformResourceName(tt.kind, func(name string) bool {
				_, exists := tt.supportedTypes[name]
				return exists
			})
			if ok != tt.wantOK {
				t.Fatalf("selectTerraformResourceName(%q) ok = %t, want %t", tt.kind, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("selectTerraformResourceName(%q) = %q, want %q", tt.kind, got, tt.want)
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
