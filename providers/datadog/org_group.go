// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/google/uuid"

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

	datadogClient.GetConfig().SetUnstableOperationEnabled("v2.ListOrgGroups", true)
	api := datadogV2.NewOrgGroupsApi(datadogClient)

	resources := []terraformutils.Resource{}
	var pageNumber int64
	const pageSize int64 = 100

	for {
		opts := datadogV2.NewListOrgGroupsOptionalParameters().WithPageNumber(pageNumber).WithPageSize(pageSize)
		resp, httpResp, err := api.ListOrgGroups(auth, *opts)
		if httpResp != nil && httpResp.Body != nil {
			_ = httpResp.Body.Close()
		}
		if err != nil {
			return err
		}

		data := resp.GetData()
		for _, group := range data {
			if group.GetId() == uuid.Nil {
				continue
			}
			resources = append(resources, g.createResource(group))
		}

		if int64(len(data)) < pageSize {
			break
		}
		pageNumber++
	}

	g.Resources = resources
	return nil
}
