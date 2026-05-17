// SPDX-License-Identifier: Apache-2.0

package kafka

import (
	"encoding/json"
	"os"
	"sort"
	"testing"
)

const kafkaUnsupportedIssue = "https://github.com/chenrui333/terraformer/issues/481"

type kafkaUnsupportedResourcesFile struct {
	Version   int                                `json:"version"`
	Resources []kafkaUnsupportedResourceMetadata `json:"resources"`
}

type kafkaUnsupportedResourceMetadata struct {
	Resource      string   `json:"resource"`
	ServiceFamily string   `json:"service_family"`
	Reason        string   `json:"reason"`
	Evidence      string   `json:"evidence"`
	Status        string   `json:"status"`
	References    []string `json:"references"`
}

func TestKafkaUnsupportedResourcesMetadata(t *testing.T) {
	data, err := os.ReadFile("unsupported_resources.json")
	if err != nil {
		t.Fatalf("read unsupported resources: %v", err)
	}

	var metadata kafkaUnsupportedResourcesFile
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("decode unsupported resources: %v", err)
	}
	if metadata.Version != 1 {
		t.Fatalf("unsupported resources version = %d, want 1", metadata.Version)
	}

	wantStatuses := map[string]string{
		"kafka_quota":                 "not-importable",
		"kafka_user_scram_credential": "secret-required",
	}
	seen := map[string]string{}
	resources := make([]string, 0, len(metadata.Resources))
	for _, resource := range metadata.Resources {
		if resource.Resource == "" || resource.ServiceFamily == "" || resource.Reason == "" || resource.Evidence == "" {
			t.Fatalf("unsupported resource entry is missing required metadata: %+v", resource)
		}
		if len(resource.References) == 0 {
			t.Fatalf("unsupported resource %q is missing references", resource.Resource)
		}
		if !hasReference(resource.References, kafkaUnsupportedIssue) {
			t.Fatalf("unsupported resource %q is missing issue #481 reference", resource.Resource)
		}
		if _, exists := seen[resource.Resource]; exists {
			t.Fatalf("duplicate unsupported resource entry: %s", resource.Resource)
		}
		seen[resource.Resource] = resource.Status
		resources = append(resources, resource.Resource)
	}
	if !sort.StringsAreSorted(resources) {
		t.Fatalf("unsupported resources are not sorted by resource: %v", resources)
	}

	for resource, wantStatus := range wantStatuses {
		if status, ok := seen[resource]; !ok {
			t.Fatalf("%s unsupported metadata entry was not found", resource)
		} else if status != wantStatus {
			t.Fatalf("%s status = %q, want %s", resource, status, wantStatus)
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
