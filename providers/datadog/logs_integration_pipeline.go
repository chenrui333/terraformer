// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// LogsIntegrationPipelineAllowEmptyValues ...
	LogsIntegrationPipelineAllowEmptyValues = []string{}
)

// LogsIntegrationPipelineGenerator ...
type LogsIntegrationPipelineGenerator struct {
	DatadogService
}

func (g *LogsIntegrationPipelineGenerator) createResources(logsIntegrationPipelines []datadogV1.LogsPipeline) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, logsIntegrationPipeline := range logsIntegrationPipelines {
		// Import logs integration pipelines only
		if logsIntegrationPipeline.GetIsReadOnly() {
			resourceID := logsIntegrationPipeline.GetId()
			resourceName := logsIntegrationPipeline.GetName()
			resources = append(resources, g.createResource(resourceID, resourceName))
		}
	}

	return resources
}

func (g *LogsIntegrationPipelineGenerator) createResource(logsIntegrationPipelineID string, logsIntegrationPipelineName string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		logsIntegrationPipelineID,
		logsIntegrationPipelineName,
		"datadog_logs_integration_pipeline",
		"datadog",
		LogsIntegrationPipelineAllowEmptyValues,
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each integration pipeline create 1 TerraformResource.
// Need LogsPipeline ID as ID for terraform resource
func (g *LogsIntegrationPipelineGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV1.NewLogsPipelinesApi(datadogClient)

	logsIntegrationPipelines, _, err := api.ListLogsPipelines(auth)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(logsIntegrationPipelines)
	return nil
}
