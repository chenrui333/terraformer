// SPDX-License-Identifier: Apache-2.0

package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const unsupportedResourcesVersion = 1

var allowedUnsupportedResourceStatuses = map[string]struct{}{
	"action-style":       {},
	"cloudflare-managed": {},
	"deferred":           {},
	"not-importable":     {},
	"policy-skip":        {},
	"request-style":      {},
	"runtime-data":       {},
	"runtime-generated":  {},
	"secret-required":    {},
	"unsupported":        {},
}

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
	resources := make([]string, 0, len(metadata.Resources))
	for index, entry := range metadata.Resources {
		resource := strings.TrimSpace(entry.Resource)
		if resource == "" {
			t.Fatalf("%s resources[%d] is missing resource", metadataFile, index)
		}
		if _, ok := seenResources[resource]; ok {
			t.Fatalf("%s contains duplicate resource %q", metadataFile, resource)
		}
		seenResources[resource] = struct{}{}
		resources = append(resources, resource)

		validateRequiredString(t, metadataFile, resource, "service_family", entry.ServiceFamily)
		validateRequiredString(t, metadataFile, resource, "reason", entry.Reason)
		validateRequiredString(t, metadataFile, resource, "evidence", entry.Evidence)
		validateUnsupportedResourceStatus(t, metadataFile, resource, entry.Status)
		validateUnsupportedResourceReferences(t, metadataFile, resource, entry.References)
	}
	if !sort.StringsAreSorted(resources) {
		t.Fatalf("%s resources are not sorted by resource: %v", metadataFile, resources)
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

	status = strings.TrimSpace(status)
	if status == "" {
		t.Fatalf("%s resource %q is missing status", metadataFile, resource)
	}
	if _, ok := allowedUnsupportedResourceStatuses[status]; !ok {
		t.Fatalf("%s resource %q has unsupported status %q, want one of %v", metadataFile, resource, status, sortedUnsupportedResourceStatuses())
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

func sortedUnsupportedResourceStatuses() []string {
	statuses := make([]string, 0, len(allowedUnsupportedResourceStatuses))
	for status := range allowedUnsupportedResourceStatuses {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)
	return statuses
}

func TestUnsupportedResourceStatusesAreDocumented(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "docs", "unsupported-resources.md"))
	if err != nil {
		t.Fatalf("read unsupported resources documentation: %v", err)
	}
	for _, status := range sortedUnsupportedResourceStatuses() {
		if !strings.Contains(string(data), fmt.Sprintf("`%s`", status)) {
			t.Fatalf("status %q is not documented", status)
		}
	}
}
