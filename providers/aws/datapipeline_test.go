// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"reflect"
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

func TestDataPipelinePipelineIDFilterIncludesDefinitionIDs(t *testing.T) {
	service := terraformutils.Service{}
	service.ParseFilters([]string{
		"datapipeline_pipeline=df-parent",
		"datapipeline_pipeline_definition=df-child",
		"Type=datapipeline_pipeline_definition;Name=pipeline_id;Value=df-grandchild",
	})

	filter := dataPipelinePipelineIDFilter(service.Filter)
	for _, pipelineID := range []string{"df-parent", "df-child", "df-grandchild"} {
		if !awsIDFilterAllows(filter, pipelineID) {
			t.Fatalf("Data Pipeline filter should allow %q: %#v", pipelineID, filter)
		}
	}
	if awsIDFilterAllows(filter, "df-other") {
		t.Fatalf("Data Pipeline filter allowed unrelated pipeline: %#v", filter)
	}
}

func TestDataPipelinePipelineIDFilterAllowsAllForPipelineAttributeFilters(t *testing.T) {
	service := terraformutils.Service{}
	service.ParseFilters([]string{
		"Type=datapipeline_pipeline;Name=name;Value=daily-import",
		"Type=datapipeline_pipeline_definition;Name=pipeline_id;Value=df-456",
	})

	filter := dataPipelinePipelineIDFilter(service.Filter)
	if !awsIDFilterAllows(filter, "df-123") {
		t.Fatalf("Data Pipeline pipeline attribute filter should disable prefilter: %#v", filter)
	}
}

func TestDataPipelineShouldEmitPipelineSkipsDefinitionOnlyFilters(t *testing.T) {
	service := terraformutils.Service{}
	service.ParseFilters([]string{"Type=datapipeline_pipeline_definition;Name=pipeline_id;Value=df-123"})

	if dataPipelineShouldEmitPipeline(service.Filter, "df-123") {
		t.Fatal("definition-only filter should scan but not emit parent pipeline")
	}
}

func TestDataPipelineShouldEmitPipelineHonorsPipelineFilters(t *testing.T) {
	service := terraformutils.Service{}
	service.ParseFilters([]string{
		"datapipeline_pipeline=df-123",
		"Type=datapipeline_pipeline_definition;Name=pipeline_id;Value=df-456",
	})

	if !dataPipelineShouldEmitPipeline(service.Filter, "df-123") {
		t.Fatal("pipeline filter should emit matching pipeline")
	}
	if dataPipelineShouldEmitPipeline(service.Filter, "df-456") {
		t.Fatal("definition-derived parent should not be emitted when it is not requested by the pipeline filter")
	}
}

func TestDataPipelinePostRefreshCleanupKeepsDefinitionForMatchedPipelineName(t *testing.T) {
	pipeline := dataPipelinePipelineResourceForCleanup("df-123", "daily-import")
	otherPipeline := dataPipelinePipelineResourceForCleanup("df-456", "hourly-import")
	definition := dataPipelineDefinitionResourceForCleanup("df-123", "daily-import")
	otherDefinition := dataPipelineDefinitionResourceForCleanup("df-456", "hourly-import")
	generator := &DataPipelineGenerator{}
	generator.Resources = []terraformutils.Resource{pipeline, definition, otherPipeline, otherDefinition}
	generator.ParseFilters([]string{"Name=name;Value=daily-import"})

	generator.PostRefreshCleanup()

	assertDataPipelineResourceIDs(t, generator.Resources, []string{
		dataPipelinePipelineResourceType + "/df-123",
		dataPipelinePipelineDefinitionResourceType + "/df-123",
	})
}

