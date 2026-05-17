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
	// ServiceDefinitionYAMLAllowEmptyValues ...
	ServiceDefinitionYAMLAllowEmptyValues = []string{}
)

// ServiceDefinitionYAMLGenerator ...
type ServiceDefinitionYAMLGenerator struct {
	DatadogService
}

func (g *ServiceDefinitionYAMLGenerator) createResource(serviceName string) (terraformutils.Resource, error) {
	if serviceName == "" {
		return terraformutils.Resource{}, fmt.Errorf("service definition missing id")
	}

	return terraformutils.NewSimpleResource(
		serviceName,
		fmt.Sprintf("service_definition_yaml_%s", serviceName),
		"datadog_service_definition_yaml",
		"datadog",
		ServiceDefinitionYAMLAllowEmptyValues,
	), nil
}

func (g *ServiceDefinitionYAMLGenerator) createResources(serviceDefinitions []datadogV2.ServiceDefinitionData) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, serviceDefinition := range serviceDefinitions {
		resource, err := g.createResource(serviceDefinition.GetId())
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each service_definition_yaml create 1 TerraformResource.
// Need Service Definition service name as ID for terraform resource.
func (g *ServiceDefinitionYAMLGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewServiceDefinitionApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	serviceDefinitions, err := listServiceDefinitions(auth, api)
	if err != nil {
		return err
	}
	g.Resources, err = g.createResources(serviceDefinitions)
	return err
}

func (g *ServiceDefinitionYAMLGenerator) filteredResources(auth context.Context, api *datadogV2.ServiceDefinitionApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	matchedIDFilter := false
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("service_definition_yaml") {
			continue
		}
		matchedIDFilter = true
		for _, value := range filter.AcceptableValues {
			serviceDefinition, httpResp, err := api.GetServiceDefinition(auth, value)
			closeDatadogResponseBody(httpResp)
			if err != nil {
				return nil, false, err
			}

			serviceDefinitionData := serviceDefinition.GetData()
			serviceName := serviceDefinitionData.GetId()
			if serviceName == "" {
				serviceName = value
			}
			resource, err := g.createResource(serviceName)
			if err != nil {
				return nil, false, err
			}
			resources = append(resources, resource)
		}
	}
	return resources, matchedIDFilter, nil
}

func listServiceDefinitions(auth context.Context, api *datadogV2.ServiceDefinitionApi) ([]datadogV2.ServiceDefinitionData, error) {
	pageSize := int64(100)
	items, cancel := api.ListServiceDefinitionsWithPagination(auth, *datadogV2.NewListServiceDefinitionsOptionalParameters().WithPageSize(pageSize))
	defer cancel()

	serviceDefinitions := []datadogV2.ServiceDefinitionData{}
	for item := range items {
		if item.Error != nil {
			return nil, item.Error
		}
		serviceDefinitions = append(serviceDefinitions, item.Item)
	}
	return serviceDefinitions, nil
}
