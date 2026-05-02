// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package datadog

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// IntegrationAWSLambdaARNAllowEmptyValues ...
	IntegrationAWSLambdaARNAllowEmptyValues = []string{}
)

// IntegrationAWSLambdaARNGenerator ...
type IntegrationAWSLambdaARNGenerator struct {
	DatadogService
}

func (g *IntegrationAWSLambdaARNGenerator) createResources(logCollections []datadogV1.AWSLogsListResponse) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, logCollection := range logCollections {
		for _, logCollectionLambdaArn := range logCollection.GetLambdas() {
			accountID := logCollection.GetAccountId()
			if v, ok := logCollectionLambdaArn.GetArnOk(); ok {
				resourceID := fmt.Sprintf("%s %s", accountID, *v)
				resources = append(resources, g.createResource(resourceID))
			}
		}
	}
	return resources
}

func (g *IntegrationAWSLambdaARNGenerator) createResource(resourceID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		resourceID,
		fmt.Sprintf("integration_aws_lambda_arn_%s", resourceID),
		"datadog_integration_aws_lambda_arn",
		"datadog",
		IntegrationAWSLambdaARNAllowEmptyValues,
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each monitor create 1 TerraformResource.
// Need IntegrationAWSLambdaARN ID formatted as '<account_id>:<role_name>' as ID for terraform resource
func (g *IntegrationAWSLambdaARNGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV1.NewAWSLogsIntegrationApi(datadogClient)

	logCollections, httpResp, err := api.ListAWSLogsIntegrations(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}
	g.Resources = g.createResources(logCollections)
	return nil
}
