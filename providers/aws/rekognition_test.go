// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	rekognitiontypes "github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewRekognitionResources(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		importID     string
		attrKey      string
		attrValue    string
		build        func() (terraformutils.Resource, bool)
	}{
		{
			name:         "collection",
			resourceType: rekognitionCollectionResourceType,
			importID:     "collection-id",
			attrKey:      "collection_id",
			attrValue:    "collection-id",
			build: func() (terraformutils.Resource, bool) {
				return newRekognitionCollectionResource("collection-id")
			},
		},
		{
			name:         "project",
			resourceType: rekognitionProjectResourceType,
			importID:     "project-name",
			attrKey:      "name",
			attrValue:    "project-name",
			build: func() (terraformutils.Resource, bool) {
				return newRekognitionProjectResource(rekognitiontypes.ProjectDescription{
					ProjectArn: aws.String("arn:aws:rekognition:us-east-1:123456789012:project/project-name/1234567890"),
					Feature:    rekognitiontypes.CustomizationFeatureCustomLabels,
					Status:     rekognitiontypes.ProjectStatusCreated,
				})
			},
		},
		{
			name:         "stream processor",
			resourceType: rekognitionStreamProcessorResourceType,
			importID:     "processor-name",
			attrKey:      "name",
			attrValue:    "processor-name",
			build: func() (terraformutils.Resource, bool) {
				return newRekognitionStreamProcessorResource(rekognitiontypes.StreamProcessor{Name: aws.String("processor-name"), Status: rekognitiontypes.StreamProcessorStatusStopped})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, ok := tt.build()
			if !ok {
				t.Fatal("expected resource")
			}
			assertRekognitionResource(t, resource, tt.resourceType, tt.importID, tt.attrKey, tt.attrValue)
		})
	}
}

func TestRekognitionConstructorsSkipEmptyIdentifiers(t *testing.T) {
	tests := []struct {
		name  string
		build func() (terraformutils.Resource, bool)
	}{
		{
			name: "collection",
			build: func() (terraformutils.Resource, bool) {
				return newRekognitionCollectionResource("")
			},
		},
		{
			name: "project",
			build: func() (terraformutils.Resource, bool) {
				return newRekognitionProjectResource(rekognitiontypes.ProjectDescription{Status: rekognitiontypes.ProjectStatusCreated})
			},
		},
		{
			name: "stream processor",
			build: func() (terraformutils.Resource, bool) {
				return newRekognitionStreamProcessorResource(rekognitiontypes.StreamProcessor{Status: rekognitiontypes.StreamProcessorStatusStopped})
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

func TestRekognitionImportabilityPredicates(t *testing.T) {
	if !rekognitionProjectImportable(rekognitiontypes.ProjectStatusCreated) || rekognitionProjectImportable(rekognitiontypes.ProjectStatusCreating) {
		t.Fatal("unexpected project importability")
	}
	if !rekognitionStreamProcessorImportable(rekognitiontypes.StreamProcessorStatusRunning) ||
		!rekognitionStreamProcessorImportable(rekognitiontypes.StreamProcessorStatusStopped) ||
		rekognitionStreamProcessorImportable(rekognitiontypes.StreamProcessorStatusFailed) {
		t.Fatal("unexpected stream processor importability")
	}
}

func TestRekognitionProjectNameFromARN(t *testing.T) {
	got := rekognitionProjectNameFromARN("arn:aws:rekognition:us-east-1:123456789012:project/project-name/1234567890")
	if got != "project-name" {
		t.Fatalf("expected project-name, got %s", got)
	}
	if got := rekognitionProjectNameFromARN("not-an-arn"); got != "" {
		t.Fatalf("expected empty name for malformed ARN, got %s", got)
	}
}

func TestRekognitionResourceNameUniqueness(t *testing.T) {
	first := rekognitionResourceName("project", "ab", "c")
	second := rekognitionResourceName("project", "a", "bc")
	if first == second {
		t.Fatalf("expected length-prefixed resource names to be unique, got %s", first)
	}
}

func TestRekognitionInitialCleanupScopesIDFilters(t *testing.T) {
	resource, ok := newRekognitionCollectionResource("collection-id")
	if !ok {
		t.Fatal("expected resource")
	}

	g := RekognitionGenerator{}
	g.Resources = []terraformutils.Resource{resource}
	g.Filter = []terraformutils.ResourceFilter{
		{ServiceName: "kendra_index", FieldPath: "id", AcceptableValues: []string{"idx-123"}},
		{ServiceName: rekognitionServiceName(rekognitionCollectionResourceType), FieldPath: "id", AcceptableValues: []string{"collection-id"}},
	}

	g.InitialCleanup()
	if len(g.Resources) != 1 {
		t.Fatalf("expected unrelated typed id filters to be ignored, got %d resources", len(g.Resources))
	}
}

func TestRekognitionResourceNotFound(t *testing.T) {
	if !rekognitionResourceNotFound(&rekognitiontypes.ResourceNotFoundException{}) {
		t.Fatal("expected ResourceNotFoundException to be detected")
	}
	if rekognitionResourceNotFound(errors.New("other")) {
		t.Fatal("expected unrelated errors not to be detected")
	}
}

func assertRekognitionResource(t *testing.T, resource terraformutils.Resource, resourceType, importID, attrKey, attrValue string) {
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
