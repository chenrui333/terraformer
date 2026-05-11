// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"encoding/json"
	"os"
	"sort"
	"testing"
)

func TestMediaUnsupportedResourceEntries(t *testing.T) {
	data, err := os.ReadFile("unsupported_resources.json")
	if err != nil {
		t.Fatalf("read unsupported resources: %v", err)
	}
	var unsupported map[string]interface{}
	if err := json.Unmarshal(data, &unsupported); err != nil {
		t.Fatalf("decode unsupported resources: %v", err)
	}

	entries, ok := unsupported["resources"].([]interface{})
	if !ok {
		t.Fatal("unsupported resources file is missing resources list")
	}

	wantEntries := map[string]string{
		"aws_elastic_beanstalk_application_version":    "elastic_beanstalk",
		"aws_elastic_beanstalk_configuration_template": "elastic_beanstalk",
		"aws_ivs_playback_key_pair":                    "ivs",
		"aws_qldb_stream":                              "qldb",
	}
	foundEntries := map[string]bool{}
	resources := make([]string, 0, len(entries))
	for _, rawEntry := range entries {
		entry, ok := rawEntry.(map[string]interface{})
		if !ok {
			t.Fatalf("unsupported resource entry has unexpected type %T", rawEntry)
		}
		resource, _ := entry["resource"].(string)
		resources = append(resources, resource)
		if wantServiceFamily, ok := wantEntries[resource]; ok {
			foundEntries[resource] = true
			if serviceFamily, _ := entry["service_family"].(string); serviceFamily != wantServiceFamily {
				t.Fatalf("%s service family = %q, want %s", resource, serviceFamily, wantServiceFamily)
			}
			if status, _ := entry["status"].(string); status != "unsupported" {
				t.Fatalf("%s status = %q, want unsupported", resource, status)
			}
			references, _ := entry["references"].([]interface{})
			reason, _ := entry["reason"].(string)
			evidence, _ := entry["evidence"].(string)
			if reason == "" || evidence == "" || len(references) == 0 {
				t.Fatalf("%s unsupported entry is missing reason, evidence, or references", resource)
			}
		}
	}
	for resource := range wantEntries {
		if !foundEntries[resource] {
			t.Fatalf("%s unsupported entry was not found", resource)
		}
	}
	if !sort.StringsAreSorted(resources) {
		t.Fatalf("unsupported resources are not sorted by resource: %v", resources)
	}
}
