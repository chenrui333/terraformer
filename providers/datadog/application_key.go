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
	// ApplicationKeyAllowEmptyValues ...
	ApplicationKeyAllowEmptyValues = []string{}
)

// ApplicationKeyGenerator ...
type ApplicationKeyGenerator struct {
	DatadogService
}

func (g *ApplicationKeyGenerator) createResource(keyID string) (terraformutils.Resource, error) {
	if keyID == "" {
		return terraformutils.Resource{}, fmt.Errorf("application key missing id")
	}

	return terraformutils.NewSimpleResource(
		keyID,
		fmt.Sprintf("application_key_%s", keyID),
		"datadog_application_key",
		"datadog",
		ApplicationKeyAllowEmptyValues,
	), nil
}

func (g *ApplicationKeyGenerator) createResources(keys []datadogV2.PartialApplicationKey) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, key := range keys {
		resource, err := g.createResource(key.GetId())
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each application_key create 1 TerraformResource.
func (g *ApplicationKeyGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewKeyManagementApi(datadogClient)

	// Handle ID filter
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("application_key") {
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

	// List all application keys with pagination
	keys, err := g.listApplicationKeys(auth, api)
	if err != nil {
		return err
	}

	resources, err := g.createResources(keys)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}

func (g *ApplicationKeyGenerator) listApplicationKeys(auth context.Context, api *datadogV2.KeyManagementApi) ([]datadogV2.PartialApplicationKey, error) {
	var keys []datadogV2.PartialApplicationKey
	pageSize := int64(100)
	pageNumber := int64(0)
	remaining := int64(1)

	for remaining > int64(0) {
		optionalParams := datadogV2.NewListCurrentUserApplicationKeysOptionalParameters().
			WithPageSize(pageSize).
			WithPageNumber(pageNumber)

		resp, httpResp, err := api.ListCurrentUserApplicationKeys(auth, *optionalParams)
		if httpResp != nil && httpResp.Body != nil {
			httpResp.Body.Close()
		}
		if err != nil {
			return nil, err
		}
		keys = append(keys, resp.GetData()...)

		meta := resp.GetMeta()
		page := meta.GetPage()
		if totalFiltered, ok := page.GetTotalFilteredCountOk(); ok {
			remaining = *totalFiltered - pageSize*(pageNumber+1)
		} else {
			remaining = 0
		}
		pageNumber++
	}

	return keys, nil
}
