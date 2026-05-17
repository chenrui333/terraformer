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
	IntegrationConfluentResourceAllowEmptyValues = []string{}
)

type IntegrationConfluentResourceGenerator struct {
	DatadogService
}

func (g *IntegrationConfluentResourceGenerator) createResource(accountID, resourceID string) terraformutils.Resource {
	importID := fmt.Sprintf("%s:%s", accountID, resourceID)
	resourceName := fmt.Sprintf("integration_confluent_resource_%s_%s", accountID, resourceID)

	return terraformutils.NewResource(
		importID,
		resourceName,
		"datadog_integration_confluent_resource",
		"datadog",
		map[string]string{
			"account_id":  accountID,
			"resource_id": resourceID,
		},
		IntegrationConfluentResourceAllowEmptyValues,
		map[string]interface{}{},
	)
}

func (g *IntegrationConfluentResourceGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewConfluentCloudApi(datadogClient)

	accountsResp, httpResp, err := api.ListConfluentAccount(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}

	resources := []terraformutils.Resource{}
	for _, account := range accountsResp.GetData() {
		accountID := account.GetId()
		if accountID == "" {
			continue
		}

		resourcesResp, httpResp, err := api.ListConfluentResource(auth, accountID)
		if httpResp != nil && httpResp.Body != nil {
			_ = httpResp.Body.Close()
		}
		if err != nil {
			return err
		}

		for _, res := range resourcesResp.GetData() {
			resourceID := res.GetId()
			if resourceID == "" {
				continue
			}
			resources = append(resources, g.createResource(accountID, resourceID))
		}
	}
	g.Resources = resources
	return nil
}
