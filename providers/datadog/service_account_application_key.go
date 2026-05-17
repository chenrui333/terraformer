// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// ServiceAccountApplicationKeyAllowEmptyValues ...
	ServiceAccountApplicationKeyAllowEmptyValues = []string{}
)

// ServiceAccountApplicationKeyGenerator ...
type ServiceAccountApplicationKeyGenerator struct {
	DatadogService
}

func (g *ServiceAccountApplicationKeyGenerator) createResource(serviceAccountID, keyID string) (terraformutils.Resource, error) {
	if keyID == "" {
		return terraformutils.Resource{}, fmt.Errorf("service account application key missing id")
	}
	if serviceAccountID == "" {
		return terraformutils.Resource{}, fmt.Errorf("service account application key missing service_account_id")
	}

	return terraformutils.NewResource(
		keyID,
		fmt.Sprintf("service_account_application_key_%s_%s", serviceAccountID, keyID),
		"datadog_service_account_application_key",
		"datadog",
		map[string]string{
			"service_account_id": serviceAccountID,
		},
		ServiceAccountApplicationKeyAllowEmptyValues,
		map[string]interface{}{},
	), nil
}

func (g *ServiceAccountApplicationKeyGenerator) createResources(serviceAccountID string, keys []datadogV2.PartialApplicationKey) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, key := range keys {
		resource, err := g.createResource(serviceAccountID, key.GetId())
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each service_account_application_key create 1 TerraformResource.
// Requires service_account_id filter.
func (g *ServiceAccountApplicationKeyGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewServiceAccountsApi(datadogClient)

	for filterIndex, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("service_account_application_key") {
			var resources []terraformutils.Resource
			var keyIDs []string
			for _, value := range filter.AcceptableValues {
				serviceAccountID, keyID := parseServiceAccountApplicationKeyFilterID(value)
				if serviceAccountID == "" {
					return fmt.Errorf("service account application key filter value %q must be in service_account_id:key_id format", value)
				}
				resource, err := g.createResource(serviceAccountID, keyID)
				if err != nil {
					return err
				}
				resources = append(resources, resource)
				keyIDs = append(keyIDs, keyID)
			}
			g.Filter[filterIndex].AcceptableValues = keyIDs
			g.Resources = resources
			return nil
		}
	}

	// Check for service_account_id filter
	for _, filter := range g.Filter {
		if filter.FieldPath == "service_account_id" && filter.IsApplicable("service_account_application_key") {
			var resources []terraformutils.Resource
			for _, serviceAccountID := range filter.AcceptableValues {
				keys, err := g.listServiceAccountApplicationKeys(auth, api, serviceAccountID)
				if err != nil {
					return err
				}
				r, err := g.createResources(serviceAccountID, keys)
				if err != nil {
					return err
				}
				resources = append(resources, r...)
			}
			g.Resources = resources
			return nil
		}
	}

	log.Print("Filter(service_account_id or composite service_account_id:key_id) is required for importing datadog_service_account_application_key resource")
	return nil
}

func (g *ServiceAccountApplicationKeyGenerator) listServiceAccountApplicationKeys(auth context.Context, api *datadogV2.ServiceAccountsApi, serviceAccountID string) ([]datadogV2.PartialApplicationKey, error) {
	var keys []datadogV2.PartialApplicationKey
	pageSize := int64(100)
	pageNumber := int64(0)
	remaining := int64(1)

	for remaining > int64(0) {
		optionalParams := datadogV2.NewListServiceAccountApplicationKeysOptionalParameters().
			WithPageSize(pageSize).
			WithPageNumber(pageNumber)

		resp, httpResp, err := api.ListServiceAccountApplicationKeys(auth, serviceAccountID, *optionalParams)
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

func parseServiceAccountApplicationKeyFilterID(value string) (serviceAccountID, keyID string) {
	if idx := strings.IndexByte(value, ':'); idx >= 0 {
		return value[:idx], value[idx+1:]
	}
	return "", value
}
