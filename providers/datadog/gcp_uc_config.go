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
	GCPUCConfigAllowEmptyValues = []string{}
)

type GCPUCConfigGenerator struct {
	DatadogService
}

func (g *GCPUCConfigGenerator) createResource(config datadogV2.GCPUsageCostConfig) terraformutils.Resource {
	id := config.GetId()
	resourceName := fmt.Sprintf("gcp_uc_config_%s", id)

	return terraformutils.NewSimpleResource(
		id,
		resourceName,
		"datadog_gcp_uc_config",
		"datadog",
		GCPUCConfigAllowEmptyValues,
	)
}

func (g *GCPUCConfigGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewCloudCostManagementApi(datadogClient)

	resp, httpResp, err := api.ListCostGCPUsageCostConfigs(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}

	resources := []terraformutils.Resource{}
	for _, config := range resp.GetData() {
		resources = append(resources, g.createResource(config))
	}
	g.Resources = resources
	return nil
}
