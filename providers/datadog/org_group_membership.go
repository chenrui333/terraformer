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

	groupsResp, httpResp, err := api.ListOrgGroups(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}

	resources := []terraformutils.Resource{}
	for _, group := range groupsResp.GetData() {
		groupID := group.GetId()
		if groupID == uuid.Nil {
			continue
		}

		opts := datadogV2.NewListOrgGroupMembershipsOptionalParameters().WithFilterOrgGroupId(groupID)
		membershipsResp, httpResp, err := api.ListOrgGroupMemberships(auth, *opts)
		if httpResp != nil && httpResp.Body != nil {
			_ = httpResp.Body.Close()
		}
		if err != nil {
			return err
		}

		for _, membership := range membershipsResp.GetData() {
			if membership.GetId() == uuid.Nil {
				continue
			}
			resources = append(resources, g.createResource(membership))
		}
	}
	g.Resources = resources
	return nil
}
