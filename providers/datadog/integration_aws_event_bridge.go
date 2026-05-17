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

func (g *IntegrationAWSEventBridgeGenerator) createResource(sourceName string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		sourceName,
		fmt.Sprintf("integration_aws_event_bridge_%s", sourceName),
		"datadog_integration_aws_event_bridge",
		"datadog",
		IntegrationAWSEventBridgeAllowEmptyValues,
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
		for _, source := range account.GetEventHubs() {
			name := source.GetName()
			if name == "" {
				continue
			}
			resources = append(resources, g.createResource(name))
		}
	}
	g.Resources = resources
	return nil
}
