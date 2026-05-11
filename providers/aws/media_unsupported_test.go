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

	resources := make([]string, 0, len(entries))
	foundPlaybackKeyPair := false
	for _, rawEntry := range entries {
		entry, ok := rawEntry.(map[string]interface{})
		if !ok {
			t.Fatalf("unsupported resource entry has unexpected type %T", rawEntry)
		}
		resource, _ := entry["resource"].(string)
		resources = append(resources, resource)
		if resource == "aws_ivs_playback_key_pair" {
			foundPlaybackKeyPair = true
			if serviceFamily, _ := entry["service_family"].(string); serviceFamily != "ivs" {
				t.Fatalf("playback key pair service family = %q, want ivs", serviceFamily)
			}
			if status, _ := entry["status"].(string); status != "unsupported" {
				t.Fatalf("playback key pair status = %q, want unsupported", status)
			}
			references, _ := entry["references"].([]interface{})
			reason, _ := entry["reason"].(string)
			evidence, _ := entry["evidence"].(string)
			if reason == "" || evidence == "" || len(references) == 0 {
				t.Fatal("playback key pair unsupported entry is missing reason, evidence, or references")
			}
		}
	}
	if !foundPlaybackKeyPair {
		t.Fatal("aws_ivs_playback_key_pair unsupported entry was not found")
	}
	if !sort.StringsAreSorted(resources) {
		t.Fatalf("unsupported resources are not sorted by resource: %v", resources)
	}
}
