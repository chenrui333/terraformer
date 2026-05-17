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
	IntegrationFastlyServiceAllowEmptyValues = []string{}
)

type IntegrationFastlyServiceGenerator struct {
	DatadogService
}

func (g *IntegrationFastlyServiceGenerator) createResource(accountID, serviceID string) terraformutils.Resource {
	importID := fmt.Sprintf("%s:%s", accountID, serviceID)
	resourceName := fmt.Sprintf("integration_fastly_service_%s_%s", accountID, serviceID)

	return terraformutils.NewResource(
		importID,
		resourceName,
		"datadog_integration_fastly_service",
		"datadog",
		map[string]string{
			"account_id": accountID,
			"service_id": serviceID,
		},
		IntegrationFastlyServiceAllowEmptyValues,
		map[string]interface{}{},
	)
}

func (g *IntegrationFastlyServiceGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewFastlyIntegrationApi(datadogClient)

	accountsResp, httpResp, err := api.ListFastlyAccounts(auth)
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

		servicesResp, httpResp, err := api.ListFastlyServices(auth, accountID)
		if httpResp != nil && httpResp.Body != nil {
			_ = httpResp.Body.Close()
		}
		if err != nil {
			return err
		}

		for _, svc := range servicesResp.GetData() {
			serviceID := svc.GetId()
			if serviceID == "" {
				continue
			}
			resources = append(resources, g.createResource(accountID, serviceID))
		}
	}
	g.Resources = resources
	return nil
}
