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
	IntegrationMSTeamsTenantBasedHandleAllowEmptyValues = []string{}
)

type IntegrationMSTeamsTenantBasedHandleGenerator struct {
	DatadogService
}

func (g *IntegrationMSTeamsTenantBasedHandleGenerator) createResource(handle datadogV2.MicrosoftTeamsTenantBasedHandleInfoResponseData) terraformutils.Resource {
	id := handle.GetId()
	resourceName := fmt.Sprintf("integration_ms_teams_tenant_based_handle_%s", id)
	attrs := handle.GetAttributes()
	if name := (&attrs).GetName(); name != "" {
		resourceName = fmt.Sprintf("integration_ms_teams_tenant_based_handle_%s_%s", name, id)
	}

	return terraformutils.NewSimpleResource(
		id,
		resourceName,
		"datadog_integration_ms_teams_tenant_based_handle",
		"datadog",
		IntegrationMSTeamsTenantBasedHandleAllowEmptyValues,
	)
}

func (g *IntegrationMSTeamsTenantBasedHandleGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewMicrosoftTeamsIntegrationApi(datadogClient)

	resp, httpResp, err := api.ListTenantBasedHandles(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}

	resources := []terraformutils.Resource{}
	for _, handle := range resp.GetData() {
		if handle.GetId() == "" {
			continue
		}
		resources = append(resources, g.createResource(handle))
	}
	g.Resources = resources
	return nil
}
