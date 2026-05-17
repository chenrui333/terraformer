// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

// AppSecWafExclusionFilterGenerator ...
type AppSecWafExclusionFilterGenerator struct {
	DatadogService
}

func (g *AppSecWafExclusionFilterGenerator) createResource(data datadogV2.ApplicationSecurityWafExclusionFilterResource) (terraformutils.Resource, error) {
	id := data.GetId()
	if id == "" {
		return terraformutils.Resource{}, fmt.Errorf("AppSec WAF exclusion filter missing id")
	}

	return terraformutils.NewSimpleResource(
		id,
		fmt.Sprintf("appsec_waf_exclusion_filter_%s", id),
		"datadog_appsec_waf_exclusion_filter",
		"datadog",
		[]string{"enabled"},
	), nil
}

func (g *AppSecWafExclusionFilterGenerator) createResources(items []datadogV2.ApplicationSecurityWafExclusionFilterResource) ([]terraformutils.Resource, error) {
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
func (g *AppSecWafExclusionFilterGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewApplicationSecurityApi(datadogClient)

	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("appsec_waf_exclusion_filter") {
			var resources []terraformutils.Resource
			for _, value := range filter.AcceptableValues {
				resp, httpResp, err := api.GetApplicationSecurityWafExclusionFilter(auth, value)
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

	resp, httpResp, err := api.ListApplicationSecurityWafExclusionFilters(auth)
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
