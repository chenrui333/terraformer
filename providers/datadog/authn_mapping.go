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
	// AuthnMappingAllowEmptyValues ...
	AuthnMappingAllowEmptyValues = []string{}
)

// AuthnMappingGenerator ...
type AuthnMappingGenerator struct {
	DatadogService
}

func (g *AuthnMappingGenerator) createResource(mappingID string) (terraformutils.Resource, error) {
	if mappingID == "" {
		return terraformutils.Resource{}, fmt.Errorf("authn mapping missing id")
	}

	return terraformutils.NewSimpleResource(
		mappingID,
		fmt.Sprintf("authn_mapping_%s", mappingID),
		"datadog_authn_mapping",
		"datadog",
		AuthnMappingAllowEmptyValues,
	), nil
}

func (g *AuthnMappingGenerator) createResources(mappings []datadogV2.AuthNMapping) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, mapping := range mappings {
		resource, err := g.createResource(mapping.GetId())
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each authn_mapping create 1 TerraformResource.
func (g *AuthnMappingGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewAuthNMappingsApi(datadogClient)

	// Handle ID filter
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("authn_mapping") {
			var resources []terraformutils.Resource
			for _, value := range filter.AcceptableValues {
				resource, err := g.createResource(value)
				if err != nil {
					return err
				}
				resources = append(resources, resource)
			}
			g.Resources = resources
			return nil
		}
	}

	// List all authn mappings with pagination
	mappings, err := g.listAuthnMappings(auth, api)
	if err != nil {
		return err
	}

	resources, err := g.createResources(mappings)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}

func (g *AuthnMappingGenerator) listAuthnMappings(auth context.Context, api *datadogV2.AuthNMappingsApi) ([]datadogV2.AuthNMapping, error) {
	var mappings []datadogV2.AuthNMapping
	pageSize := int64(100)
	pageNumber := int64(0)
	remaining := int64(1)

	for remaining > int64(0) {
		optionalParams := datadogV2.NewListAuthNMappingsOptionalParameters().
			WithPageSize(pageSize).
			WithPageNumber(pageNumber)

		resp, httpResp, err := api.ListAuthNMappings(auth, *optionalParams)
		if httpResp != nil && httpResp.Body != nil {
			httpResp.Body.Close()
		}
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, resp.GetData()...)

		if meta, ok := resp.GetMetaOk(); ok {
			page := meta.GetPage()
			if totalCount, ok := page.GetTotalCountOk(); ok {
				remaining = *totalCount - pageSize*(pageNumber+1)
			} else {
				remaining = 0
			}
		} else {
			remaining = 0
		}
		pageNumber++
	}

	return mappings, nil
}
