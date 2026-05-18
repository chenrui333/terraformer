// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	datadogObservabilityPipelineServiceName = "observability_pipeline"
	datadogObservabilityPipelinePageSize    = int64(100)
)

var (
	// ObservabilityPipelineAllowEmptyValues ...
	ObservabilityPipelineAllowEmptyValues = observabilityPipelineIndexedFlatmapPaths(
		"config.destination.elasticsearch.data_stream.auto_routing",
		"config.destination.elasticsearch.data_stream.sync_fields",
		"config.destination.elasticsearch.request_retry_partial",
		"config.destination.splunk_hec.auto_extract_timestamp",
		"config.destination.datadog_logs.routes.include",
		"config.processor_group.include",
		"config.processor_group.enabled",
		"config.processor_group.processor.include",
		"config.processor_group.processor.custom_processor.remap.include",
		"config.processor_group.processor.custom_processor.remap.drop_on_error",
		"config.processor_group.processor.custom_processor.remap.enabled",
		"config.processor_group.processor.enabled",
		"config.processor_group.processor.enrichment_table.file.encoding.includes_headers",
		"config.processor_group.processor.generate_datadog_metrics.metric.include",
		"config.processor_group.processor.metric_tags.rule.include",
		"config.processor_group.processor.ocsf_mapper.mapping.include",
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
		"config.processor_group.processor.split_array.array.include",
		"config.source.splunk_hec.store_hec_token",
		"config.use_legacy_search_syntax",
	)

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
		"config.processor_group.processor.sensitive_data_scanner.rule.scope.exclude": {},
		"config.processor_group.processor.sensitive_data_scanner.rule.scope.include": {},
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

	observabilityPipelineEmptyListPaths = map[string]struct{}{
		"config.processor_group.processor.quota.partition_fields":           {},
		"config.processor_group.processor.reduce.group_by":                  {},
		"config.processor_group.processor.sensitive_data_scanner.rule.tags": {},
	}
)

func observabilityPipelineIndexedFlatmapPaths(paths ...string) []string {
	patterns := make([]string, 0, len(paths))
	for _, path := range paths {
		patterns = append(patterns, observabilityPipelineIndexedFlatmapPath(path))
	}
	return patterns
}

func observabilityPipelineIndexedFlatmapPath(path string) string {
	parts := strings.Split(path, ".")
	pattern := strings.Builder{}
	for i, part := range parts {
		if i > 0 {
			pattern.WriteString("\\.")
		}
		pattern.WriteString(regexp.QuoteMeta(part))
		if i < len(parts)-1 {
			pattern.WriteString("(\\.[0-9]+)?")
		}
	}
	return pattern.String()
}

// ObservabilityPipelineGenerator ...
type ObservabilityPipelineGenerator struct {
	DatadogService
}

func (g *ObservabilityPipelineGenerator) createResource(pipelineID string) (terraformutils.Resource, error) {
	return newDatadogIDResource(datadogObservabilityPipelineServiceName, pipelineID, ObservabilityPipelineAllowEmptyValues)
}

func (g *ObservabilityPipelineGenerator) createResources(pipelineIDs []string) ([]terraformutils.Resource, error) {
	return datadogIDResources(datadogObservabilityPipelineServiceName, pipelineIDs, ObservabilityPipelineAllowEmptyValues)
}

func (g *ObservabilityPipelineGenerator) PostConvertHook() error {
	for i := range g.Resources {
		preserveObservabilityPipelineVariantBlocks(&g.Resources[i])
	}
	return nil
}

func preserveObservabilityPipelineVariantBlocks(resource *terraformutils.Resource) {
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
		normalizedPath := observabilityPipelineNormalizeFlatmapPath(flatmapPath)
		count, err := strconv.Atoi(countValue)
		if err != nil {
			continue
		}
		if count == 0 && observabilityPipelineIsEmptyListPath(normalizedPath) {
			observabilityPipelineEnsureEmptyList(resource.Item, flatmapPath)
			continue
		}
		if count <= 0 || !observabilityPipelineIsEmptyVariantBlockPath(normalizedPath) {
			continue
		}
		observabilityPipelineEnsureEmptyBlockList(resource.Item, flatmapPath, count)
	}
}

