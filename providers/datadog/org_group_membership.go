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
	OrgGroupMembershipAllowEmptyValues = []string{}
)

type OrgGroupMembershipGenerator struct {
	DatadogService
}

func (g *OrgGroupMembershipGenerator) createResource(membership datadogV2.OrgGroupMembershipData) terraformutils.Resource {
	id := membership.GetId().String()
	resourceName := fmt.Sprintf("org_group_membership_%s", id)

	return terraformutils.NewSimpleResource(
		id,
		resourceName,
		"datadog_org_group_membership",
		"datadog",
		OrgGroupMembershipAllowEmptyValues,
	)
}

func (g *OrgGroupMembershipGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)

	datadogClient.GetConfig().SetUnstableOperationEnabled("v2.ListOrgGroups", true)
	datadogClient.GetConfig().SetUnstableOperationEnabled("v2.ListOrgGroupMemberships", true)
	api := datadogV2.NewOrgGroupsApi(datadogClient)

	groups, err := g.listAllOrgGroups(auth, api)
	if err != nil {
		return err
	}

	resources := []terraformutils.Resource{}
	for _, group := range groups {
		groupID := group.GetId()
		if groupID == uuid.Nil {
			continue
		}

		var pageNumber int64
		const pageSize int64 = 100

		for {
			opts := datadogV2.NewListOrgGroupMembershipsOptionalParameters().
				WithFilterOrgGroupId(groupID).
				WithPageNumber(pageNumber).
				WithPageSize(pageSize)
			resp, httpResp, err := api.ListOrgGroupMemberships(auth, *opts)
			if httpResp != nil && httpResp.Body != nil {
				_ = httpResp.Body.Close()
			}
			if err != nil {
				return err
			}

			data := resp.GetData()
			for _, membership := range data {
				if membership.GetId() == uuid.Nil {
					continue
				}
				resources = append(resources, g.createResource(membership))
			}

			if int64(len(data)) < pageSize {
				break
			}
			pageNumber++
		}
	}
	g.Resources = resources
	return nil
}

func (g *OrgGroupMembershipGenerator) listAllOrgGroups(ctx context.Context, api *datadogV2.OrgGroupsApi) ([]datadogV2.OrgGroupData, error) {
	var all []datadogV2.OrgGroupData
	var pageNumber int64
	const pageSize int64 = 100

	for {
		opts := datadogV2.NewListOrgGroupsOptionalParameters().WithPageNumber(pageNumber).WithPageSize(pageSize)
		resp, httpResp, err := api.ListOrgGroups(ctx, *opts)
		if httpResp != nil && httpResp.Body != nil {
			_ = httpResp.Body.Close()
		}
		if err != nil {
			return nil, err
		}

		data := resp.GetData()
		all = append(all, data...)

		if int64(len(data)) < pageSize {
			break
		}
		pageNumber++
	}
	return all, nil
}
