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
	// APMRetentionFilterAllowEmptyValues ...
	APMRetentionFilterAllowEmptyValues = []string{}
)

// APMRetentionFilterGenerator ...
type APMRetentionFilterGenerator struct {
	DatadogService
}

func (g *APMRetentionFilterGenerator) createResources(filters []datadogV2.RetentionFilterAll) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, filter := range filters {
		if !isImportableAPMRetentionFilter(filter) {
			continue
		}
		resourceName := filter.GetId()
		resources = append(resources, g.createResource(resourceName))
	}

	return resources
}

func (g *APMRetentionFilterGenerator) createResource(filterID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		filterID,
		fmt.Sprintf("apm_retention_filter_%s", filterID),
		"datadog_apm_retention_filter",
		"datadog",
		APMRetentionFilterAllowEmptyValues,
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each APM retention filter create 1 TerraformResource.
// Need retention filter ID as ID for terraform resource.
func (g *APMRetentionFilterGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewAPMRetentionFiltersApi(datadogClient)

	resources := []terraformutils.Resource{}
	hasIDFilter := false
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("apm_retention_filter") {
			hasIDFilter = true
			for _, value := range filter.AcceptableValues {
				retentionFilter, httpResp, err := api.GetApmRetentionFilter(auth, value)
				if httpResp != nil && httpResp.Body != nil {
					_ = httpResp.Body.Close()
				}
				if err != nil {
					return err
				}

				filterData := retentionFilter.GetData()
				if !isImportableAPMRetentionFilter(filterData) {
					continue
				}
				resources = append(resources, g.createResource(filterData.GetId()))
			}
		}
	}

	if hasIDFilter || len(resources) > 0 {
		g.Resources = resources
		return nil
	}

	retentionFilters, httpResp, err := api.ListApmRetentionFilters(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}
	g.Resources = g.createResources(retentionFilters.GetData())
	return nil
}

func isImportableAPMRetentionFilter(filter datadogV2.RetentionFilterAll) bool {
	attributes := filter.GetAttributes()
	return attributes.GetFilterType() == datadogV2.RETENTIONFILTERALLTYPE_SPANS_SAMPLING_PROCESSOR
}
