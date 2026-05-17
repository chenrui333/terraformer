// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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

	observabilityPipelineEmptyVariantBlockPaths = map[string]struct{}{
		"config.destination.amazon_opensearch":                                       {},
		"config.destination.amazon_s3":                                               {},
		"config.destination.amazon_s3_generic":                                       {},
		"config.destination.amazon_security_lake":                                    {},
		"config.destination.azure_storage":                                           {},
		"config.destination.cloud_prem":                                              {},
		"config.destination.crowdstrike_next_gen_siem":                               {},
		"config.destination.databricks_zerobus":                                      {},
		"config.destination.datadog_logs":                                            {},
		"config.destination.datadog_metrics":                                         {},
		"config.destination.elasticsearch":                                           {},
		"config.destination.google_cloud_storage":                                    {},
		"config.destination.google_pubsub":                                           {},
		"config.destination.google_secops":                                           {},
		"config.destination.http_client":                                             {},
		"config.destination.kafka":                                                   {},
		"config.destination.microsoft_sentinel":                                      {},
		"config.destination.new_relic":                                               {},
		"config.destination.opensearch":                                              {},
		"config.destination.rsyslog":                                                 {},
		"config.destination.sentinel_one":                                            {},
		"config.destination.socket":                                                  {},
		"config.destination.splunk_hec":                                              {},
		"config.destination.sumo_logic":                                              {},
		"config.destination.syslog_ng":                                               {},
		"config.processor_group.processor.add_env_vars":                              {},
		"config.processor_group.processor.add_fields":                                {},
		"config.processor_group.processor.add_hostname":                              {},
		"config.processor_group.processor.custom_processor":                          {},
		"config.processor_group.processor.datadog_tags":                              {},
		"config.processor_group.processor.dedupe":                                    {},
		"config.processor_group.processor.enrichment_table":                          {},
		"config.processor_group.processor.filter":                                    {},
		"config.processor_group.processor.generate_datadog_metrics":                  {},
		"config.processor_group.processor.metric_tags":                               {},
		"config.processor_group.processor.ocsf_mapper":                               {},
		"config.processor_group.processor.parse_grok":                                {},
		"config.processor_group.processor.parse_json":                                {},
		"config.processor_group.processor.parse_xml":                                 {},
		"config.processor_group.processor.quota":                                     {},
		"config.processor_group.processor.reduce":                                    {},
		"config.processor_group.processor.remove_fields":                             {},
		"config.processor_group.processor.rename_fields":                             {},
		"config.processor_group.processor.sample":                                    {},
		"config.processor_group.processor.sensitive_data_scanner":                    {},
		"config.processor_group.processor.sensitive_data_scanner.rule.on_match.hash": {},
		"config.processor_group.processor.split_array":                               {},
		"config.processor_group.processor.throttle":                                  {},
		"config.source.amazon_data_firehose":                                         {},
		"config.source.amazon_s3":                                                    {},
		"config.source.datadog_agent":                                                {},
		"config.source.fluent_bit":                                                   {},
		"config.source.fluentd":                                                      {},
		"config.source.google_pubsub":                                                {},
		"config.source.http_client":                                                  {},
		"config.source.http_server":                                                  {},
		"config.source.kafka":                                                        {},
		"config.source.logstash":                                                     {},
		"config.source.opentelemetry":                                                {},
		"config.source.rsyslog":                                                      {},
		"config.source.socket":                                                       {},
		"config.source.splunk_hec":                                                   {},
		"config.source.splunk_tcp":                                                   {},
		"config.source.sumo_logic":                                                   {},
		"config.source.syslog_ng":                                                    {},
	}
)

// ObservabilityPipelineGenerator ...
type ObservabilityPipelineGenerator struct {
	DatadogService
}

func (g *ObservabilityPipelineGenerator) PostConvertHook() error {
	for i := range g.Resources {
		preserveObservabilityPipelineEmptyVariantBlocks(&g.Resources[i])
	}
	return nil
}

func preserveObservabilityPipelineEmptyVariantBlocks(resource *terraformutils.Resource) {
	if resource == nil || resource.InstanceState == nil || resource.InstanceState.Attributes == nil {
		return
	}
	if resource.Item == nil {
		resource.Item = map[string]interface{}{}
	}

	for key, countValue := range resource.InstanceState.Attributes {
		flatmapPath, ok := strings.CutSuffix(key, ".#")
		if !ok {
			continue
		}
		if _, ok := observabilityPipelineEmptyVariantBlockPaths[observabilityPipelineNormalizeFlatmapPath(flatmapPath)]; !ok {
			continue
		}
		count, err := strconv.Atoi(countValue)
		if err != nil || count <= 0 {
			continue
		}
		observabilityPipelineEnsureEmptyBlockList(resource.Item, flatmapPath, count)
	}
}

func observabilityPipelineNormalizeFlatmapPath(path string) string {
	parts := strings.Split(path, ".")
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		if _, err := strconv.Atoi(part); err == nil {
			continue
		}
		normalized = append(normalized, part)
	}
	return strings.Join(normalized, ".")
}

func observabilityPipelineEnsureEmptyBlockList(item map[string]interface{}, flatmapPath string, count int) {
	parts := strings.Split(flatmapPath, ".")
	current := item
	for i := 0; i < len(parts); {
		name := parts[i]
		if i == len(parts)-1 {
			if existing, ok := current[name]; ok {
				list, ok := existing.([]interface{})
				if !ok || len(list) > 0 {
					return
				}
			}
			current[name] = observabilityPipelineEmptyBlocks(count)
			return
		}

		if i+1 >= len(parts) {
			return
		}
		index, err := strconv.Atoi(parts[i+1])
		if err != nil || index < 0 {
			return
		}
		list, ok := observabilityPipelineEnsureList(current, name, index+1)
		if !ok {
			return
		}
		child, ok := list[index].(map[string]interface{})
		if !ok {
			if list[index] != nil {
				return
			}
			child = map[string]interface{}{}
			list[index] = child
		}
		current = child
		i += 2
	}
}

func observabilityPipelineEnsureList(parent map[string]interface{}, name string, length int) ([]interface{}, bool) {
	value, ok := parent[name]
	if !ok || value == nil {
		list := make([]interface{}, length)
		parent[name] = list
		return list, true
	}
	list, ok := value.([]interface{})
	if !ok {
		return nil, false
	}
	for len(list) < length {
		list = append(list, nil)
	}
	parent[name] = list
	return list, true
}

func observabilityPipelineEmptyBlocks(count int) []interface{} {
	blocks := make([]interface{}, count)
	for i := range blocks {
		blocks[i] = map[string]interface{}{}
	}
	return blocks
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
