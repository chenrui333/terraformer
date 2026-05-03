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
	// OnCallEscalationPolicyAllowEmptyValues ...
	OnCallEscalationPolicyAllowEmptyValues = []string{"teams"}
)

// OnCallEscalationPolicyGenerator ...
type OnCallEscalationPolicyGenerator struct {
	DatadogService
}

func (g *OnCallEscalationPolicyGenerator) createResource(onCallEscalationPolicy datadogV2.EscalationPolicy) (terraformutils.Resource, error) {
	data := onCallEscalationPolicy.GetData()
	onCallEscalationPolicyID := data.GetId()
	if onCallEscalationPolicyID == "" {
		return terraformutils.Resource{}, fmt.Errorf("On-Call escalation policy missing id")
	}

	return terraformutils.NewSimpleResource(
		onCallEscalationPolicyID,
		fmt.Sprintf("on_call_escalation_policy_%s", onCallEscalationPolicyID),
		"datadog_on_call_escalation_policy",
		"datadog",
		OnCallEscalationPolicyAllowEmptyValues,
	), nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each On-Call escalation policy create 1 TerraformResource.
// Need On-Call Escalation Policy ID as ID for terraform resource.
func (g *OnCallEscalationPolicyGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewOnCallApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	g.Resources = []terraformutils.Resource{}
	return nil
}

func (g *OnCallEscalationPolicyGenerator) filteredResources(auth context.Context, api *datadogV2.OnCallApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for _, filter := range g.Filter {
		if !filter.IsApplicable("on_call_escalation_policy") || filter.FieldPath != "id" {
			continue
		}

		filtered = true
		for _, value := range filter.AcceptableValues {
			onCallEscalationPolicy, err := getOnCallEscalationPolicy(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(onCallEscalationPolicy)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, filtered, nil
}

func getOnCallEscalationPolicy(auth context.Context, api *datadogV2.OnCallApi, policyID string) (datadogV2.EscalationPolicy, error) {
	include := "steps.targets"
	onCallEscalationPolicy, httpResp, err := api.GetOnCallEscalationPolicy(auth, policyID, datadogV2.GetOnCallEscalationPolicyOptionalParameters{Include: &include})
	if httpResp != nil && httpResp.Body != nil {
		defer httpResp.Body.Close()
	}
	if err != nil {
		return datadogV2.EscalationPolicy{}, err
	}

	data := onCallEscalationPolicy.GetData()
	if data.GetId() == "" {
		data.SetId(policyID)
		onCallEscalationPolicy.SetData(data)
	}
	return onCallEscalationPolicy, nil
}
