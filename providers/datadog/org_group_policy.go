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

		policiesResp, httpResp, err := api.ListOrgGroupPolicies(auth, groupID)
		if httpResp != nil && httpResp.Body != nil {
			_ = httpResp.Body.Close()
		}
		if err != nil {
			return err
		}

		for _, policy := range policiesResp.GetData() {
			if policy.GetId() == uuid.Nil {
				continue
			}
			resources = append(resources, g.createResource(policy, groupID.String()))
		}
	}
	g.Resources = resources
	return nil
}
