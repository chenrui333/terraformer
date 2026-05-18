// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestObservabilityPipelineAllowEmptyValuesMatchesIndexedFlatmapPaths(t *testing.T) {
	allowEmptyValues := allowEmptyValueRegexps(ObservabilityPipelineAllowEmptyValues)
	requiredPaths := []string{
		"config.0.destination.0.elasticsearch.0.data_stream.0.auto_routing",
		"config.0.destination.0.elasticsearch.0.data_stream.0.sync_fields",
		"config.0.destination.0.elasticsearch.0.request_retry_partial",
		"config.0.destination.0.splunk_hec.0.auto_extract_timestamp",
		"config.0.destination.0.datadog_logs.0.routes.0.include",
		"config.0.processor_group.0.include",
		"config.0.processor_group.0.enabled",
		"config.0.processor_group.0.processor.0.include",
		"config.0.processor_group.0.processor.0.custom_processor.0.remap.0.include",
		"config.0.processor_group.0.processor.0.custom_processor.0.remap.0.drop_on_error",
		"config.0.processor_group.0.processor.0.custom_processor.0.remap.0.enabled",
		"config.0.processor_group.0.processor.0.enabled",
		"config.0.processor_group.0.processor.0.enrichment_table.0.file.0.encoding.0.includes_headers",
		"config.0.processor_group.0.processor.0.generate_datadog_metrics.0.metric.0.include",
		"config.0.processor_group.0.processor.0.metric_tags.0.rule.0.include",
		"config.0.processor_group.0.processor.0.ocsf_mapper.0.mapping.0.include",
		"config.0.processor_group.0.processor.0.ocsf_mapper.0.keep_unmatched",
		"config.0.processor_group.0.processor.0.parse_grok.0.disable_library_rules",
		"config.0.processor_group.0.processor.0.parse_xml.0.always_use_text_key",
		"config.0.processor_group.0.processor.0.parse_xml.0.include_attr",
		"config.0.processor_group.0.processor.0.parse_xml.0.parse_bool",
		"config.0.processor_group.0.processor.0.parse_xml.0.parse_null",
		"config.0.processor_group.0.processor.0.parse_xml.0.parse_number",
		"config.0.processor_group.0.processor.0.quota.0.drop_events",
		"config.0.processor_group.0.processor.0.quota.0.ignore_when_missing_partitions",
		"config.0.processor_group.0.processor.0.rename_fields.0.field.0.preserve_source",
		"config.0.processor_group.0.processor.0.sensitive_data_scanner.0.rule.0.pattern.0.library.0.use_recommended_keywords",
		"config.0.processor_group.0.processor.0.sensitive_data_scanner.0.rule.0.scope.0.all",
		"config.0.processor_group.0.processor.0.split_array.0.array.0.include",
		"config.0.source.0.splunk_hec.0.store_hec_token",
		"config.0.use_legacy_search_syntax",
	}
	for _, path := range requiredPaths {
		matched := false
		for _, pattern := range allowEmptyValues {
			if pattern.MatchString(path) {
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("ObservabilityPipelineAllowEmptyValues must match indexed path %q", path)
		}
	}
}

