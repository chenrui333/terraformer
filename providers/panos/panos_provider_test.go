// SPDX-License-Identifier: Apache-2.0

package panos

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type panosUnsupportedResourcesFile struct {
	Version   int                                `json:"version"`
	Resources []panosUnsupportedResourceMetadata `json:"resources"`
}

type panosUnsupportedResourceMetadata struct {
	Resource   string   `json:"resource"`
	Status     string   `json:"status"`
	References []string `json:"references"`
}

func TestPanosProviderInitRequiresArgs(t *testing.T) {
	provider := PanosProvider{vsys: "old-vsys"}

	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected missing args error")
	}
	if !strings.Contains(err.Error(), "vsys is required") {
		t.Fatalf("Init error = %q, want missing PAN-OS args", err)
	}
	if provider.vsys != "" {
		t.Fatalf("vsys = %q, want empty after failed init", provider.vsys)
	}
}

func TestPanosProviderInitStoresArgs(t *testing.T) {
	var provider PanosProvider

	if err := provider.Init([]string{"vsys1"}); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.vsys != "vsys1" {
		t.Fatalf("vsys = %q, want vsys1", provider.vsys)
	}
}

func TestPanosBGPAuthProfileResourcesAreSkipped(t *testing.T) {
	if resources := (&FirewallNetworkingGenerator{}).createBGPAuthProfileResources("virtual-router"); len(resources) != 0 {
		t.Fatalf("firewall BGP auth profile resources len = %d, want 0", len(resources))
	}

	if resources := (&PanoramaNetworkingGenerator{}).createBGPAuthProfileResources("template", "template-stack", "virtual-router"); len(resources) != 0 {
		t.Fatalf("panorama BGP auth profile resources len = %d, want 0", len(resources))
	}
}

func TestPanosUnsupportedResourcesMetadata(t *testing.T) {
	data, err := os.ReadFile("unsupported_resources.json")
	if err != nil {
		t.Fatalf("read unsupported resources: %v", err)
	}

	var metadata panosUnsupportedResourcesFile
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("decode unsupported resources: %v", err)
	}
	if metadata.Version != 1 {
		t.Fatalf("unsupported resources version = %d, want 1", metadata.Version)
	}

	entries := map[string]panosUnsupportedResourceMetadata{}
	for _, resource := range metadata.Resources {
		entries[resource.Resource] = resource
	}

	for _, resourceName := range []string{"panos_bgp_auth_profile", "panos_panorama_bgp_auth_profile"} {
		resource, ok := entries[resourceName]
		if !ok {
			t.Fatalf("unsupported metadata is missing %s", resourceName)
		}
		if resource.Status != "secret-required" {
			t.Fatalf("%s status = %q, want secret-required", resourceName, resource.Status)
		}
		if len(resource.References) == 0 {
			t.Fatalf("%s metadata is missing references", resourceName)
		}
	}
}
