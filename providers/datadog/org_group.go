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
	OrgGroupAllowEmptyValues = []string{}
)

type OrgGroupGenerator struct {
	DatadogService
}

func (g *OrgGroupGenerator) createResource(group datadogV2.OrgGroupData) terraformutils.Resource {
	id := group.GetId().String()
	resourceName := fmt.Sprintf("org_group_%s", id)
	attrs := group.GetAttributes()
	if name := (&attrs).GetName(); name != "" {
		resourceName = fmt.Sprintf("org_group_%s", name)
	}

	return terraformutils.NewSimpleResource(
		id,
		resourceName,
		"datadog_org_group",
		"datadog",
		OrgGroupAllowEmptyValues,
	)
}

func (g *OrgGroupGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewOrgGroupsApi(datadogClient)

	resp, httpResp, err := api.ListOrgGroups(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}

	resources := []terraformutils.Resource{}
	for _, group := range resp.GetData() {
		if group.GetId().String() == "00000000-0000-0000-0000-000000000000" {
			continue
		}
		resources = append(resources, g.createResource(group))
	}
	g.Resources = resources
	return nil
}
