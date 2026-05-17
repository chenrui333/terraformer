// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"encoding/json"
	"os"
	"sort"
	"testing"
)

func TestUnsupportedResourceMetadata(t *testing.T) {
	data, err := os.ReadFile("unsupported_resources.json")
	if err != nil {
		t.Fatalf("read unsupported resources: %v", err)
	}

	var unsupported map[string]interface{}
	if err := json.Unmarshal(data, &unsupported); err != nil {
		t.Fatalf("decode unsupported resources: %v", err)
	}
	version, ok := unsupported["version"].(float64)
	if !ok || int(version) != 1 {
		t.Fatalf("unsupported resources version = %v, want 1", unsupported["version"])
	}
	rawEntries, ok := unsupported["resources"].([]interface{})
	if !ok || len(rawEntries) == 0 {
		t.Fatal("unsupported resources file is missing resources list")
	}

	allowedStatuses := map[string]struct{}{
		"not-importable":    {},
		"policy-skip":       {},
		"runtime-generated": {},
	}
	entries := map[string]string{}
	resources := make([]string, 0, len(rawEntries))
	for _, rawEntry := range rawEntries {
		entry, ok := rawEntry.(map[string]interface{})
		if !ok {
			t.Fatalf("unsupported resource entry has unexpected type %T", rawEntry)
		}
		resource, _ := entry["resource"].(string)
		serviceFamily, _ := entry["service_family"].(string)
		reason, _ := entry["reason"].(string)
		evidence, _ := entry["evidence"].(string)
		status, _ := entry["status"].(string)
		references, _ := entry["references"].([]interface{})
		if resource == "" || serviceFamily == "" || reason == "" || evidence == "" {
			t.Fatalf("unsupported resource entry is missing required metadata: %+v", entry)
		}
		if len(references) == 0 {
			t.Fatalf("%s unsupported entry is missing references", resource)
		}
		if _, ok := allowedStatuses[status]; !ok {
			t.Fatalf("%s status = %q, want one of %v", resource, status, sortedKeys(allowedStatuses))
		}
		if _, exists := entries[resource]; exists {
			t.Fatalf("duplicate unsupported resource entry: %s", resource)
		}
		entries[resource] = status
		resources = append(resources, resource)
	}
	if !sort.StringsAreSorted(resources) {
		t.Fatalf("unsupported resources are not sorted by resource: %v", resources)
	}

	for resource := range skippedImportResources {
		name := unsupportedResourceMetadataName(resource)
		if status, ok := entries[name]; !ok {
			t.Fatalf("%s runtime-generated entry was not found", name)
		} else if status != "runtime-generated" {
			t.Fatalf("%s status = %q, want runtime-generated", name, status)
		}
	}

	wantEntries := map[string]string{
		"coordination.k8s.io/v1 Lease":           "policy-skip",
		"events.k8s.io/v1 Event":                 "policy-skip",
		"kubernetes_token_request_v1":            "not-importable",
		"resource.k8s.io/v1alpha2 ResourceClass": "policy-skip",
	}
	for resource, wantStatus := range wantEntries {
		if status, ok := entries[resource]; !ok {
			t.Fatalf("%s entry was not found", resource)
		} else if status != wantStatus {
			t.Fatalf("%s status = %q, want %s", resource, status, wantStatus)
		}
	}
}

func unsupportedResourceMetadataName(resource kubernetesResourceID) string {
	if resource.group == "" {
		return resource.version + " " + resource.kind
	}
	return resource.group + "/" + resource.version + " " + resource.kind
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
