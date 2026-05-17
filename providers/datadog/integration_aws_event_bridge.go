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
	IntegrationAWSEventBridgeAllowEmptyValues = []string{}
)

type IntegrationAWSEventBridgeGenerator struct {
	DatadogService
}

func (g *IntegrationAWSEventBridgeGenerator) createResource(sourceName, accountID, region string) terraformutils.Resource {
	return terraformutils.NewResource(
		sourceName,
		fmt.Sprintf("integration_aws_event_bridge_%s", sourceName),
		"datadog_integration_aws_event_bridge",
		"datadog",
		map[string]string{
			"account_id":           accountID,
			"region":               region,
			"event_generator_name": sourceName,
		},
		IntegrationAWSEventBridgeAllowEmptyValues,
		map[string]interface{}{},
	)
}

func (g *IntegrationAWSEventBridgeGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewAWSIntegrationApi(datadogClient)

	resp, httpResp, err := api.ListAWSEventBridgeSources(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}

	resources := []terraformutils.Resource{}
	data := resp.GetData()
	attrs := data.GetAttributes()
	for _, account := range attrs.GetAccounts() {
		accountID := account.GetAccountId()
		for _, source := range account.GetEventHubs() {
			name := source.GetName()
			if name == "" {
				continue
			}
			region := source.GetRegion()
			resources = append(resources, g.createResource(name, accountID, region))
		}
	}
	g.Resources = resources
	return nil
}
