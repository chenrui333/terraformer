// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// ObservabilityPipelineAllowEmptyValues ...
	ObservabilityPipelineAllowEmptyValues = []string{
		"config.destination.elasticsearch.data_stream.auto_routing",
		"config.destination.elasticsearch.data_stream.sync_fields",
		"config.destination.elasticsearch.request_retry_partial",
		"config.destination.splunk_hec.auto_extract_timestamp",
		"config.processor_group.enabled",
		"config.processor_group.processor.custom_processor.remap.drop_on_error",
		"config.processor_group.processor.custom_processor.remap.enabled",
		"config.processor_group.processor.enabled",
		"config.processor_group.processor.enrichment_table.file.encoding.includes_headers",
		"config.processor_group.processor.ocsf_mapper.keep_unmatched",
		"config.processor_group.processor.parse_grok.disable_library_rules",
		"config.processor_group.processor.parse_xml.always_use_text_key",
		"config.processor_group.processor.parse_xml.include_attr",
		"config.processor_group.processor.parse_xml.parse_bool",
		"config.processor_group.processor.parse_xml.parse_null",
		"config.processor_group.processor.parse_xml.parse_number",
		"config.processor_group.processor.quota.drop_events",
		"config.processor_group.processor.quota.ignore_when_missing_partitions",
		"config.processor_group.processor.rename_fields.field.preserve_source",
		"config.processor_group.processor.sensitive_data_scanner.rule.pattern.library.use_recommended_keywords",
		"config.processor_group.processor.sensitive_data_scanner.rule.scope.all",
		"config.source.splunk_hec.store_hec_token",
		"config.use_legacy_search_syntax",
	}
)

// ObservabilityPipelineGenerator ...
type ObservabilityPipelineGenerator struct {
	DatadogService
}

func (g *ObservabilityPipelineGenerator) createResource(pipeline datadogV2.ObservabilityPipelineData) (terraformutils.Resource, error) {
	pipelineID := pipeline.GetId()
	if pipelineID == "" {
		return terraformutils.Resource{}, fmt.Errorf("observability pipeline missing id")
	}

	return terraformutils.NewSimpleResource(
		pipelineID,
		fmt.Sprintf("observability_pipeline_%s", pipelineID),
		"datadog_observability_pipeline",
		"datadog",
		ObservabilityPipelineAllowEmptyValues,
	), nil
}

func (g *ObservabilityPipelineGenerator) createResources(pipelines []datadogV2.ObservabilityPipelineData) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, pipeline := range pipelines {
		resource, err := g.createResource(pipeline)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each observability_pipeline create 1 TerraformResource.
// Need Observability Pipeline ID as ID for terraform resource.
func (g *ObservabilityPipelineGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewObservabilityPipelinesApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	pipelines, err := listObservabilityPipelines(auth, api)
	if err != nil {
		return err
	}
	g.Resources, err = g.createResources(pipelines)
	return err
}

func (g *ObservabilityPipelineGenerator) filteredResources(auth context.Context, api *datadogV2.ObservabilityPipelinesApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	matchedIDFilter := false

	for _, filter := range g.Filter {
		if filter.FieldPath != "id" {
			continue
		}
		if !filter.IsApplicable("observability_pipeline") {
			continue
		}
		matchedIDFilter = true
		for _, value := range filter.AcceptableValues {
			pipeline, httpResp, err := api.GetPipeline(auth, value)
			closeDatadogResponseBody(httpResp)
			if err != nil {
				return nil, false, err
			}
			resource, err := g.createResource(pipeline.GetData())
			if err != nil {
				return nil, false, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, matchedIDFilter, nil
}

func listObservabilityPipelines(auth context.Context, api *datadogV2.ObservabilityPipelinesApi) ([]datadogV2.ObservabilityPipelineData, error) {
	pageSize := int64(100)
	pageNumber := int64(0)
	pipelines := []datadogV2.ObservabilityPipelineData{}

	for {
		optionalParams := datadogV2.NewListPipelinesOptionalParameters().
			WithPageSize(pageSize).
			WithPageNumber(pageNumber)
		response, httpResp, err := api.ListPipelines(auth, *optionalParams)
		closeDatadogResponseBody(httpResp)
		if err != nil {
			return nil, err
		}

		data := response.GetData()
		pipelines = append(pipelines, data...)

		if meta, ok := response.GetMetaOk(); ok {
			if totalCount, ok := meta.GetTotalCountOk(); ok && int64(len(pipelines)) >= *totalCount {
				break
			}
		}
		if int64(len(data)) < pageSize {
			break
		}
		pageNumber++
	}
	return pipelines, nil
}
