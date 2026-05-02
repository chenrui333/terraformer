// SPDX-License-Identifier: Apache-2.0

package kubernetesmanifest

import "testing"

func TestConfigFromObjectStripsServerOwnedFields(t *testing.T) {
	object := map[string]interface{}{
		"apiVersion": "example.com/v1",
		"kind":       "Widget",
		"metadata": map[string]interface{}{
			"name":              "sample",
			"namespace":         "default",
			"resourceVersion":   "123",
			"uid":               "uid-123",
			"managedFields":     []interface{}{map[string]interface{}{"manager": "controller"}},
			"creationTimestamp": "2026-05-02T00:00:00Z",
			"labels": map[string]interface{}{
				"app": "sample",
			},
		},
		"spec": map[string]interface{}{
			"replicas": float64(1),
		},
		"status": map[string]interface{}{
			"phase": "Ready",
		},
	}

	manifest := ConfigFromObject(object)

	if _, ok := manifest["status"]; ok {
		t.Fatal("status was not stripped")
	}
	metadata := manifest["metadata"].(map[string]interface{})
	for _, key := range []string{"resourceVersion", "uid", "managedFields", "creationTimestamp"} {
		if _, ok := metadata[key]; ok {
			t.Fatalf("metadata.%s was not stripped", key)
		}
	}
	if metadata["name"] != "sample" {
		t.Fatalf("metadata.name = %v, want %q", metadata["name"], "sample")
	}
	labels := metadata["labels"].(map[string]interface{})
	if labels["app"] != "sample" {
		t.Fatalf("metadata.labels.app = %v, want %q", labels["app"], "sample")
	}

	originalMetadata := object["metadata"].(map[string]interface{})
	if _, ok := originalMetadata["uid"]; !ok {
		t.Fatal("original object metadata.uid was mutated")
	}
	if _, ok := object["status"]; !ok {
		t.Fatal("original object status was mutated")
	}
}
