// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

// CSMThreatsPolicyGenerator ...
type CSMThreatsPolicyGenerator struct {
	DatadogService
}

func (g *CSMThreatsPolicyGenerator) createResource(data datadogV2.CloudWorkloadSecurityAgentPolicyData) (terraformutils.Resource, error) {
	id := data.GetId()
	if id == "" {
		return terraformutils.Resource{}, fmt.Errorf("CSM threats policy missing id")
	}

	return terraformutils.NewSimpleResource(
		id,
		fmt.Sprintf("csm_threats_policy_%s", id),
		"datadog_csm_threats_policy",
		"datadog",
		[]string{},
	), nil
}

func (g *CSMThreatsPolicyGenerator) createResources(items []datadogV2.CloudWorkloadSecurityAgentPolicyData) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	for _, item := range items {
		resource, err := g.createResource(item)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// InitResources Generate TerraformResources from Datadog API.
func (g *CSMThreatsPolicyGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewCSMThreatsApi(datadogClient)

	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("csm_threats_policy") {
			var resources []terraformutils.Resource
			for _, value := range filter.AcceptableValues {
				resp, httpResp, err := api.GetCSMThreatsAgentPolicy(auth, value)
				if httpResp != nil && httpResp.Body != nil {
					httpResp.Body.Close()
				}
				if err != nil {
					return err
				}
				resource, err := g.createResource(resp.GetData())
				if err != nil {
					return err
				}
				resources = append(resources, resource)
			}
			g.Resources = resources
			return nil
		}
	}

	resp, httpResp, err := api.ListCSMThreatsAgentPolicies(auth)
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		return err
	}

	resources, err := g.createResources(resp.GetData())
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}
