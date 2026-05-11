// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	kendratypes "github.com/aws/aws-sdk-go-v2/service/kendra/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestKendraChildImportID(t *testing.T) {
	got := kendraChildImportID("child-id", "index-id")
	if got != "child-id/index-id" {
		t.Fatalf("expected child-id/index-id, got %s", got)
	}
}

func TestNewKendraIndexResource(t *testing.T) {
	resource, ok := newKendraIndexResource(kendratypes.IndexConfigurationSummary{
		Id:     aws.String("idx-123"),
		Name:   aws.String("search"),
		Status: kendratypes.IndexStatusActive,
	})
	if !ok {
		t.Fatal("expected resource")
	}
	assertKendraResource(t, resource, kendraIndexResourceType, "idx-123", map[string]string{
		"id":   "idx-123",
		"name": "search",
	})
}

func TestNewKendraChildResources(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		expectedID   string
		expectedAttr string
		build        func() (terraformutils.Resource, bool)
	}{
		{
			name:         "data source",
			resourceType: kendraDataSourceResourceType,
			expectedID:   "ds-123/idx-123",
			expectedAttr: "data_source_id",
			build: func() (terraformutils.Resource, bool) {
				return newKendraDataSourceResource("idx-123", kendratypes.DataSourceSummary{Id: aws.String("ds-123"), Name: aws.String("docs"), Status: kendratypes.DataSourceStatusActive})
			},
		},
		{
			name:         "faq",
			resourceType: kendraFaqResourceType,
			expectedID:   "faq-123/idx-123",
			expectedAttr: "faq_id",
			build: func() (terraformutils.Resource, bool) {
				return newKendraFaqResource("idx-123", kendratypes.FaqSummary{Id: aws.String("faq-123"), Name: aws.String("answers"), Status: kendratypes.FaqStatusActive})
			},
		},
		{
			name:         "query suggestions block list",
			resourceType: kendraQuerySuggestionsBlockListResourceType,
			expectedID:   "block-123/idx-123",
			expectedAttr: "query_suggestions_block_list_id",
			build: func() (terraformutils.Resource, bool) {
				return newKendraQuerySuggestionsBlockListResource("idx-123", kendratypes.QuerySuggestionsBlockListSummary{Id: aws.String("block-123"), Name: aws.String("blocked"), Status: kendratypes.QuerySuggestionsBlockListStatusActive})
			},
		},
		{
			name:         "thesaurus",
			resourceType: kendraThesaurusResourceType,
			expectedID:   "th-123/idx-123",
			expectedAttr: "thesaurus_id",
			build: func() (terraformutils.Resource, bool) {
				return newKendraThesaurusResource("idx-123", kendratypes.ThesaurusSummary{Id: aws.String("th-123"), Name: aws.String("synonyms"), Status: kendratypes.ThesaurusStatusActive})
			},
		},
		{
			name:         "experience",
			resourceType: kendraExperienceResourceType,
			expectedID:   "exp-123/idx-123",
			expectedAttr: "experience_id",
			build: func() (terraformutils.Resource, bool) {
				return newKendraExperienceResource("idx-123", kendratypes.ExperiencesSummary{Id: aws.String("exp-123"), Name: aws.String("portal"), Status: kendratypes.ExperienceStatusActive})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, ok := tt.build()
			if !ok {
				t.Fatal("expected resource")
			}
			assertKendraResource(t, resource, tt.resourceType, tt.expectedID, map[string]string{
				tt.expectedAttr: tt.expectedID[:len(tt.expectedID)-len("/idx-123")],
				"index_id":      "idx-123",
			})
		})
	}
}

