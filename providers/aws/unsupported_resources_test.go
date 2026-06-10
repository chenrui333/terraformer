// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const awsUnsupportedIssue = "https://github.com/chenrui333/terraformer/issues/338"

func TestAWSUnsupportedResourcesMetadata(t *testing.T) {
	data, err := os.ReadFile("unsupported_resources.json")
	if err != nil {
		t.Fatalf("read unsupported resources: %v", err)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("decode unsupported resources: %v", err)
	}
	if version, ok := metadata["version"].(float64); !ok || int(version) != 1 {
		t.Fatalf("unsupported resources version = %v, want 1", metadata["version"])
	}
	rawResources, ok := metadata["resources"].([]interface{})
	if !ok || len(rawResources) == 0 {
		t.Fatal("unsupported resources file is missing resources list")
	}

	allowedStatuses := map[string]bool{
		"deferred":       true,
		"not-importable": true,
		"runtime-data":   true,
		"unsupported":    true,
	}
	seen := map[string]bool{}
	previousResource := ""
	for _, rawResource := range rawResources {
		resource, ok := rawResource.(map[string]interface{})
		if !ok {
			t.Fatalf("unsupported resource entry has unexpected type %T", rawResource)
		}
		name, _ := resource["resource"].(string)
		serviceFamily, _ := resource["service_family"].(string)
		reason, _ := resource["reason"].(string)
		evidence, _ := resource["evidence"].(string)
		status, _ := resource["status"].(string)
		references, _ := resource["references"].([]interface{})

		if name == "" {
			t.Fatal("unsupported resource entry is missing resource")
		}
		if !strings.HasPrefix(name, "aws_") {
			t.Fatalf("unsupported resource %q does not use aws_ prefix", name)
		}
		if seen[name] {
			t.Fatalf("unsupported resource %q is duplicated", name)
		}
		seen[name] = true
		if previousResource != "" && previousResource > name {
			t.Fatalf("unsupported resources are not sorted by resource: %q before %q", previousResource, name)
		}
		previousResource = name

		if serviceFamily == "" {
			t.Fatalf("unsupported resource %q is missing service_family", name)
		}
		if reason == "" {
			t.Fatalf("unsupported resource %q is missing reason", name)
		}
		if evidence == "" {
			t.Fatalf("unsupported resource %q is missing evidence", name)
		}
		if !allowedStatuses[status] {
			t.Fatalf("unsupported resource %q has unknown status %q", name, status)
		}
		if !hasUnsupportedResourceReference(references, awsUnsupportedIssue) {
			t.Fatalf("unsupported resource %q is missing issue #338 reference", name)
		}
		for _, rawReference := range references {
			reference, _ := rawReference.(string)
			if strings.TrimSpace(reference) == "" {
				t.Fatalf("unsupported resource %q has an empty reference", name)
			}
		}
	}
}

func hasUnsupportedResourceReference(references []interface{}, expected string) bool {
	for _, rawReference := range references {
		if reference, _ := rawReference.(string); reference == expected {
			return true
		}
	}
	return false
}
