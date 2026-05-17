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
	AzureUCConfigAllowEmptyValues = []string{}
)

type AzureUCConfigGenerator struct {
	DatadogService
}

func (g *AzureUCConfigGenerator) createResource(config datadogV2.AzureUCConfigPair) terraformutils.Resource {
	id := config.GetId()
	resourceName := fmt.Sprintf("azure_uc_config_%s", id)

	return terraformutils.NewSimpleResource(
		id,
		resourceName,
		"datadog_azure_uc_config",
		"datadog",
		AzureUCConfigAllowEmptyValues,
	)
}

func (g *AzureUCConfigGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewCloudCostManagementApi(datadogClient)

	resp, httpResp, err := api.ListCostAzureUCConfigs(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}

	resources := []terraformutils.Resource{}
	for _, config := range resp.GetData() {
		if config.GetId() == "" {
			continue
		}
		resources = append(resources, g.createResource(config))
	}
	g.Resources = resources
	return nil
}
