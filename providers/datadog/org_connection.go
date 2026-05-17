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
	OrgConnectionAllowEmptyValues = []string{}
)

type OrgConnectionGenerator struct {
	DatadogService
}

func (g *OrgConnectionGenerator) createResource(conn datadogV2.OrgConnection) terraformutils.Resource {
	id := conn.GetId().String()
	resourceName := fmt.Sprintf("org_connection_%s", id)

	return terraformutils.NewSimpleResource(
		id,
		resourceName,
		"datadog_org_connection",
		"datadog",
		OrgConnectionAllowEmptyValues,
	)
}

func (g *OrgConnectionGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewOrgConnectionsApi(datadogClient)

	resp, httpResp, err := api.ListOrgConnections(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}

	resources := []terraformutils.Resource{}
	for _, conn := range resp.GetData() {
		if conn.GetId().String() == "00000000-0000-0000-0000-000000000000" {
			continue
		}
		resources = append(resources, g.createResource(conn))
	}
	g.Resources = resources
	return nil
}
