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
	// LogsArchiveAllowEmptyValues ...
	LogsArchiveAllowEmptyValues = []string{"path", "query"}
)

// LogsArchiveGenerator ...
type LogsArchiveGenerator struct {
	DatadogService
}

func (g *LogsArchiveGenerator) createResources(logsArchives []datadogV2.LogsArchiveDefinition) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, logsArchive := range logsArchives {
		logsArchiveID := logsArchive.GetId()
		resources = append(resources, g.createResource(logsArchiveID))
	}

	return resources
}

func (g *LogsArchiveGenerator) createResource(logsArchiveID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		logsArchiveID,
		fmt.Sprintf("logs_archive_%s", logsArchiveID),
		"datadog_logs_archive",
		"datadog",
		LogsArchiveAllowEmptyValues,
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each archive create 1 TerraformResource.
// Need LogsArchive ID as ID for terraform resource
func (g *LogsArchiveGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewLogsArchivesApi(datadogClient)

	resources := []terraformutils.Resource{}
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("logs_archive") {
			for _, value := range filter.AcceptableValues {
				resp, httpResp, err := api.GetLogsArchive(auth, value)
				if httpResp != nil && httpResp.Body != nil {
					_ = httpResp.Body.Close()
				}
				if err != nil {
					return err
				}
				logsArchiveData := resp.GetData()
				resources = append(resources, g.createResource(logsArchiveData.GetId()))
			}
		}
	}

	if len(resources) > 0 {
		g.Resources = resources
		return nil
	}

	logsArchiveListResp, httpResp, err := api.ListLogsArchives(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}
	logsArchiveList := logsArchiveListResp.GetData()
	g.Resources = g.createResources(logsArchiveList)
	return nil
}
