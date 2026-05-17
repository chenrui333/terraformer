// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

// AppSecWafCustomRuleGenerator ...
type AppSecWafCustomRuleGenerator struct {
	DatadogService
}

func (g *AppSecWafCustomRuleGenerator) createResource(data datadogV2.ApplicationSecurityWafCustomRuleData) (terraformutils.Resource, error) {
	id := data.GetId()
	if id == "" {
		return terraformutils.Resource{}, fmt.Errorf("AppSec WAF custom rule missing id")
	}

	return terraformutils.NewSimpleResource(
		id,
		fmt.Sprintf("appsec_waf_custom_rule_%s", id),
		"datadog_appsec_waf_custom_rule",
		"datadog",
		[]string{"blocking", "enabled"},
	), nil
}

func (g *AppSecWafCustomRuleGenerator) createResources(items []datadogV2.ApplicationSecurityWafCustomRuleData) ([]terraformutils.Resource, error) {
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
func (g *AppSecWafCustomRuleGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewApplicationSecurityApi(datadogClient)

	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("appsec_waf_custom_rule") {
			var resources []terraformutils.Resource
			for _, value := range filter.AcceptableValues {
				resp, httpResp, err := api.GetApplicationSecurityWafCustomRule(auth, value)
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

	resp, httpResp, err := api.ListApplicationSecurityWAFCustomRules(auth)
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
