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
	// RumApplicationAllowEmptyValues ...
	RumApplicationAllowEmptyValues = []string{}
)

// RumApplicationGenerator ...
type RumApplicationGenerator struct {
	DatadogService
}

func (g *RumApplicationGenerator) createResources(rumApplications []datadogV2.RUMApplicationList) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, rumApplication := range rumApplications {
		applicationID := rumApplication.GetId()
		if applicationID == "" {
			attributes := rumApplication.GetAttributes()
			applicationID = attributes.GetApplicationId()
		}
		if applicationID == "" {
			continue
		}
		resources = append(resources, g.createResource(applicationID))
	}

	return resources
}

func (g *RumApplicationGenerator) createResource(applicationID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		applicationID,
		fmt.Sprintf("rum_application_%s", applicationID),
		"datadog_rum_application",
		"datadog",
		RumApplicationAllowEmptyValues,
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each RUM application create 1 TerraformResource.
func (g *RumApplicationGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewRUMApi(datadogClient)

	resources := []terraformutils.Resource{}
	hasIDFilter := false
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("rum_application") {
			hasIDFilter = true
			for _, value := range filter.AcceptableValues {
				rumApplication, httpResp, err := api.GetRUMApplication(auth, value)
				if httpResp != nil && httpResp.Body != nil {
					_ = httpResp.Body.Close()
				}
				if err != nil {
					return err
				}

				applicationData := rumApplication.GetData()
				applicationID := applicationData.GetId()
				if applicationID == "" {
					applicationID = value
				}
				resources = append(resources, g.createResource(applicationID))
			}
		}
	}

	if hasIDFilter || len(resources) > 0 {
		g.Resources = resources
		return nil
	}

	rumApplications, httpResp, err := api.GetRUMApplications(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}
	g.Resources = g.createResources(rumApplications.GetData())
	return nil
}
