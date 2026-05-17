// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

const datadogSyntheticsSuitePageSize = int64(100)

var (
	// SyntheticsSuiteAllowEmptyValues ...
	SyntheticsSuiteAllowEmptyValues = []string{"tags."}
)

// SyntheticsSuiteGenerator ...
type SyntheticsSuiteGenerator struct {
	DatadogService
}

func (g *SyntheticsSuiteGenerator) createResource(suiteID string) (terraformutils.Resource, error) {
	if suiteID == "" {
		return terraformutils.Resource{}, fmt.Errorf("synthetics suite missing id")
	}

	return terraformutils.NewSimpleResource(
		suiteID,
		fmt.Sprintf("synthetics_suite_%s", suiteID),
		"datadog_synthetics_suite",
		"datadog",
		SyntheticsSuiteAllowEmptyValues,
	), nil
}

func (g *SyntheticsSuiteGenerator) createResources(suites []datadogV2.SyntheticsSuite) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, suite := range suites {
		resource, err := g.createResource(suite.GetPublicId())
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each Synthetics suite create 1 TerraformResource.
func (g *SyntheticsSuiteGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewSyntheticsApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	suites, err := listSyntheticsSuites(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(suites)
	if err != nil {
		return err
	}

	g.Resources = resources
	return nil
}

func (g *SyntheticsSuiteGenerator) filteredResources(auth context.Context, api *datadogV2.SyntheticsApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("synthetics_suite") {
			continue
		}

		filtered = true
		for _, value := range filter.AcceptableValues {
			suite, err := getSyntheticsSuite(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			data := suite.GetData()
			resource, err := g.createResource(data.GetId())
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, filtered, nil
}

func getSyntheticsSuite(auth context.Context, api *datadogV2.SyntheticsApi, suiteID string) (datadogV2.SyntheticsSuiteResponse, error) {
	resp, httpResp, err := api.GetSyntheticsSuite(auth, suiteID)
	closeDatadogResponseBody(httpResp)
	if err != nil {
		return datadogV2.SyntheticsSuiteResponse{}, err
	}
	data := resp.GetData()
	if data.GetId() == "" {
		return datadogV2.SyntheticsSuiteResponse{}, fmt.Errorf("synthetics suite %q not found", suiteID)
	}
	return resp, nil
}

func listSyntheticsSuites(auth context.Context, api *datadogV2.SyntheticsApi) ([]datadogV2.SyntheticsSuite, error) {
	suites := []datadogV2.SyntheticsSuite{}
	start := int64(0)

	for {
		optionalParams := datadogV2.NewSearchSuitesOptionalParameters().
			WithStart(start).
			WithCount(datadogSyntheticsSuitePageSize)

		resp, httpResp, err := api.SearchSuites(auth, *optionalParams)
		closeDatadogResponseBody(httpResp)
		if err != nil {
			return nil, err
		}

		data := resp.GetData()
		attrs := data.GetAttributes()
		pageSuites := attrs.GetSuites()
		suites = append(suites, pageSuites...)

		if len(pageSuites) == 0 || len(pageSuites) < int(datadogSyntheticsSuitePageSize) {
			break
		}
		if total := attrs.GetTotal(); total > 0 && len(suites) >= int(total) {
			break
		}
		start += int64(len(pageSuites))
	}

	return suites, nil
}