func TestDataPipelinePostRefreshCleanupPrunesDefinitionsForTypedPipelineNameFilter(t *testing.T) {
	pipeline := dataPipelinePipelineResourceForCleanup("df-123", "daily-import")
	otherPipeline := dataPipelinePipelineResourceForCleanup("df-456", "hourly-import")
	definition := dataPipelineDefinitionResourceForCleanup("df-123", "daily-import")
	otherDefinition := dataPipelineDefinitionResourceForCleanup("df-456", "hourly-import")
	generator := &DataPipelineGenerator{}
	generator.Resources = []terraformutils.Resource{pipeline, definition, otherPipeline, otherDefinition}
	generator.ParseFilters([]string{"Type=datapipeline_pipeline;Name=name;Value=daily-import"})

	generator.PostRefreshCleanup()

	assertDataPipelineResourceIDs(t, generator.Resources, []string{
		dataPipelinePipelineResourceType + "/df-123",
		dataPipelinePipelineDefinitionResourceType + "/df-123",
	})
}

func TestDataPipelinePostRefreshCleanupPreservesExplicitDefinitionFilters(t *testing.T) {
	pipeline := dataPipelinePipelineResourceForCleanup("df-123", "daily-import")
	otherPipeline := dataPipelinePipelineResourceForCleanup("df-456", "hourly-import")
	definition := dataPipelineDefinitionResourceForCleanup("df-123", "daily-import")
	otherDefinition := dataPipelineDefinitionResourceForCleanup("df-456", "hourly-import")
	generator := &DataPipelineGenerator{}
	generator.Resources = []terraformutils.Resource{pipeline, definition, otherPipeline, otherDefinition}
	generator.ParseFilters([]string{
		"Type=datapipeline_pipeline;Name=name;Value=daily-import",
		"Type=datapipeline_pipeline_definition;Name=pipeline_id;Value=df-456",
	})

	generator.PostRefreshCleanup()

	assertDataPipelineResourceIDs(t, generator.Resources, []string{
		dataPipelinePipelineResourceType + "/df-123",
		dataPipelinePipelineDefinitionResourceType + "/df-456",
	})
}

func TestDataPipelinePostRefreshCleanupDoesNotBroadenDefinitionSpecificFilters(t *testing.T) {
	pipeline := dataPipelinePipelineResourceForCleanup("df-123", "daily-import")
	otherPipeline := dataPipelinePipelineResourceForCleanup("df-456", "hourly-import")
	definition := dataPipelineDefinitionResourceForCleanup("df-123", "daily-import")
	otherDefinition := dataPipelineDefinitionResourceForCleanup("df-456", "hourly-import")
	generator := &DataPipelineGenerator{}
	generator.Resources = []terraformutils.Resource{pipeline, definition, otherPipeline, otherDefinition}
	generator.ParseFilters([]string{"Type=datapipeline_pipeline_definition;Name=pipeline_id;Value=df-123"})

	generator.PostRefreshCleanup()

	assertDataPipelineResourceIDs(t, generator.Resources, []string{
		dataPipelinePipelineDefinitionResourceType + "/df-123",
	})
}

func dataPipelinePipelineResourceForCleanup(pipelineID, pipelineName string) terraformutils.Resource {
	resource, ok := newDataPipelinePipelineResource(pipelineID, pipelineName)
	if !ok {
		panic("expected Data Pipeline pipeline resource")
	}
	resource.InstanceState.Attributes = map[string]string{"name": pipelineName}
	return resource
}

func dataPipelineDefinitionResourceForCleanup(pipelineID, pipelineName string) terraformutils.Resource {
	resource := newDataPipelinePipelineDefinitionResource(pipelineID, pipelineName)
	resource.InstanceState.Attributes = map[string]string{"pipeline_id": pipelineID}
	return resource
}

func assertDataPipelineResourceIDs(t *testing.T, resources []terraformutils.Resource, want []string) {
	t.Helper()
	if len(resources) != len(want) {
		t.Fatalf("resources len = %d, want %d: %#v", len(resources), len(want), resources)
	}
	got := make([]string, 0, len(resources))
	for _, resource := range resources {
		got = append(got, resource.InstanceInfo.Type+"/"+resource.InstanceState.ID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resources = %v, want %v", got, want)
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
