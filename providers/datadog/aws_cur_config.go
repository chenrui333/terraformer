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
	AwsCURConfigAllowEmptyValues = []string{}
)

type AwsCURConfigGenerator struct {
	DatadogService
}

func (g *AwsCURConfigGenerator) createResource(config datadogV2.AwsCURConfig) terraformutils.Resource {
	id := config.GetId()
	resourceName := fmt.Sprintf("aws_cur_config_%s", id)
	attrs := config.GetAttributes()
	if accountID := attrs.GetAccountId(); accountID != "" {
		resourceName = fmt.Sprintf("aws_cur_config_%s", accountID)
	}

	return terraformutils.NewSimpleResource(
		id,
		resourceName,
		"datadog_aws_cur_config",
		"datadog",
		AwsCURConfigAllowEmptyValues,
	)
}

func (g *AwsCURConfigGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewCloudCostManagementApi(datadogClient)

	resp, httpResp, err := api.ListCostAWSCURConfigs(auth)
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
