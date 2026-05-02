// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// SLOCorrectionAllowEmptyValues ...
	SLOCorrectionAllowEmptyValues = []string{}
)

// SLOCorrectionGenerator ...
type SLOCorrectionGenerator struct {
	DatadogService
}

func (g *SLOCorrectionGenerator) createResources(sloCorrections []datadogV1.SLOCorrection) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, sloCorrection := range sloCorrections {
		resource, err := g.createResource(sloCorrection)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *SLOCorrectionGenerator) createResource(sloCorrection datadogV1.SLOCorrection) (terraformutils.Resource, error) {
	sloCorrectionID := sloCorrection.GetId()
	if sloCorrectionID == "" {
		return terraformutils.Resource{}, fmt.Errorf("slo correction missing id")
	}

	return terraformutils.NewSimpleResource(
		sloCorrectionID,
		fmt.Sprintf("slo_correction_%s", sloCorrectionID),
		"datadog_slo_correction",
		"datadog",
		SLOCorrectionAllowEmptyValues,
	), nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each slo_correction create 1 TerraformResource.
func (g *SLOCorrectionGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV1.NewServiceLevelObjectiveCorrectionsApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	sloCorrections, err := listSLOCorrections(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(sloCorrections)
	if err != nil {
		return err
	}

	g.Resources = resources
	return nil
}

func (g *SLOCorrectionGenerator) filteredResources(auth context.Context, api *datadogV1.ServiceLevelObjectiveCorrectionsApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("slo_correction") {
			continue
		}

		filtered = true
		for _, value := range filter.AcceptableValues {
			sloCorrection, err := getSLOCorrection(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(sloCorrection)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, filtered, nil
}

func getSLOCorrection(auth context.Context, api *datadogV1.ServiceLevelObjectiveCorrectionsApi, sloCorrectionID string) (datadogV1.SLOCorrection, error) {
	response, httpResponse, err := api.GetSLOCorrection(auth, sloCorrectionID)
	defer closeDatadogResponseBody(httpResponse)
	if err != nil {
		return datadogV1.SLOCorrection{}, err
	}

	if sloCorrection, ok := response.GetDataOk(); ok {
		return *sloCorrection, nil
	}
	if sloCorrection, ok := sloCorrectionFromRawData(response.UnparsedObject["data"]); ok {
		return sloCorrection, nil
	}

	return datadogV1.SLOCorrection{}, fmt.Errorf("slo correction %q not found", sloCorrectionID)
}

func listSLOCorrections(auth context.Context, api *datadogV1.ServiceLevelObjectiveCorrectionsApi) ([]datadogV1.SLOCorrection, error) {
	sloCorrections := []datadogV1.SLOCorrection{}
	const pageSize int64 = 100
	var offset int64

	for {
		optionalParams := datadogV1.NewListSLOCorrectionOptionalParameters().
			WithLimit(pageSize).
			WithOffset(offset)

		response, httpResponse, err := api.ListSLOCorrection(auth, *optionalParams)
		closeDatadogResponseBody(httpResponse)
		if err != nil {
			return nil, err
		}

		pageCorrections := response.GetData()
		if len(pageCorrections) == 0 {
			pageCorrections = sloCorrectionsFromRawData(response.UnparsedObject["data"])
		}
		sloCorrections = append(sloCorrections, pageCorrections...)

		if len(pageCorrections) < int(pageSize) {
			break
		}
		offset += pageSize
	}

	return sloCorrections, nil
}

func sloCorrectionsFromRawData(rawData interface{}) []datadogV1.SLOCorrection {
	rawCorrections, ok := rawData.([]interface{})
	if !ok {
		return nil
	}

	sloCorrections := []datadogV1.SLOCorrection{}
	for _, rawCorrection := range rawCorrections {
		sloCorrection, ok := sloCorrectionFromRawData(rawCorrection)
		if !ok {
			continue
		}
		sloCorrections = append(sloCorrections, sloCorrection)
	}
	return sloCorrections
}

func sloCorrectionFromRawData(rawData interface{}) (datadogV1.SLOCorrection, bool) {
	rawCorrection, ok := rawData.(map[string]interface{})
	if !ok {
		return datadogV1.SLOCorrection{}, false
	}
	if rawType, ok := rawCorrection["type"].(string); ok && rawType != string(datadogV1.SLOCORRECTIONTYPE_CORRECTION) {
		return datadogV1.SLOCorrection{}, false
	}
	rawID, ok := rawCorrection["id"].(string)
	if !ok || rawID == "" {
		return datadogV1.SLOCorrection{}, false
	}

	sloCorrection := datadogV1.NewSLOCorrectionWithDefaults()
	sloCorrection.SetId(rawID)
	return *sloCorrection, true
}
