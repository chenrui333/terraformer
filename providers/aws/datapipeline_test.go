// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/datapipeline"
	datapipelinetypes "github.com/aws/aws-sdk-go-v2/service/datapipeline/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewDataPipelinePipelineResource(t *testing.T) {
	resource, ok := newDataPipelinePipelineResource("df-1234567890ABC", "daily-import")
	assertDataPipelineResource(t, resource, ok, "df-1234567890ABC", "daily-import", dataPipelinePipelineResourceType)

	if _, ok := newDataPipelinePipelineResource("", "missing-id"); ok {
		t.Fatal("pipeline with empty ID should be skipped")
	}
}

func TestNewDataPipelinePipelineDefinitionResource(t *testing.T) {
	resource := newDataPipelinePipelineDefinitionResource("df-1234567890ABC", "daily-import")
	assertDataPipelineResource(t, resource, true, "df-1234567890ABC", dataPipelineResourceName("pipeline-definition", "daily-import", "df-1234567890ABC"), dataPipelinePipelineDefinitionResourceType)
	assertDataPipelineAttribute(t, resource, "name", "daily-import")
	assertDataPipelineAttribute(t, resource, "pipeline_id", "df-1234567890ABC")
}

func TestDataPipelineDefinitionImportable(t *testing.T) {
	tests := []struct {
		name       string
		definition *datapipeline.GetPipelineDefinitionOutput
		want       bool
	}{
		{name: "nil", want: false},
		{name: "empty", definition: &datapipeline.GetPipelineDefinitionOutput{}, want: false},
		{name: "with objects", definition: &datapipeline.GetPipelineDefinitionOutput{
			PipelineObjects: []datapipelinetypes.PipelineObject{{Id: aws.String("Default")}},
		}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dataPipelineDefinitionImportable(tt.definition); got != tt.want {
				t.Fatalf("dataPipelineDefinitionImportable(%#v) = %t, want %t", tt.definition, got, tt.want)
			}
		})
	}
}

func TestDataPipelineImportIDs(t *testing.T) {
	if got, want := dataPipelinePipelineImportID("df-123"), "df-123"; got != want {
		t.Fatalf("Data Pipeline import ID = %q, want %q", got, want)
	}
	if got, want := dataPipelinePipelineDefinitionImportID("df-123"), "df-123"; got != want {
		t.Fatalf("Data Pipeline definition import ID = %q, want %q", got, want)
	}
}

func TestDataPipelineResourceNamesPreserveSegmentBoundaries(t *testing.T) {
	left := terraformutils.TfSanitize(dataPipelineResourceName("pipeline-definition", "a/b_c"))
	right := terraformutils.TfSanitize(dataPipelineResourceName("pipeline", "definition/a_b_c"))
	if left == right {
		t.Fatalf("Data Pipeline resource names collide: %q", left)
	}
}

func assertDataPipelineResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantName, wantType string) {
	t.Helper()
	if !ok {
		t.Fatal("resource was skipped")
	}
	if got := resource.InstanceState.ID; got != wantID {
		t.Fatalf("resource ID = %q, want %q", got, wantID)
	}
	if got := resource.ResourceName; got != terraformutils.TfSanitize(wantName) {
		t.Fatalf("resource name = %q, want %q", got, terraformutils.TfSanitize(wantName))
	}
	if got := resource.InstanceInfo.Type; got != wantType {
		t.Fatalf("resource type = %q, want %q", got, wantType)
	}
}

func assertDataPipelineAttribute(t *testing.T, resource terraformutils.Resource, key, want string) {
	t.Helper()
	if got := resource.InstanceState.Attributes[key]; got != want {
		t.Fatalf("resource attribute %q = %q, want %q", key, got, want)
	}
}
