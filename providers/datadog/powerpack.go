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
	// PowerpackAllowEmptyValues ...
	PowerpackAllowEmptyValues = []string{"tags."}
	// PowerpackV2AllowEmptyValues ...
	PowerpackV2AllowEmptyValues = []string{"tags."}
)

// PowerpackGenerator ...
type PowerpackGenerator struct {
	DatadogService
}

// PowerpackV2Generator ...
type PowerpackV2Generator struct {
	DatadogService
}

type powerpackResourceConfig struct {
	serviceName      string
	resourceNameBase string
	resourceType     string
	allowEmptyValues []string
}

func (g *PowerpackGenerator) createResources(powerpacks []datadogV2.PowerpackData) ([]terraformutils.Resource, error) {
	return createPowerpackResources(powerpacks, powerpackResourceConfig{
		serviceName:      "powerpack",
		resourceNameBase: "powerpack",
		resourceType:     "datadog_powerpack",
		allowEmptyValues: PowerpackAllowEmptyValues,
	})
}

func (g *PowerpackGenerator) createResource(powerpack datadogV2.PowerpackData) (terraformutils.Resource, error) {
	return createPowerpackResource(powerpack, powerpackResourceConfig{
		serviceName:      "powerpack",
		resourceNameBase: "powerpack",
		resourceType:     "datadog_powerpack",
		allowEmptyValues: PowerpackAllowEmptyValues,
	})
}

// InitResources Generate TerraformResources from Datadog API,
// from each powerpack create 1 TerraformResource.
func (g *PowerpackGenerator) InitResources() error {
	return initPowerpackResources(&g.DatadogService, "powerpack", g.createResource, g.createResources)
}

func (g *PowerpackV2Generator) createResources(powerpacks []datadogV2.PowerpackData) ([]terraformutils.Resource, error) {
	return createPowerpackResources(powerpacks, powerpackResourceConfig{
		serviceName:      "powerpack_v2",
		resourceNameBase: "powerpack_v2",
		resourceType:     "datadog_powerpack_v2",
		allowEmptyValues: PowerpackV2AllowEmptyValues,
	})
}

func (g *PowerpackV2Generator) createResource(powerpack datadogV2.PowerpackData) (terraformutils.Resource, error) {
	return createPowerpackResource(powerpack, powerpackResourceConfig{
		serviceName:      "powerpack_v2",
		resourceNameBase: "powerpack_v2",
		resourceType:     "datadog_powerpack_v2",
		allowEmptyValues: PowerpackV2AllowEmptyValues,
	})
}

// InitResources Generate TerraformResources from Datadog API,
// from each powerpack_v2 create 1 TerraformResource.
func (g *PowerpackV2Generator) InitResources() error {
	return initPowerpackResources(&g.DatadogService, "powerpack_v2", g.createResource, g.createResources)
}

func createPowerpackResources(powerpacks []datadogV2.PowerpackData, config powerpackResourceConfig) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, powerpack := range powerpacks {
		resource, err := createPowerpackResource(powerpack, config)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func createPowerpackResource(powerpack datadogV2.PowerpackData, config powerpackResourceConfig) (terraformutils.Resource, error) {
	powerpackID := powerpackID(powerpack)
	if powerpackID == "" {
		return terraformutils.Resource{}, fmt.Errorf("%s missing id", config.serviceName)
	}

	return terraformutils.NewSimpleResource(
		powerpackID,
		fmt.Sprintf("%s_%s", config.resourceNameBase, powerpackID),
		config.resourceType,
		"datadog",
		config.allowEmptyValues,
	), nil
}

func initPowerpackResources(
	service *DatadogService,
	serviceName string,
	createResource func(datadogV2.PowerpackData) (terraformutils.Resource, error),
	createResources func([]datadogV2.PowerpackData) ([]terraformutils.Resource, error),
) error {
	datadogClient := service.Args["datadogClient"].(*datadog.APIClient)
	auth := service.Args["auth"].(context.Context)
	api := datadogV2.NewPowerpackApi(datadogClient)

	resources := []terraformutils.Resource{}
	for _, filter := range service.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable(serviceName) {
			continue
		}

		for _, value := range filter.AcceptableValues {
			powerpack, err := getPowerpack(auth, api, value)
			if err != nil {
				return err
			}
			resource, err := createResource(powerpack)
			if err != nil {
				return err
			}
			resources = append(resources, resource)
		}
	}

	if len(resources) > 0 {
		service.Resources = resources
		return nil
	}

	powerpacks, err := listPowerpacks(auth, api)
	if err != nil {
		return err
	}
	resources, err = createResources(powerpacks)
	if err != nil {
		return err
	}

	service.Resources = resources
	return nil
}

func getPowerpack(auth context.Context, api *datadogV2.PowerpackApi, powerpackID string) (datadogV2.PowerpackData, error) {
	response, httpResp, err := api.GetPowerpack(auth, powerpackID)
	closeDatadogResponseBody(httpResp)
	if err != nil {
		return datadogV2.PowerpackData{}, err
	}

	if powerpack, ok := response.GetDataOk(); ok {
		return *powerpack, nil
	}
	if powerpack, ok := powerpackFromRawData(response.UnparsedObject["data"]); ok {
		return powerpack, nil
	}

	return datadogV2.PowerpackData{}, fmt.Errorf("powerpack %q not found", powerpackID)
}

func listPowerpacks(auth context.Context, api *datadogV2.PowerpackApi) ([]datadogV2.PowerpackData, error) {
	pageSize := int64(100)
	items, cancel := api.ListPowerpacksWithPagination(auth, *datadogV2.NewListPowerpacksOptionalParameters().WithPageLimit(pageSize))
	defer cancel()

	powerpacks := []datadogV2.PowerpackData{}
	for item := range items {
		if item.Error != nil {
			return nil, item.Error
		}
		powerpacks = append(powerpacks, item.Item)
	}

	return powerpacks, nil
}

func powerpackID(powerpack datadogV2.PowerpackData) string {
	if id := powerpack.GetId(); id != "" {
		return id
	}
	if id, ok := powerpack.UnparsedObject["id"].(string); ok {
		return id
	}
	return ""
}

func powerpackFromRawData(rawData interface{}) (datadogV2.PowerpackData, bool) {
	rawPowerpack, ok := rawData.(map[string]interface{})
	if !ok {
		return datadogV2.PowerpackData{}, false
	}

	rawPowerpackID, ok := rawPowerpack["id"].(string)
	if !ok || rawPowerpackID == "" {
		return datadogV2.PowerpackData{}, false
	}

	powerpack := datadogV2.NewPowerpackDataWithDefaults()
	powerpack.SetId(rawPowerpackID)
	return *powerpack, true
}
