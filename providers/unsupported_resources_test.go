// SPDX-License-Identifier: Apache-2.0

package providers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/chenrui333/terraformer/internal/unsupportedresources"
)

const unsupportedResourcesVersion = 1

type unsupportedResourcesFile struct {
	Version   *int                          `json:"version"`
	Resources []unsupportedResourceMetadata `json:"resources"`
}

type unsupportedResourceMetadata struct {
	Resource      string   `json:"resource"`
	ServiceFamily string   `json:"service_family"`
	Reason        string   `json:"reason"`
	Evidence      string   `json:"evidence"`
	Status        string   `json:"status"`
	References    []string `json:"references"`
}

func TestUnsupportedResourcesMetadata(t *testing.T) {
	metadataFiles, err := filepath.Glob("*/unsupported_resources.json")
	if err != nil {
		t.Fatalf("discover unsupported resource metadata: %v", err)
	}
	if len(metadataFiles) == 0 {
		t.Fatal("no provider unsupported_resources.json files were found")
	}
	sort.Strings(metadataFiles)

	for _, metadataFile := range metadataFiles {
		t.Run(filepath.Dir(metadataFile), func(t *testing.T) {
			validateUnsupportedResourcesMetadataFile(t, metadataFile)
		})
	}
}

func validateUnsupportedResourcesMetadataFile(t *testing.T, metadataFile string) {
	t.Helper()

	data, err := os.ReadFile(metadataFile)
	if err != nil {
		t.Fatalf("read %s: %v", metadataFile, err)
	}

	var metadata unsupportedResourcesFile
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("decode %s: %v", metadataFile, err)
	}
	if metadata.Version == nil {
		t.Fatalf("%s is missing top-level version", metadataFile)
	}
	if *metadata.Version != unsupportedResourcesVersion {
		t.Fatalf("%s version = %d, want %d", metadataFile, *metadata.Version, unsupportedResourcesVersion)
	}
	if len(metadata.Resources) == 0 {
		t.Fatalf("%s is missing resource entries", metadataFile)
	}

	seenResources := map[string]struct{}{}
	for index, entry := range metadata.Resources {
		resource := strings.TrimSpace(entry.Resource)
		if resource == "" {
			t.Fatalf("%s resources[%d] is missing resource", metadataFile, index)
		}
		if _, ok := seenResources[resource]; ok {
			t.Fatalf("%s contains duplicate resource %q", metadataFile, resource)
		}
		seenResources[resource] = struct{}{}
		validateRequiredString(t, metadataFile, resource, "service_family", entry.ServiceFamily)
		validateRequiredString(t, metadataFile, resource, "reason", entry.Reason)
		validateRequiredString(t, metadataFile, resource, "evidence", entry.Evidence)
		validateUnsupportedResourceStatus(t, metadataFile, resource, entry.Status)
		validateUnsupportedResourceReferences(t, metadataFile, resource, entry.References)
	}
}

func validateRequiredString(t *testing.T, metadataFile, resource, field, value string) {
	t.Helper()

	if strings.TrimSpace(value) == "" {
		t.Fatalf("%s resource %q is missing %s", metadataFile, resource, field)
	}
}

func validateUnsupportedResourceStatus(t *testing.T, metadataFile, resource, status string) {
	t.Helper()

	if strings.TrimSpace(status) == "" {
		t.Fatalf("%s resource %q is missing status", metadataFile, resource)
	}
	if !unsupportedresources.IsValidStatus(status) {
		t.Fatalf("%s resource %q has unsupported status %q, want one of %v", metadataFile, resource, status, unsupportedresources.Statuses())
	}
}

func validateUnsupportedResourceReferences(t *testing.T, metadataFile, resource string, references []string) {
	t.Helper()

	if len(references) == 0 {
		t.Fatalf("%s resource %q is missing references", metadataFile, resource)
	}
	for index, reference := range references {
		if strings.TrimSpace(reference) == "" {
			t.Fatalf("%s resource %q has empty references[%d]", metadataFile, resource, index)
		}
	}
}
