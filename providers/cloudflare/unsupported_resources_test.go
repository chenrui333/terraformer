// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const cloudflareUnsupportedIssue = "https://github.com/chenrui333/terraformer/issues/335"

type cloudflareUnsupportedResourcesFile struct {
	Version   int                                   `json:"version"`
	Resources []cloudflareUnsupportedResourceRecord `json:"resources"`
}

type cloudflareUnsupportedResourceRecord struct {
	Resource      string   `json:"resource"`
	ServiceFamily string   `json:"service_family"`
	Reason        string   `json:"reason"`
	Evidence      string   `json:"evidence"`
	Status        string   `json:"status"`
	References    []string `json:"references"`
}

func TestCloudflareUnsupportedResourcesMetadata(t *testing.T) {
	data, err := os.ReadFile("unsupported_resources.json")
	if err != nil {
		t.Fatalf("read unsupported resources: %v", err)
	}

	var metadata cloudflareUnsupportedResourcesFile
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("decode unsupported resources: %v", err)
	}
	if metadata.Version != 1 {
		t.Fatalf("unsupported resources version = %d, want 1", metadata.Version)
	}
	if len(metadata.Resources) == 0 {
		t.Fatal("unsupported resources file is missing resources list")
	}

	allowedStatuses := map[string]bool{
		"cloudflare-managed": true,
		"deferred":           true,
		"not-importable":     true,
		"request-style":      true,
		"secret-required":    true,
		"unsupported":        true,
	}
	seen := map[string]bool{}
	previousResource := ""
	for _, resource := range metadata.Resources {
		if resource.Resource == "" {
			t.Fatal("unsupported resource entry is missing resource")
		}
		if !strings.HasPrefix(resource.Resource, "cloudflare_") {
			t.Fatalf("unsupported resource %q does not use cloudflare_ prefix", resource.Resource)
		}
		if seen[resource.Resource] {
			t.Fatalf("unsupported resource %q is duplicated", resource.Resource)
		}
		seen[resource.Resource] = true
		if previousResource != "" && previousResource > resource.Resource {
			t.Fatalf("unsupported resources are not sorted by resource: %q before %q", previousResource, resource.Resource)
		}
		previousResource = resource.Resource

		if resource.ServiceFamily == "" {
			t.Fatalf("unsupported resource %q is missing service_family", resource.Resource)
		}
		if resource.Reason == "" {
			t.Fatalf("unsupported resource %q is missing reason", resource.Resource)
		}
		if resource.Evidence == "" {
			t.Fatalf("unsupported resource %q is missing evidence", resource.Resource)
		}
		if !allowedStatuses[resource.Status] {
			t.Fatalf("unsupported resource %q has unknown status %q", resource.Resource, resource.Status)
		}
		if !hasReference(resource.References, cloudflareUnsupportedIssue) {
			t.Fatalf("unsupported resource %q is missing issue #335 reference", resource.Resource)
		}
		for _, reference := range resource.References {
			if strings.TrimSpace(reference) == "" {
				t.Fatalf("unsupported resource %q has an empty reference", resource.Resource)
			}
		}
	}
}

func hasReference(references []string, expected string) bool {
	for _, reference := range references {
		if reference == expected {
			return true
		}
	}
	return false
}
