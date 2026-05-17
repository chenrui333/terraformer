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
	IntegrationAWSAccountAllowEmptyValues = []string{
		"logs_config.",
		"metrics_config.",
		"resources_config.",
		"traces_config.",
		"auth_config.",
		"aws_regions.",
	}
)

type IntegrationAWSAccountGenerator struct {
	DatadogService
}

func (g *IntegrationAWSAccountGenerator) createResource(account datadogV2.AWSAccountResponseData) terraformutils.Resource {
	id := account.GetId()
	resourceName := fmt.Sprintf("integration_aws_account_%s", id)
	attrs := account.GetAttributes()
	if awsAccountID := (&attrs).GetAwsAccountId(); awsAccountID != "" {
		resourceName = fmt.Sprintf("integration_aws_account_%s", awsAccountID)
	}

	return terraformutils.NewSimpleResource(
		id,
		resourceName,
		"datadog_integration_aws_account",
		"datadog",
		IntegrationAWSAccountAllowEmptyValues,
	)
}

func (g *IntegrationAWSAccountGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewAWSIntegrationApi(datadogClient)

	resp, httpResp, err := api.ListAWSAccounts(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}

	resources := []terraformutils.Resource{}
	for _, account := range resp.GetData() {
		if account.GetId() == "" {
			continue
		}
		resources = append(resources, g.createResource(account))
	}
	g.Resources = resources
	return nil
}