func observabilityPipelineIsEmptyListPath(path string) bool {
	_, ok := observabilityPipelineEmptyListPaths[path]
	return ok
}

func observabilityPipelineIsEmptyVariantBlockPath(path string) bool {
	if _, ok := observabilityPipelineEmptyVariantBlockPaths[path]; ok {
		return true
	}
	return observabilityPipelineIsEmptySensitiveDataScannerActionBlockPath(path)
}

func observabilityPipelineIsEmptySensitiveDataScannerActionBlockPath(path string) bool {
	const prefix = "config.processor_group.processor.sensitive_data_scanner.rule.on_match."
	action, ok := strings.CutPrefix(path, prefix)
	if !ok {
		return false
	}
	return action == "partial_redact" || action == "redact"
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

func observabilityPipelineEnsureEmptyList(item map[string]interface{}, flatmapPath string) {
	parts := strings.Split(flatmapPath, ".")
	current := item
	for i := 0; i < len(parts); {
		name := parts[i]
		if i == len(parts)-1 {
			if _, ok := current[name]; ok {
				return
			}
			current[name] = []interface{}{}
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

// InitResources Generate TerraformResources from Datadog API,
// from each observability_pipeline create 1 TerraformResource.
func (g *ObservabilityPipelineGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewObservabilityPipelinesApi(datadogClient)

	resources, hasIDFilter, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if hasIDFilter {
		g.Resources = resources
		return nil
	}

	pipelineIDs, err := g.listObservabilityPipelineIDs(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(pipelineIDs)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}

func (g *ObservabilityPipelineGenerator) filteredResources(auth context.Context, api *datadogV2.ObservabilityPipelinesApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	hasIDFilter := false
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || filter.ServiceName != datadogObservabilityPipelineServiceName {
			continue
		}
		hasIDFilter = true
		for _, value := range filter.AcceptableValues {
			pipelineID, err := g.getObservabilityPipelineID(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(pipelineID)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}
	return resources, hasIDFilter, nil
}

func (g *ObservabilityPipelineGenerator) getObservabilityPipelineID(auth context.Context, api *datadogV2.ObservabilityPipelinesApi, pipelineID string) (string, error) {
	resp, httpResp, err := api.GetPipeline(auth, pipelineID)
	closeDatadogResponseBody(httpResp)
	if err != nil {
		return "", err
	}
	data := resp.GetData()
	responseID := data.GetId()
	if responseID == "" {
		return pipelineID, nil
	}
	return responseID, nil
}

func (g *ObservabilityPipelineGenerator) listObservabilityPipelineIDs(auth context.Context, api *datadogV2.ObservabilityPipelinesApi) ([]string, error) {
	ids := []string{}
	pageNumber := int64(0)
	pipelinesSeen := int64(0)

	for {
		opts := datadogV2.NewListPipelinesOptionalParameters().
			WithPageSize(datadogObservabilityPipelinePageSize).
			WithPageNumber(pageNumber)

		resp, httpResp, err := api.ListPipelines(auth, *opts)
		closeDatadogResponseBody(httpResp)
		if err != nil {
			return nil, err
		}

		pipelines := resp.GetData()
		for _, pipeline := range pipelines {
			pipelineID := pipeline.GetId()
			if pipelineID == "" {
				continue
			}
			ids = append(ids, pipelineID)
		}
		if len(pipelines) == 0 {
			break
		}
		pipelinesSeen += int64(len(pipelines))

		meta := resp.GetMeta()
		if totalCount, ok := meta.GetTotalCountOk(); ok {
			if pipelinesSeen >= *totalCount {
				break
			}
		} else if int64(len(pipelines)) < datadogObservabilityPipelinePageSize {
			break
		}
		pageNumber++
	}

	return ids, nil
}
