// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/datapipeline"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	dataPipelinePipelineResourceType           = "aws_datapipeline_pipeline"
	dataPipelinePipelineDefinitionResourceType = "aws_datapipeline_pipeline_definition"
)

var datapipelineAllowEmptyValues = []string{"tags."}

type DataPipelineGenerator struct {
	AWSService
}

func (g *DataPipelineGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := datapipeline.NewFromConfig(config)
	pipelineIDFilter := dataPipelinePipelineIDFilter(g.Filter)
	p := datapipeline.NewListPipelinesPaginator(svc, &datapipeline.ListPipelinesInput{})
	var resources []terraformutils.Resource
	for p.HasMorePages() {
		page, e := p.NextPage(context.TODO())
		if e != nil {
			return e
		}
		for _, pipeline := range page.PipelineIdList {
			pipelineID := StringValue(pipeline.Id)
			pipelineName := StringValue(pipeline.Name)
			if !awsIDFilterAllows(pipelineIDFilter, pipelineID) {
				continue
			}
			if resource, ok := newDataPipelinePipelineResource(pipelineID, pipelineName); ok && dataPipelineShouldEmitPipeline(g.Filter, pipelineID) {
				resources = append(resources, resource)
			}
			resource, ok, err := getDataPipelinePipelineDefinitionResource(svc, pipelineID, pipelineName)
			if err != nil {
				log.Printf("[WARN] Skipping Data Pipeline definition for %s: %v", pipelineID, err)
				continue
			}
			if ok {
				resources = append(resources, resource)
			}
		}
	}
	g.Resources = resources
	return nil
}

