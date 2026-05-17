// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/google/uuid"

	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	datadogOpenapiAPIServiceName = "openapi_api"
	datadogOpenapiAPIPageLimit   = int64(100)
)

var (
	// OpenapiAPIAllowEmptyValues ...
	OpenapiAPIAllowEmptyValues = []string{}
)

// OpenapiAPIGenerator ...
type OpenapiAPIGenerator struct {
	DatadogService
}

func (g *OpenapiAPIGenerator) createResource(apiID string) (terraformutils.Resource, error) {
	return newDatadogIDResource(datadogOpenapiAPIServiceName, apiID, OpenapiAPIAllowEmptyValues)
}

func (g *OpenapiAPIGenerator) createResources(apiIDs []string) ([]terraformutils.Resource, error) {
	return datadogIDResources(datadogOpenapiAPIServiceName, apiIDs, OpenapiAPIAllowEmptyValues)
}

// InitResources Generate TerraformResources from Datadog API,
// from each openapi_api create 1 TerraformResource.
func (g *OpenapiAPIGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	datadogClient.GetConfig().SetUnstableOperationEnabled("v2.GetOpenAPI", true)
	datadogClient.GetConfig().SetUnstableOperationEnabled("v2.ListAPIs", true)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewAPIManagementApi(datadogClient)

	resources, hasIDFilter, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if hasIDFilter {
		g.Resources = resources
		return nil
	}

	apiIDs, err := g.listOpenapiAPIIDs(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(apiIDs)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}

func (g *OpenapiAPIGenerator) filteredResources(auth context.Context, api *datadogV2.APIManagementApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	hasIDFilter := false
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || filter.ServiceName != datadogOpenapiAPIServiceName {
			continue
		}
		hasIDFilter = true
		for _, value := range filter.AcceptableValues {
			apiID, err := g.getOpenapiAPIID(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(apiID)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}
	return resources, hasIDFilter, nil
}

func (g *OpenapiAPIGenerator) getOpenapiAPIID(auth context.Context, api *datadogV2.APIManagementApi, apiID string) (string, error) {
	parsedAPIID, err := uuid.Parse(apiID)
	if err != nil {
		return "", err
	}
	_, httpResp, err := api.GetOpenAPI(auth, parsedAPIID)
	closeDatadogResponseBody(httpResp)
	if err != nil {
		return "", err
	}
	return parsedAPIID.String(), nil
}

func (g *OpenapiAPIGenerator) listOpenapiAPIIDs(auth context.Context, api *datadogV2.APIManagementApi) ([]string, error) {
	ids := []string{}
	offset := int64(0)

	for {
		opts := datadogV2.NewListAPIsOptionalParameters().
			WithPageLimit(datadogOpenapiAPIPageLimit).
			WithPageOffset(offset)

		resp, httpResp, err := api.ListAPIs(auth, *opts)
		closeDatadogResponseBody(httpResp)
		if err != nil {
			return nil, err
		}

		apis := resp.GetData()
		for _, apiData := range apis {
			apiID := apiData.GetId()
			if apiID == uuid.Nil {
				continue
			}
			ids = append(ids, apiID.String())
		}

		meta := resp.GetMeta()
		pagination := meta.GetPagination()
		if totalCount, ok := pagination.GetTotalCountOk(); ok {
			if int64(len(ids)) >= *totalCount {
				break
			}
		} else if int64(len(apis)) < datadogOpenapiAPIPageLimit {
			break
		}
		offset += datadogOpenapiAPIPageLimit
	}

	return ids, nil
}
