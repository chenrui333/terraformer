// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"

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
