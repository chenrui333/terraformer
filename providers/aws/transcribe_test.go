// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
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
		{
			name:         "medical vocabulary",
			resourceType: transcribeMedicalVocabularyResourceType,
			importID:     "medical-vocabulary",
			attrKey:      "vocabulary_name",
			build: func() (terraformutils.Resource, bool) {
				return newTranscribeMedicalVocabularyResource(transcribetypes.VocabularyInfo{VocabularyName: aws.String("medical-vocabulary"), VocabularyState: transcribetypes.VocabularyStateReady})
			},
		},
		{
			name:         "vocabulary",
			resourceType: transcribeVocabularyResourceType,
			importID:     "vocabulary",
			attrKey:      "vocabulary_name",
			build: func() (terraformutils.Resource, bool) {
				return newTranscribeVocabularyResource(transcribetypes.VocabularyInfo{VocabularyName: aws.String("vocabulary"), VocabularyState: transcribetypes.VocabularyStateReady})
			},
		},
		{
			name:         "vocabulary filter",
			resourceType: transcribeVocabularyFilterResourceType,
			importID:     "vocabulary-filter",
			attrKey:      "vocabulary_filter_name",
			build: func() (terraformutils.Resource, bool) {
				return newTranscribeVocabularyFilterResource(transcribetypes.VocabularyFilterInfo{VocabularyFilterName: aws.String("vocabulary-filter")})
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
		{
			name: "medical vocabulary",
			build: func() (terraformutils.Resource, bool) {
				return newTranscribeMedicalVocabularyResource(transcribetypes.VocabularyInfo{VocabularyState: transcribetypes.VocabularyStateReady})
			},
		},
		{
			name: "vocabulary",
			build: func() (terraformutils.Resource, bool) {
				return newTranscribeVocabularyResource(transcribetypes.VocabularyInfo{VocabularyState: transcribetypes.VocabularyStateReady})
			},
		},
		{
			name: "vocabulary filter",
			build: func() (terraformutils.Resource, bool) {
				return newTranscribeVocabularyFilterResource(transcribetypes.VocabularyFilterInfo{})
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
	if !transcribeVocabularyImportable(transcribetypes.VocabularyStateReady) || transcribeVocabularyImportable(transcribetypes.VocabularyStateFailed) {
		t.Fatal("unexpected vocabulary importability")
	}
}

func TestTranscribeResourceNameUniqueness(t *testing.T) {
	first := transcribeResourceName("vocabulary", "ab", "c")
	second := transcribeResourceName("vocabulary", "a", "bc")
	if first == second {
		t.Fatalf("expected length-prefixed resource names to be unique, got %s", first)
	}
}

func TestTranscribeInitialCleanupScopesIDFilters(t *testing.T) {
	resource, ok := newTranscribeVocabularyResource(transcribetypes.VocabularyInfo{
		VocabularyName:  aws.String("vocabulary"),
		VocabularyState: transcribetypes.VocabularyStateReady,
	})
	if !ok {
		t.Fatal("expected resource")
	}

	g := TranscribeGenerator{}
	g.Resources = []terraformutils.Resource{resource}
	g.Filter = []terraformutils.ResourceFilter{
		{ServiceName: "kendra_index", FieldPath: "id", AcceptableValues: []string{"idx-123"}},
		{ServiceName: transcribeServiceName(transcribeVocabularyResourceType), FieldPath: "id", AcceptableValues: []string{"vocabulary"}},
	}

	g.InitialCleanup()
	if len(g.Resources) != 1 {
		t.Fatalf("expected unrelated typed id filters to be ignored, got %d resources", len(g.Resources))
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
