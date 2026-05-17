// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"strconv"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

const syntheticsConcurrencyCapID = "synthetics-concurrency-cap"

var (
	// SyntheticsConcurrencyCapAllowEmptyValues ...
	SyntheticsConcurrencyCapAllowEmptyValues = []string{}
)

// SyntheticsConcurrencyCapGenerator ...
type SyntheticsConcurrencyCapGenerator struct {
	DatadogService
}

func (g *SyntheticsConcurrencyCapGenerator) createResource(resp datadogV2.OnDemandConcurrencyCapResponse) (terraformutils.Resource, error) {
	data, ok := resp.GetDataOk()
	if !ok {
		return terraformutils.Resource{}, fmt.Errorf("synthetics concurrency cap missing data")
	}
	attrs, ok := data.GetAttributesOk()
	if !ok {
		return terraformutils.Resource{}, fmt.Errorf("synthetics concurrency cap missing attributes")
	}
	capValue, ok := attrs.GetOnDemandConcurrencyCapOk()
	if !ok {
		return terraformutils.Resource{}, fmt.Errorf("synthetics concurrency cap missing on_demand_concurrency_cap")
	}

	return terraformutils.NewResource(
		syntheticsConcurrencyCapID,
		"synthetics_concurrency_cap",
		"datadog_synthetics_concurrency_cap",
		"datadog",
		map[string]string{
			"on_demand_concurrency_cap": strconv.FormatInt(int64(*capValue), 10),
		},
		SyntheticsConcurrencyCapAllowEmptyValues,
		map[string]interface{}{},
	), nil
}

// InitResources Generate TerraformResources from Datadog API.
// datadog_synthetics_concurrency_cap is a singleton; the provider read path
// ignores the import ID and stores synthetics-concurrency-cap.
func (g *SyntheticsConcurrencyCapGenerator) InitResources() error {
	for i, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("synthetics_concurrency_cap") {
			g.Filter[i].AcceptableValues = []string{syntheticsConcurrencyCapID}
		}
	}

	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewSyntheticsApi(datadogClient)

	resp, httpResp, err := api.GetOnDemandConcurrencyCap(auth)
	closeDatadogResponseBody(httpResp)
	if err != nil {
		return err
	}

	resource, err := g.createResource(resp)
	if err != nil {
		return err
	}
	g.Resources = []terraformutils.Resource{resource}
	return nil
}
