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
	// IntegrationAWSAllowEmptyValues ...
	IntegrationAWSAllowEmptyValues = []string{}
)

// IntegrationAWSGenerator ...
type IntegrationAWSGenerator struct {
	DatadogService
}

func (g *IntegrationAWSGenerator) createResources(awsAccounts []datadogV1.AWSAccount) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, account := range awsAccounts {
		resourceID := fmt.Sprintf("%s:%s", account.GetAccountId(), account.GetRoleName())
		resources = append(resources, g.createResource(resourceID))
	}

	return resources
}

func (g *IntegrationAWSGenerator) createResource(resourceID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		resourceID,
		fmt.Sprintf("integration_aws_%s", resourceID),
		"datadog_integration_aws",
		"datadog",
		IntegrationAWSAllowEmptyValues,
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each monitor create 1 TerraformResource.
// Need IntegrationAWS ID formatted as '<account_id>:<role_name>' as ID for terraform resource
func (g *IntegrationAWSGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV1.NewAWSIntegrationApi(datadogClient)

	integrations, httpResp, err := api.ListAWSAccounts(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}
	g.Resources = g.createResources(integrations.GetAccounts())
	return nil
}
