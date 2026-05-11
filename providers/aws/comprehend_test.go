// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	comprehendtypes "github.com/aws/aws-sdk-go-v2/service/comprehend/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewComprehendResources(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		importID     string
		modelName    string
		build        func() (terraformutils.Resource, bool)
	}{
		{
			name:         "document classifier",
			resourceType: comprehendDocumentClassifierResourceType,
			importID:     "arn:aws:comprehend:us-east-1:123456789012:document-classifier/classifier-name",
			modelName:    "classifier-name",
			build: func() (terraformutils.Resource, bool) {
				return newComprehendDocumentClassifierResource(comprehendtypes.DocumentClassifierProperties{
					DocumentClassifierArn: aws.String("arn:aws:comprehend:us-east-1:123456789012:document-classifier/classifier-name"),
					Status:                comprehendtypes.ModelStatusTrained,
				})
			},
		},
		{
			name:         "entity recognizer",
			resourceType: comprehendEntityRecognizerResourceType,
			importID:     "arn:aws:comprehend:us-east-1:123456789012:entity-recognizer/recognizer-name/version/v1",
			modelName:    "recognizer-name",
			build: func() (terraformutils.Resource, bool) {
				return newComprehendEntityRecognizerResource(comprehendtypes.EntityRecognizerProperties{
					EntityRecognizerArn: aws.String("arn:aws:comprehend:us-east-1:123456789012:entity-recognizer/recognizer-name/version/v1"),
					Status:              comprehendtypes.ModelStatusTrainedWithWarning,
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, ok := tt.build()
			if !ok {
				t.Fatal("expected resource")
			}
			assertComprehendResource(t, resource, tt.resourceType, tt.importID, tt.modelName)
			if len(resource.IgnoreKeys) != 1 || resource.IgnoreKeys[0] != "^version_name_prefix$" {
				t.Fatalf("expected version_name_prefix ignore, got %#v", resource.IgnoreKeys)
			}
		})
	}
}

func TestComprehendConstructorsSkipEmptyIdentifiers(t *testing.T) {
	tests := []struct {
		name  string
		build func() (terraformutils.Resource, bool)
	}{
		{
			name: "document classifier",
			build: func() (terraformutils.Resource, bool) {
				return newComprehendDocumentClassifierResource(comprehendtypes.DocumentClassifierProperties{Status: comprehendtypes.ModelStatusTrained})
			},
		},
		{
			name: "entity recognizer",
			build: func() (terraformutils.Resource, bool) {
				return newComprehendEntityRecognizerResource(comprehendtypes.EntityRecognizerProperties{Status: comprehendtypes.ModelStatusTrained})
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

func TestComprehendModelImportable(t *testing.T) {
	if !comprehendModelImportable(comprehendtypes.ModelStatusTrained) ||
		!comprehendModelImportable(comprehendtypes.ModelStatusTrainedWithWarning) ||
		comprehendModelImportable(comprehendtypes.ModelStatusTraining) ||
		comprehendModelImportable(comprehendtypes.ModelStatusInError) {
		t.Fatal("unexpected model importability")
	}
}

func TestComprehendModelNameFromARN(t *testing.T) {
	tests := map[string]string{
		"arn:aws:comprehend:us-east-1:123456789012:document-classifier/classifier-name":            "classifier-name",
		"arn:aws:comprehend:us-east-1:123456789012:document-classifier/classifier-name/version/v1": "classifier-name",
		"arn:aws:comprehend:us-east-1:123456789012:entity-recognizer/recognizer-name/version/v1":   "recognizer-name",
	}
	for input, expected := range tests {
		if got := comprehendModelNameFromARN(input, "document-classifier"); got != expected && expected == "classifier-name" {
			t.Fatalf("expected %s, got %s", expected, got)
		}
		if got := comprehendModelNameFromARN(input, "entity-recognizer"); got != expected && expected == "recognizer-name" {
			t.Fatalf("expected %s, got %s", expected, got)
		}
	}
}

func TestComprehendResourceNameUniqueness(t *testing.T) {
	first := comprehendResourceName("document-classifier", "ab", "c")
	second := comprehendResourceName("document-classifier", "a", "bc")
	if first == second {
		t.Fatalf("expected length-prefixed resource names to be unique, got %s", first)
	}
}

func TestComprehendInitialCleanupScopesIDFilters(t *testing.T) {
	classifierARN := "arn:aws:comprehend:us-east-1:123456789012:document-classifier/classifier-name"
	resource, ok := newComprehendDocumentClassifierResource(comprehendtypes.DocumentClassifierProperties{
		DocumentClassifierArn: aws.String(classifierARN),
		Status:                comprehendtypes.ModelStatusTrained,
	})
	if !ok {
		t.Fatal("expected resource")
	}

	g := ComprehendGenerator{}
	g.Resources = []terraformutils.Resource{resource}
	g.Filter = []terraformutils.ResourceFilter{
		{ServiceName: "kendra_index", FieldPath: "id", AcceptableValues: []string{"idx-123"}},
		{ServiceName: comprehendServiceName(comprehendDocumentClassifierResourceType), FieldPath: "id", AcceptableValues: []string{classifierARN}},
	}

	g.InitialCleanup()
	if len(g.Resources) != 1 {
		t.Fatalf("expected unrelated typed id filters to be ignored, got %d resources", len(g.Resources))
	}
}

func TestComprehendResourceNotFound(t *testing.T) {
	if !comprehendResourceNotFound(&comprehendtypes.ResourceNotFoundException{}) {
		t.Fatal("expected ResourceNotFoundException to be detected")
	}
	if comprehendResourceNotFound(errors.New("other")) {
		t.Fatal("expected unrelated errors not to be detected")
	}
}

func assertComprehendResource(t *testing.T, resource terraformutils.Resource, resourceType, importID, name string) {
	t.Helper()
	if resource.InstanceInfo.Type != resourceType {
		t.Fatalf("expected type %s, got %s", resourceType, resource.InstanceInfo.Type)
	}
	if resource.InstanceState.ID != importID {
		t.Fatalf("expected id %s, got %s", importID, resource.InstanceState.ID)
	}
	if got := resource.InstanceState.Attributes["arn"]; got != importID {
		t.Fatalf("expected arn %s, got %s", importID, got)
	}
	if got := resource.InstanceState.Attributes["name"]; got != name {
		t.Fatalf("expected name %s, got %s", name, got)
	}
}
