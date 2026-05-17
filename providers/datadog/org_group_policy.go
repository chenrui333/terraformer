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
	OrgGroupPolicyAllowEmptyValues = []string{}
)

type OrgGroupPolicyGenerator struct {
	DatadogService
}

func (g *OrgGroupPolicyGenerator) createResource(policy datadogV2.OrgGroupPolicyData, orgGroupID string) terraformutils.Resource {
	id := policy.GetId().String()
	resourceName := fmt.Sprintf("org_group_policy_%s", id)

	return terraformutils.NewResource(
		id,
		resourceName,
		"datadog_org_group_policy",
		"datadog",
		map[string]string{
			"org_group_id": orgGroupID,
		},
		OrgGroupPolicyAllowEmptyValues,
		map[string]interface{}{},
	)
}

func (g *OrgGroupPolicyGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)

	datadogClient.GetConfig().SetUnstableOperationEnabled("v2.ListOrgGroups", true)
	datadogClient.GetConfig().SetUnstableOperationEnabled("v2.ListOrgGroupPolicies", true)
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
			opts := datadogV2.NewListOrgGroupPoliciesOptionalParameters().
				WithPageNumber(pageNumber).
				WithPageSize(pageSize)
			resp, httpResp, err := api.ListOrgGroupPolicies(auth, groupID, *opts)
			if httpResp != nil && httpResp.Body != nil {
				_ = httpResp.Body.Close()
			}
			if err != nil {
				return err
			}

			data := resp.GetData()
			for _, policy := range data {
				if policy.GetId() == uuid.Nil {
					continue
				}
				resources = append(resources, g.createResource(policy, groupID.String()))
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

func (g *OrgGroupPolicyGenerator) listAllOrgGroups(ctx context.Context, api *datadogV2.OrgGroupsApi) ([]datadogV2.OrgGroupData, error) {
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
