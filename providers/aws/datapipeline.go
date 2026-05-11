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
			if resource, ok := newDataPipelinePipelineResource(pipelineID, pipelineName); ok {
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
