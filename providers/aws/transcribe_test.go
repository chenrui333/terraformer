// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	transcribetypes "github.com/aws/aws-sdk-go-v2/service/transcribe/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewTranscribeResources(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		importID     string
		attrKey      string
		build        func() (terraformutils.Resource, bool)
	}{
		{
			name:         "language model",
			resourceType: transcribeLanguageModelResourceType,
			importID:     "language-model",
			attrKey:      "model_name",
			build: func() (terraformutils.Resource, bool) {
				return newTranscribeLanguageModelResource(transcribetypes.LanguageModel{ModelName: aws.String("language-model"), ModelStatus: transcribetypes.ModelStatusCompleted})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, ok := tt.build()
			if !ok {
				t.Fatal("expected resource")
			}
			assertTranscribeResource(t, resource, tt.resourceType, tt.importID, tt.attrKey, tt.importID)
		})
	}
}

func TestTranscribeConstructorsSkipEmptyIdentifiers(t *testing.T) {
	tests := []struct {
		name  string
		build func() (terraformutils.Resource, bool)
	}{
		{
			name: "language model",
			build: func() (terraformutils.Resource, bool) {
				return newTranscribeLanguageModelResource(transcribetypes.LanguageModel{ModelStatus: transcribetypes.ModelStatusCompleted})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := tt.build(); ok {
				t.Fatal("expected empty identifier to be skipped")
			}
		})
	}
}

func TestTranscribeImportabilityPredicates(t *testing.T) {
	if !transcribeLanguageModelImportable(transcribetypes.ModelStatusCompleted) || transcribeLanguageModelImportable(transcribetypes.ModelStatusInProgress) {
		t.Fatal("unexpected language model importability")
	}
}

func TestTranscribeResourceNameUniqueness(t *testing.T) {
	first := transcribeResourceName("language-model", "ab", "c")
	second := transcribeResourceName("language-model", "a", "bc")
	if first == second {
		t.Fatalf("expected length-prefixed resource names to be unique, got %s", first)
	}
}

func TestTranscribeInitialCleanupScopesIDFilters(t *testing.T) {
	resource, ok := newTranscribeLanguageModelResource(transcribetypes.LanguageModel{
		ModelName:   aws.String("language-model"),
		ModelStatus: transcribetypes.ModelStatusCompleted,
	})
	if !ok {
		t.Fatal("expected resource")
	}

	g := TranscribeGenerator{}
	g.Resources = []terraformutils.Resource{resource}
	g.Filter = []terraformutils.ResourceFilter{
		{ServiceName: "kendra_index", FieldPath: "id", AcceptableValues: []string{"idx-123"}},
		{ServiceName: transcribeServiceName(transcribeLanguageModelResourceType), FieldPath: "id", AcceptableValues: []string{"language-model"}},
	}

	g.InitialCleanup()
	if len(g.Resources) != 1 {
		t.Fatalf("expected unrelated typed id filters to be ignored, got %d resources", len(g.Resources))
	}
}

func TestTranscribeInitialCleanupHonorsTypedAttributeFilters(t *testing.T) {
	resource, ok := newTranscribeLanguageModelResource(transcribetypes.LanguageModel{
		ModelName:   aws.String("language-model"),
		ModelStatus: transcribetypes.ModelStatusCompleted,
	})
	if !ok {
		t.Fatal("expected resource")
	}

	g := TranscribeGenerator{}
	g.Resources = []terraformutils.Resource{resource}
	g.Filter = []terraformutils.ResourceFilter{
		{ServiceName: transcribeServiceName(transcribeLanguageModelResourceType), FieldPath: "model_name", AcceptableValues: []string{"other-model"}},
	}

	g.InitialCleanup()
	if len(g.Resources) != 0 {
		t.Fatalf("expected non-matching typed attribute filter to remove resource, got %d resources", len(g.Resources))
	}
}

func TestTranscribeTypedUnsupportedFilterDoesNotLoadLanguageModels(t *testing.T) {
	resource, ok := newTranscribeLanguageModelResource(transcribetypes.LanguageModel{
		ModelName:   aws.String("language-model"),
		ModelStatus: transcribetypes.ModelStatusCompleted,
	})
	if !ok {
		t.Fatal("expected resource")
	}

	g := TranscribeGenerator{}
	g.Resources = []terraformutils.Resource{resource}
	g.Filter = []terraformutils.ResourceFilter{
		{ServiceName: "transcribe_vocabulary", FieldPath: "id", AcceptableValues: []string{"vocabulary"}},
	}

	if !g.hasTypedTranscribeFilter() {
		t.Fatal("expected unsupported transcribe typed filter to be recognized")
	}
	if g.shouldLoadTranscribeResource(transcribeServiceName(transcribeLanguageModelResourceType)) {
		t.Fatal("expected unsupported transcribe typed filter not to load language models")
	}

	g.InitialCleanup()
	if len(g.Resources) != 0 {
		t.Fatalf("expected unsupported typed filter to remove language model resource, got %d resources", len(g.Resources))
	}
}

func TestTranscribeUnsupportedResourceEntries(t *testing.T) {
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

	found := map[string]bool{
		"aws_transcribe_medical_vocabulary": false,
		"aws_transcribe_vocabulary":         false,
		"aws_transcribe_vocabulary_filter":  false,
	}
	resources := make([]string, 0, len(entries))
	for _, rawEntry := range entries {
		entry, ok := rawEntry.(map[string]interface{})
		if !ok {
			t.Fatalf("unsupported resource entry has unexpected type %T", rawEntry)
		}
		resource, _ := entry["resource"].(string)
		resources = append(resources, resource)
		if _, ok := found[resource]; !ok {
			continue
		}
		found[resource] = true
		if serviceFamily, _ := entry["service_family"].(string); serviceFamily != "transcribe" {
			t.Fatalf("%s service family = %q, want transcribe", resource, serviceFamily)
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
	for resource, ok := range found {
		if !ok {
			t.Fatalf("%s unsupported entry was not found", resource)
		}
	}
	if !sort.StringsAreSorted(resources) {
		t.Fatalf("unsupported resources are not sorted by resource: %v", resources)
	}
}

func TestTranscribeResourceNotFound(t *testing.T) {
	if !transcribeResourceNotFound(&transcribetypes.NotFoundException{}) {
		t.Fatal("expected NotFoundException to be detected")
	}
	if transcribeResourceNotFound(errors.New("other")) {
		t.Fatal("expected unrelated errors not to be detected")
	}
}

func assertTranscribeResource(t *testing.T, resource terraformutils.Resource, resourceType, importID, attrKey, attrValue string) {
	t.Helper()
	if resource.InstanceInfo.Type != resourceType {
		t.Fatalf("expected type %s, got %s", resourceType, resource.InstanceInfo.Type)
	}
	if resource.InstanceState.ID != importID {
		t.Fatalf("expected id %s, got %s", importID, resource.InstanceState.ID)
	}
	if got := resource.InstanceState.Attributes[attrKey]; got != attrValue {
		t.Fatalf("expected attribute %s=%s, got %s", attrKey, attrValue, got)
	}
}
