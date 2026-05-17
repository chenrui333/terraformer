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
	api := datadogV2.NewOrgGroupsApi(datadogClient)

	resp, httpResp, err := api.ListOrgGroupMemberships(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}

	resources := []terraformutils.Resource{}
	for _, membership := range resp.GetData() {
		if membership.GetId().String() == "00000000-0000-0000-0000-000000000000" {
			continue
		}
		resources = append(resources, g.createResource(membership))
	}
	g.Resources = resources
	return nil
}