func (g *DataPipelineGenerator) PostRefreshCleanup() {
	if len(g.Filter) == 0 {
		return
	}
	definitionsByPipelineID := map[string][]terraformutils.Resource{}
	matchedPipelineIDs := map[string]bool{}
	for _, resource := range g.Resources {
		switch resource.InstanceInfo.Type {
		case dataPipelinePipelineResourceType:
			if dataPipelineResourceMatchesPostRefreshFilters(resource, g.Filter) {
				matchedPipelineIDs[resource.InstanceState.ID] = true
			}
		case dataPipelinePipelineDefinitionResourceType:
			if pipelineID := dataPipelineDefinitionPipelineID(resource); pipelineID != "" {
				definitionsByPipelineID[pipelineID] = append(definitionsByPipelineID[pipelineID], resource)
			}
		}
	}

	terraformutils.FilterCleanup(&g.Service, false)
	if dataPipelineHasTypedFilter(g.Filter) && !awsHasApplicableFilter(g.Filter, dataPipelinePipelineResourceType) {
		g.Resources = dataPipelineResourcesWithoutType(g.Resources, dataPipelinePipelineResourceType)
	}
	for pipelineID := range matchedPipelineIDs {
		for _, resource := range definitionsByPipelineID[pipelineID] {
			if !terraformutils.ContainsResource(g.Resources, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
}

func getDataPipelinePipelineDefinitionResource(svc *datapipeline.Client, pipelineID, pipelineName string) (terraformutils.Resource, bool, error) {
	if pipelineID == "" {
		return terraformutils.Resource{}, false, nil
	}
	output, err := svc.GetPipelineDefinition(context.TODO(), &datapipeline.GetPipelineDefinitionInput{
		PipelineId: &pipelineID,
	})
	if err != nil {
		return terraformutils.Resource{}, false, err
	}
	if !dataPipelineDefinitionImportable(output) {
		return terraformutils.Resource{}, false, nil
	}
	return newDataPipelinePipelineDefinitionResource(pipelineID, pipelineName), true, nil
}

func newDataPipelinePipelineResource(pipelineID, pipelineName string) (terraformutils.Resource, bool) {
	if pipelineID == "" {
		return terraformutils.Resource{}, false
	}
	if pipelineName == "" {
		pipelineName = pipelineID
	}
	return terraformutils.NewSimpleResource(
		dataPipelinePipelineImportID(pipelineID),
		pipelineName,
		dataPipelinePipelineResourceType,
		"aws",
		datapipelineAllowEmptyValues), true
}

func newDataPipelinePipelineDefinitionResource(pipelineID, pipelineName string) terraformutils.Resource {
	if pipelineName == "" {
		pipelineName = pipelineID
	}
	return terraformutils.NewResource(
		dataPipelinePipelineDefinitionImportID(pipelineID),
		dataPipelineResourceName("pipeline-definition", pipelineName, pipelineID),
		dataPipelinePipelineDefinitionResourceType,
		"aws",
		map[string]string{
			"name":        pipelineName,
			"pipeline_id": pipelineID,
		},
		datapipelineAllowEmptyValues,
		map[string]interface{}{})
}

func dataPipelineDefinitionImportable(definition *datapipeline.GetPipelineDefinitionOutput) bool {
	return definition != nil && len(definition.PipelineObjects) > 0
}

func dataPipelineResourceMatchesPostRefreshFilters(resource terraformutils.Resource, filters []terraformutils.ResourceFilter) bool {
	matchedApplicableFilter := false
	serviceName := strings.TrimPrefix(resource.InstanceInfo.Type, resource.Provider+"_")
	for _, filter := range filters {
		if filter.FieldPath == "id" || !filter.IsApplicable(serviceName) {
			continue
		}
		matchedApplicableFilter = true
		if !filter.Filter(resource) {
			return false
		}
	}
	return matchedApplicableFilter
}

func dataPipelineResourcesWithoutType(resources []terraformutils.Resource, resourceType string) []terraformutils.Resource {
	filtered := []terraformutils.Resource{}
	for _, resource := range resources {
		if resource.InstanceInfo.Type != resourceType {
			filtered = append(filtered, resource)
		}
	}
	return filtered
}

func dataPipelineDefinitionPipelineID(resource terraformutils.Resource) string {
	if resource.InstanceState == nil || resource.InstanceState.Attributes == nil {
		return ""
	}
	return resource.InstanceState.Attributes["pipeline_id"]
}

func dataPipelinePipelineIDFilter(filters []terraformutils.ResourceFilter) map[string]bool {
	return awsMergeIDFilterValues(
		awsTypedIDFilterValues(filters, dataPipelinePipelineResourceType),
		dataPipelineDefinitionPipelineIDFilter(filters),
	)
}

func dataPipelineDefinitionPipelineIDFilter(filters []terraformutils.ResourceFilter) map[string]bool {
	return awsMergeIDFilterValues(
		awsTypedIDFilterValues(filters, dataPipelinePipelineDefinitionResourceType),
		awsTypedFilterValues(filters, dataPipelinePipelineDefinitionResourceType, "pipeline_id"),
	)
}

func dataPipelineShouldEmitPipeline(filters []terraformutils.ResourceFilter, pipelineID string) bool {
	if !dataPipelineHasTypedFilter(filters) {
		return true
	}
	if !awsHasApplicableFilter(filters, dataPipelinePipelineResourceType) {
		return false
	}
	return awsIDFilterAllows(awsTypedIDFilterValues(filters, dataPipelinePipelineResourceType), pipelineID)
}

func dataPipelineHasTypedFilter(filters []terraformutils.ResourceFilter) bool {
	return awsHasTypedFilter(filters, dataPipelinePipelineResourceType) ||
		awsHasTypedFilter(filters, dataPipelinePipelineDefinitionResourceType)
}

func dataPipelinePipelineImportID(pipelineID string) string {
	return pipelineID
}

func dataPipelinePipelineDefinitionImportID(pipelineID string) string {
	return pipelineID
}

func dataPipelineResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d/%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return "datapipeline-resource"
	}
	return strings.Join(cleanParts, "/")
}