func TestObservabilityPipelineAllowEmptyValuesPreservesRequiredIncludeQueries(t *testing.T) {
	parser := terraformutils.NewFlatmapParser(map[string]string{
		"config.#":                                                                           "1",
		"config.0.destination.#":                                                             "1",
		"config.0.destination.0.id":                                                          "datadog-logs",
		"config.0.destination.0.datadog_logs.#":                                              "1",
		"config.0.destination.0.datadog_logs.0.routes.#":                                     "1",
		"config.0.destination.0.datadog_logs.0.routes.0.include":                             "",
		"config.0.processor_group.#":                                                         "1",
		"config.0.processor_group.0.id":                                                      "processor-group",
		"config.0.processor_group.0.include":                                                 "",
		"config.0.processor_group.0.processor.#":                                             "1",
		"config.0.processor_group.0.processor.0.id":                                          "processor",
		"config.0.processor_group.0.processor.0.include":                                     "",
		"config.0.processor_group.0.processor.0.custom_processor.#":                          "1",
		"config.0.processor_group.0.processor.0.custom_processor.0.remap.#":                  "1",
		"config.0.processor_group.0.processor.0.custom_processor.0.remap.0.include":          "",
		"config.0.processor_group.0.processor.0.generate_datadog_metrics.#":                  "1",
		"config.0.processor_group.0.processor.0.generate_datadog_metrics.0.metric.#":         "1",
		"config.0.processor_group.0.processor.0.generate_datadog_metrics.0.metric.0.include": "",
		"config.0.processor_group.0.processor.0.metric_tags.#":                               "1",
		"config.0.processor_group.0.processor.0.metric_tags.0.rule.#":                        "1",
		"config.0.processor_group.0.processor.0.metric_tags.0.rule.0.include":                "",
		"config.0.processor_group.0.processor.0.ocsf_mapper.#":                               "1",
		"config.0.processor_group.0.processor.0.ocsf_mapper.0.mapping.#":                     "1",
		"config.0.processor_group.0.processor.0.ocsf_mapper.0.mapping.0.include":             "",
		"config.0.processor_group.0.processor.0.split_array.#":                               "1",
		"config.0.processor_group.0.processor.0.split_array.0.array.#":                       "1",
		"config.0.processor_group.0.processor.0.split_array.0.array.0.include":               "",
	}, nil, allowEmptyValueRegexps(ObservabilityPipelineAllowEmptyValues))
	pipelineType := cty.Object(map[string]cty.Type{
		"config": cty.List(cty.Object(map[string]cty.Type{
			"destination": cty.List(cty.Object(map[string]cty.Type{
				"id": cty.String,
				"datadog_logs": cty.List(cty.Object(map[string]cty.Type{
					"routes": cty.List(cty.Object(map[string]cty.Type{
						"include": cty.String,
					})),
				})),
			})),
			"processor_group": cty.List(cty.Object(map[string]cty.Type{
				"id":      cty.String,
				"include": cty.String,
				"processor": cty.List(cty.Object(map[string]cty.Type{
					"id":      cty.String,
					"include": cty.String,
					"custom_processor": cty.List(cty.Object(map[string]cty.Type{
						"remap": cty.List(cty.Object(map[string]cty.Type{
							"include": cty.String,
						})),
					})),
					"generate_datadog_metrics": cty.List(cty.Object(map[string]cty.Type{
						"metric": cty.List(cty.Object(map[string]cty.Type{
							"include": cty.String,
						})),
					})),
					"metric_tags": cty.List(cty.Object(map[string]cty.Type{
						"rule": cty.List(cty.Object(map[string]cty.Type{
							"include": cty.String,
						})),
					})),
					"ocsf_mapper": cty.List(cty.Object(map[string]cty.Type{
						"mapping": cty.List(cty.Object(map[string]cty.Type{
							"include": cty.String,
						})),
					})),
					"split_array": cty.List(cty.Object(map[string]cty.Type{
						"array": cty.List(cty.Object(map[string]cty.Type{
							"include": cty.String,
						})),
					})),
				})),
			})),
		})),
	})

	result, err := parser.Parse(pipelineType)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	config := requireObservabilityPipelineMapInList(t, result, "config", 0)
	destination := requireObservabilityPipelineMapInList(t, config, "destination", 0)
	datadogLogs := requireObservabilityPipelineMapInList(t, destination, "datadog_logs", 0)
	route := requireObservabilityPipelineMapInList(t, datadogLogs, "routes", 0)
	if route["include"] != "" {
		t.Fatalf("destination route include = %v, want empty string", route["include"])
	}
	processorGroup := requireObservabilityPipelineMapInList(t, config, "processor_group", 0)
	if processorGroup["include"] != "" {
		t.Fatalf("processor_group include = %v, want empty string", processorGroup["include"])
	}
	processor := requireObservabilityPipelineMapInList(t, processorGroup, "processor", 0)
	if processor["include"] != "" {
		t.Fatalf("processor include = %v, want empty string", processor["include"])
	}
	customProcessor := requireObservabilityPipelineMapInList(t, processor, "custom_processor", 0)
	remap := requireObservabilityPipelineMapInList(t, customProcessor, "remap", 0)
	if remap["include"] != "" {
		t.Fatalf("custom processor remap include = %v, want empty string", remap["include"])
	}
	generatedMetrics := requireObservabilityPipelineMapInList(t, processor, "generate_datadog_metrics", 0)
	metric := requireObservabilityPipelineMapInList(t, generatedMetrics, "metric", 0)
	if metric["include"] != "" {
		t.Fatalf("generated metric include = %v, want empty string", metric["include"])
	}
	metricTags := requireObservabilityPipelineMapInList(t, processor, "metric_tags", 0)
	rule := requireObservabilityPipelineMapInList(t, metricTags, "rule", 0)
	if rule["include"] != "" {
		t.Fatalf("metric tags rule include = %v, want empty string", rule["include"])
	}
	ocsfMapper := requireObservabilityPipelineMapInList(t, processor, "ocsf_mapper", 0)
	mapping := requireObservabilityPipelineMapInList(t, ocsfMapper, "mapping", 0)
	if mapping["include"] != "" {
		t.Fatalf("ocsf mapping include = %v, want empty string", mapping["include"])
	}
	splitArray := requireObservabilityPipelineMapInList(t, processor, "split_array", 0)
	array := requireObservabilityPipelineMapInList(t, splitArray, "array", 0)
	if array["include"] != "" {
		t.Fatalf("split array include = %v, want empty string", array["include"])
	}
}

