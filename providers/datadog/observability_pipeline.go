// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
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
	ObservabilityPipelineAllowEmptyValues = []string{}
)

var (
	observabilityPipelineDestinationSkippedVariantKeys = map[string]struct{}{
		"inputs": {},
	}
)

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
	if resource == nil || resource.Item == nil || resource.InstanceState == nil || resource.InstanceState.Attributes == nil {
		return
	}
	configs, ok := resource.Item["config"].([]interface{})
	if !ok {
		return
	}

	for configIndex, config := range configs {
		configBlock, ok := config.(map[string]interface{})
		if !ok {
			continue
		}
		configPrefix := fmt.Sprintf("config.%d", configIndex)
		preserveObservabilityPipelineVariantList(resource.InstanceState.Attributes, configBlock, configPrefix+".source", "source", nil)
		preserveObservabilityPipelineVariantList(resource.InstanceState.Attributes, configBlock, configPrefix+".destination", "destination", observabilityPipelineDestinationSkippedVariantKeys)
		preserveObservabilityPipelineProcessorVariants(resource.InstanceState.Attributes, configBlock, configPrefix)
	}
}

func preserveObservabilityPipelineProcessorVariants(attributes map[string]string, configBlock map[string]interface{}, configPrefix string) {
	processorGroups, ok := configBlock["processor_group"].([]interface{})
	if !ok {
		return
	}

	for groupIndex, processorGroup := range processorGroups {
		groupBlock, ok := processorGroup.(map[string]interface{})
		if !ok {
			continue
		}
		processorPrefix := fmt.Sprintf("%s.processor_group.%d.processor", configPrefix, groupIndex)
		preserveObservabilityPipelineVariantList(attributes, groupBlock, processorPrefix, "processor", nil)
	}
}

func preserveObservabilityPipelineVariantList(attributes map[string]string, parentBlock map[string]interface{}, statePrefix, itemKey string, skippedVariantKeys map[string]struct{}) {
	items, ok := parentBlock[itemKey].([]interface{})
	if !ok {
		return
	}

	for itemIndex, item := range items {
		itemBlock, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		variantCounts := observabilityPipelineVariantCounts(attributes, fmt.Sprintf("%s.%d.", statePrefix, itemIndex), skippedVariantKeys)
		for variantKey, count := range variantCounts {
			if count == "0" {
				continue
			}
			if value, ok := itemBlock[variantKey]; ok && observabilityPipelineValueHasValue(value) {
				continue
			}
			itemBlock[variantKey] = []interface{}{map[string]interface{}{}}
		}
	}
}

func observabilityPipelineVariantCounts(attributes map[string]string, prefix string, skippedVariantKeys map[string]struct{}) map[string]string {
	counts := map[string]string{}
	for key, count := range attributes {
		if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, ".#") {
			continue
		}
		variantKey := strings.TrimSuffix(strings.TrimPrefix(key, prefix), ".#")
		if strings.Contains(variantKey, ".") {
			continue
		}
		if _, skipped := skippedVariantKeys[variantKey]; skipped {
			continue
		}
		counts[variantKey] = count
	}
	return counts
}

func observabilityPipelineValueHasValue(value interface{}) bool {
	switch typedValue := value.(type) {
	case nil:
		return false
	case []interface{}:
		return len(typedValue) > 0
	case map[string]interface{}:
		return len(typedValue) > 0
	default:
		return true
	}
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