func TestKendraConstructorsSkipEmptyIdentifiers(t *testing.T) {
	tests := []struct {
		name  string
		build func() (terraformutils.Resource, bool)
	}{
		{
			name: "index",
			build: func() (terraformutils.Resource, bool) {
				return newKendraIndexResource(kendratypes.IndexConfigurationSummary{Status: kendratypes.IndexStatusActive})
			},
		},
		{
			name: "data source",
			build: func() (terraformutils.Resource, bool) {
				return newKendraDataSourceResource("idx-123", kendratypes.DataSourceSummary{Status: kendratypes.DataSourceStatusActive})
			},
		},
		{
			name: "faq",
			build: func() (terraformutils.Resource, bool) {
				return newKendraFaqResource("idx-123", kendratypes.FaqSummary{Status: kendratypes.FaqStatusActive})
			},
		},
		{
			name: "query suggestions block list",
			build: func() (terraformutils.Resource, bool) {
				return newKendraQuerySuggestionsBlockListResource("idx-123", kendratypes.QuerySuggestionsBlockListSummary{Status: kendratypes.QuerySuggestionsBlockListStatusActive})
			},
		},
		{
			name: "thesaurus",
			build: func() (terraformutils.Resource, bool) {
				return newKendraThesaurusResource("idx-123", kendratypes.ThesaurusSummary{Status: kendratypes.ThesaurusStatusActive})
			},
		},
		{
			name: "experience",
			build: func() (terraformutils.Resource, bool) {
				return newKendraExperienceResource("idx-123", kendratypes.ExperiencesSummary{Status: kendratypes.ExperienceStatusActive})
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

func TestKendraImportabilityPredicates(t *testing.T) {
	if !kendraIndexImportable(kendratypes.IndexStatusActive) || kendraIndexImportable(kendratypes.IndexStatusUpdating) {
		t.Fatal("unexpected index importability")
	}
	if !kendraDataSourceImportable(kendratypes.DataSourceStatusActive) || kendraDataSourceImportable(kendratypes.DataSourceStatusFailed) {
		t.Fatal("unexpected data source importability")
	}
	if !kendraFaqImportable(kendratypes.FaqStatusActive) || kendraFaqImportable(kendratypes.FaqStatusDeleting) {
		t.Fatal("unexpected faq importability")
	}
	if !kendraQuerySuggestionsBlockListImportable(kendratypes.QuerySuggestionsBlockListStatusActive) || kendraQuerySuggestionsBlockListImportable(kendratypes.QuerySuggestionsBlockListStatusActiveButUpdateFailed) {
		t.Fatal("unexpected block list importability")
	}
	if !kendraThesaurusImportable(kendratypes.ThesaurusStatusActive) || kendraThesaurusImportable(kendratypes.ThesaurusStatusActiveButUpdateFailed) {
		t.Fatal("unexpected thesaurus importability")
	}
	if !kendraExperienceImportable(kendratypes.ExperienceStatusActive) || kendraExperienceImportable(kendratypes.ExperienceStatusFailed) {
		t.Fatal("unexpected experience importability")
	}
}

func TestKendraResourceNameUniqueness(t *testing.T) {
	first := kendraResourceName("data-source", "idx-1", "ab", "c")
	second := kendraResourceName("data-source", "idx-1", "a", "bc")
	if first == second {
		t.Fatalf("expected length-prefixed resource names to be unique, got %s", first)
	}
}

func TestKendraInitialCleanupScopesIDFilters(t *testing.T) {
	resource, ok := newKendraIndexResource(kendratypes.IndexConfigurationSummary{
		Id:     aws.String("idx-123"),
		Name:   aws.String("search"),
		Status: kendratypes.IndexStatusActive,
	})
	if !ok {
		t.Fatal("expected resource")
	}

	g := KendraGenerator{}
	g.Resources = []terraformutils.Resource{resource}
	g.Filter = []terraformutils.ResourceFilter{
		{ServiceName: "transcribe_vocabulary", FieldPath: "id", AcceptableValues: []string{"other-id"}},
		{ServiceName: kendraServiceName(kendraIndexResourceType), FieldPath: "id", AcceptableValues: []string{"idx-123"}},
	}

	g.InitialCleanup()
	if len(g.Resources) != 1 {
		t.Fatalf("expected unrelated typed id filters to be ignored, got %d resources", len(g.Resources))
	}
}

func TestKendraResourceNotFound(t *testing.T) {
	if !kendraResourceNotFound(&kendratypes.ResourceNotFoundException{}) {
		t.Fatal("expected ResourceNotFoundException to be detected")
	}
	if kendraResourceNotFound(errors.New("other")) {
		t.Fatal("expected unrelated errors not to be detected")
	}
}

func assertKendraResource(t *testing.T, resource terraformutils.Resource, resourceType, importID string, expectedAttributes map[string]string) {
	t.Helper()
	if resource.InstanceInfo.Type != resourceType {
		t.Fatalf("expected type %s, got %s", resourceType, resource.InstanceInfo.Type)
	}
	if resource.InstanceState.ID != importID {
		t.Fatalf("expected id %s, got %s", importID, resource.InstanceState.ID)
	}
	for key, value := range expectedAttributes {
		if got := resource.InstanceState.Attributes[key]; got != value {
			t.Fatalf("expected attribute %s=%s, got %s", key, value, got)
		}
	}
}