func TestObservabilityPipelinePostConvertHookPreservesNestedEmptyVariantBlocks(t *testing.T) {
	resource := observabilityPipelineResourceWithItemForTest(map[string]interface{}{
		"config": []interface{}{
			map[string]interface{}{
				"destination": []interface{}{
					map[string]interface{}{"id": "logs", "inputs": []interface{}{"source"}},
					map[string]interface{}{"id": "metrics", "inputs": []interface{}{"source"}},
				},
				"processor_group": []interface{}{
					map[string]interface{}{
						"processor": []interface{}{
							map[string]interface{}{"id": "filter"},
							map[string]interface{}{"id": "hostname"},
							map[string]interface{}{
								"id": "scanner",
								"sensitive_data_scanner": []interface{}{
									map[string]interface{}{
										"rule": []interface{}{
											map[string]interface{}{
												"on_match": []interface{}{
													map[string]interface{}{},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	resource.InstanceState.Attributes = map[string]string{
		"config.0.destination.0.datadog_logs.#":                                                              "1",
		"config.0.destination.1.datadog_metrics.#":                                                           "1",
		"config.0.processor_group.0.processor.0.filter.#":                                                    "1",
		"config.0.processor_group.0.processor.1.add_hostname.#":                                              "1",
		"config.0.processor_group.0.processor.2.sensitive_data_scanner.0.rule.0.on_match.0.hash.#":           "1",
		"config.0.processor_group.0.processor.2.sensitive_data_scanner.0.rule.0.on_match.0.partial_redact.#": "1",
		"config.0.processor_group.0.processor.2.sensitive_data_scanner.0.rule.0.on_match.0.redact.#":         "1",
	}
	generator := &ObservabilityPipelineGenerator{}
	generator.Resources = []terraformutils.Resource{resource}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}

	config := requireObservabilityPipelineMapInList(t, generator.Resources[0].Item, "config", 0)
	requireObservabilityPipelineEmptyBlockList(t, requireObservabilityPipelineMapInList(t, config, "destination", 0), "datadog_logs")
	requireObservabilityPipelineEmptyBlockList(t, requireObservabilityPipelineMapInList(t, config, "destination", 1), "datadog_metrics")

	processorGroup := requireObservabilityPipelineMapInList(t, config, "processor_group", 0)
	requireObservabilityPipelineEmptyBlockList(t, requireObservabilityPipelineMapInList(t, processorGroup, "processor", 0), "filter")
	requireObservabilityPipelineEmptyBlockList(t, requireObservabilityPipelineMapInList(t, processorGroup, "processor", 1), "add_hostname")

	scannerProcessor := requireObservabilityPipelineMapInList(t, processorGroup, "processor", 2)
	scanner := requireObservabilityPipelineMapInList(t, scannerProcessor, "sensitive_data_scanner", 0)
	rule := requireObservabilityPipelineMapInList(t, scanner, "rule", 0)
	onMatch := requireObservabilityPipelineMapInList(t, rule, "on_match", 0)
	requireObservabilityPipelineEmptyBlockList(t, onMatch, "hash")
	requireObservabilityPipelineEmptyBlockList(t, onMatch, "partial_redact")
	requireObservabilityPipelineEmptyBlockList(t, onMatch, "redact")
}

func TestObservabilityPipelinePostConvertHookPreservesEmptyListsAndScopeVariants(t *testing.T) {
	resource := observabilityPipelineResourceWithItemForTest(map[string]interface{}{
		"config": []interface{}{
			map[string]interface{}{
				"processor_group": []interface{}{
					map[string]interface{}{
						"processor": []interface{}{
							map[string]interface{}{
								"id": "quota",
								"quota": []interface{}{
									map[string]interface{}{},
								},
							},
							map[string]interface{}{
								"id": "reduce",
								"reduce": []interface{}{
									map[string]interface{}{},
								},
							},
							map[string]interface{}{
								"id": "scanner",
								"sensitive_data_scanner": []interface{}{
									map[string]interface{}{
										"rule": []interface{}{
											map[string]interface{}{
												"scope": []interface{}{
													map[string]interface{}{},
												},
											},
											map[string]interface{}{
												"scope": []interface{}{
													map[string]interface{}{},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	resource.InstanceState.Attributes = map[string]string{
		"config.0.processor_group.0.processor.0.quota.#":                                                    "1",
		"config.0.processor_group.0.processor.0.quota.0.partition_fields.#":                                 "0",
		"config.0.processor_group.0.processor.1.reduce.#":                                                   "1",
		"config.0.processor_group.0.processor.1.reduce.0.group_by.#":                                        "0",
		"config.0.processor_group.0.processor.2.sensitive_data_scanner.0.rule.0.scope.0.include.#":          "1",
		"config.0.processor_group.0.processor.2.sensitive_data_scanner.0.rule.0.scope.0.include.0.fields.#": "0",
		"config.0.processor_group.0.processor.2.sensitive_data_scanner.0.rule.0.tags.#":                     "0",
		"config.0.processor_group.0.processor.2.sensitive_data_scanner.0.rule.1.scope.0.exclude.#":          "1",
		"config.0.processor_group.0.processor.2.sensitive_data_scanner.0.rule.1.scope.0.exclude.0.fields.#": "0",
		"config.0.processor_group.0.processor.2.sensitive_data_scanner.0.rule.1.tags.#":                     "0",
	}
	generator := &ObservabilityPipelineGenerator{}
	generator.Resources = []terraformutils.Resource{resource}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}

	config := requireObservabilityPipelineMapInList(t, generator.Resources[0].Item, "config", 0)
	processorGroup := requireObservabilityPipelineMapInList(t, config, "processor_group", 0)
	quotaProcessor := requireObservabilityPipelineMapInList(t, processorGroup, "processor", 0)
	quota := requireObservabilityPipelineMapInList(t, quotaProcessor, "quota", 0)
	requireObservabilityPipelineEmptyList(t, quota, "partition_fields")

	reduceProcessor := requireObservabilityPipelineMapInList(t, processorGroup, "processor", 1)
	reduce := requireObservabilityPipelineMapInList(t, reduceProcessor, "reduce", 0)
	requireObservabilityPipelineEmptyList(t, reduce, "group_by")

	scannerProcessor := requireObservabilityPipelineMapInList(t, processorGroup, "processor", 2)
	scanner := requireObservabilityPipelineMapInList(t, scannerProcessor, "sensitive_data_scanner", 0)
	includeRule := requireObservabilityPipelineMapInList(t, scanner, "rule", 0)
	includeScope := requireObservabilityPipelineMapInList(t, includeRule, "scope", 0)
	requireObservabilityPipelineEmptyBlockList(t, includeScope, "include")
	requireObservabilityPipelineEmptyList(t, includeRule, "tags")
	excludeRule := requireObservabilityPipelineMapInList(t, scanner, "rule", 1)
	excludeScope := requireObservabilityPipelineMapInList(t, excludeRule, "scope", 0)
	requireObservabilityPipelineEmptyBlockList(t, excludeScope, "exclude")
	requireObservabilityPipelineEmptyList(t, excludeRule, "tags")
}

func TestObservabilityPipelinePostConvertHookDoesNotInventMissingVariantBlocks(t *testing.T) {
	resource := observabilityPipelineResourceWithItemForTest(map[string]interface{}{
		"config": []interface{}{
			map[string]interface{}{
				"destination": []interface{}{
					map[string]interface{}{"id": "logs", "inputs": []interface{}{"source"}},
				},
			},
		},
	})
	resource.InstanceState.Attributes = map[string]string{"id": "pipeline-123"}
	generator := &ObservabilityPipelineGenerator{}
	generator.Resources = []terraformutils.Resource{resource}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}

	config := requireObservabilityPipelineMapInList(t, generator.Resources[0].Item, "config", 0)
	destination := requireObservabilityPipelineMapInList(t, config, "destination", 0)
	if _, ok := destination["datadog_logs"]; ok {
		t.Fatal("PostConvertHook added datadog_logs without a matching state count marker")
	}
}

func TestObservabilityPipelinePostConvertHookDoesNotOverwriteExistingVariantBlocks(t *testing.T) {
	existing := []interface{}{
		map[string]interface{}{"routes": []interface{}{"route-a"}},
	}
	resource := observabilityPipelineResourceWithItemForTest(map[string]interface{}{
		"config": []interface{}{
			map[string]interface{}{
				"destination": []interface{}{
					map[string]interface{}{
						"id":           "logs",
						"inputs":       []interface{}{"source"},
						"datadog_logs": existing,
					},
				},
			},
		},
	})
	resource.InstanceState.Attributes = map[string]string{
		"config.0.destination.0.datadog_logs.#": "1",
	}
	generator := &ObservabilityPipelineGenerator{}
	generator.Resources = []terraformutils.Resource{resource}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}

	config := requireObservabilityPipelineMapInList(t, generator.Resources[0].Item, "config", 0)
	destination := requireObservabilityPipelineMapInList(t, config, "destination", 0)
	got := requireObservabilityPipelineList(t, destination, "datadog_logs")
	if len(got) != 1 {
		t.Fatalf("datadog_logs length = %d, want 1", len(got))
	}
	block, ok := got[0].(map[string]interface{})
	if !ok {
		t.Fatalf("datadog_logs[0] = %T, want map[string]interface{}", got[0])
	}
	if _, ok := block["routes"]; !ok {
		t.Fatalf("datadog_logs = %#v, want existing routes preserved", got)
	}
}

func observabilityPipelineResourceWithItemForTest(item map[string]interface{}) terraformutils.Resource {
	resource := terraformutils.NewSimpleResource(
		"pipeline-123",
		"observability_pipeline_pipeline-123",
		"datadog_observability_pipeline",
		"datadog",
		ObservabilityPipelineAllowEmptyValues,
	)
	resource.Item = item
	return resource
}

func requireObservabilityPipelineList(t *testing.T, parent map[string]interface{}, key string) []interface{} {
	t.Helper()
	list, ok := parent[key].([]interface{})
	if !ok {
		t.Fatalf("%s = %T, want []interface{}", key, parent[key])
	}
	return list
}

func requireObservabilityPipelineMapInList(t *testing.T, parent map[string]interface{}, key string, index int) map[string]interface{} {
	t.Helper()
	list := requireObservabilityPipelineList(t, parent, key)
	if len(list) <= index {
		t.Fatalf("%s length = %d, want index %d", key, len(list), index)
	}
	item, ok := list[index].(map[string]interface{})
	if !ok {
		t.Fatalf("%s[%d] = %T, want map[string]interface{}", key, index, list[index])
	}
	return item
}

func requireObservabilityPipelineEmptyList(t *testing.T, parent map[string]interface{}, key string) {
	t.Helper()
	list := requireObservabilityPipelineList(t, parent, key)
	if len(list) != 0 {
		t.Fatalf("%s length = %d, want 0", key, len(list))
	}
}

func requireObservabilityPipelineEmptyBlockList(t *testing.T, parent map[string]interface{}, key string) {
	t.Helper()
	list := requireObservabilityPipelineList(t, parent, key)
	if len(list) != 1 {
		t.Fatalf("%s length = %d, want 1", key, len(list))
	}
	block, ok := list[0].(map[string]interface{})
	if !ok {
		t.Fatalf("%s[0] = %T, want map[string]interface{}", key, list[0])
	}
	if len(block) != 0 {
		t.Fatalf("%s[0] = %#v, want empty block", key, block)
	}
}
