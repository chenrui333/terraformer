// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	datadogDeploymentGateServiceName = "deployment_gate"
	datadogDeploymentGatePageSize    = int64(100)
)

var (
	// DeploymentGateAllowEmptyValues ...
	DeploymentGateAllowEmptyValues = []string{}
)

// DeploymentGateGenerator ...
type DeploymentGateGenerator struct {
	DatadogService
}

func (g *DeploymentGateGenerator) createResource(gateID string) (terraformutils.Resource, error) {
	return newDatadogIDResource(datadogDeploymentGateServiceName, gateID, DeploymentGateAllowEmptyValues)
}

func (g *DeploymentGateGenerator) createResources(gateIDs []string) ([]terraformutils.Resource, error) {
	return datadogIDResources(datadogDeploymentGateServiceName, gateIDs, DeploymentGateAllowEmptyValues)
}

// InitResources Generate TerraformResources from Datadog API,
// from each deployment_gate create 1 TerraformResource.
func (g *DeploymentGateGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	datadogClient.GetConfig().SetUnstableOperationEnabled("v2.GetDeploymentGate", true)
	datadogClient.GetConfig().SetUnstableOperationEnabled("v2.ListDeploymentGates", true)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewDeploymentGatesApi(datadogClient)

	resources, hasIDFilter, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if hasIDFilter {
		g.Resources = resources
		return nil
	}

	gateIDs, err := g.listDeploymentGateIDs(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(gateIDs)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}

func (g *DeploymentGateGenerator) filteredResources(auth context.Context, api *datadogV2.DeploymentGatesApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	hasIDFilter := false
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || filter.ServiceName != datadogDeploymentGateServiceName {
			continue
		}
		hasIDFilter = true
		for _, value := range filter.AcceptableValues {
			gateID, err := g.getDeploymentGateID(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(gateID)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}
	return resources, hasIDFilter, nil
}

func (g *DeploymentGateGenerator) getDeploymentGateID(auth context.Context, api *datadogV2.DeploymentGatesApi, gateID string) (string, error) {
	resp, httpResp, err := api.GetDeploymentGate(auth, gateID)
	closeDatadogResponseBody(httpResp)
	if err != nil {
		return "", err
	}
	data := resp.GetData()
	responseID := data.GetId()
	if responseID == "" {
		return gateID, nil
	}
	return responseID, nil
}

func (g *DeploymentGateGenerator) listDeploymentGateIDs(auth context.Context, api *datadogV2.DeploymentGatesApi) ([]string, error) {
	ids := []string{}
	nextCursor := ""

	for {
		opts := datadogV2.NewListDeploymentGatesOptionalParameters().
			WithPageSize(datadogDeploymentGatePageSize)
		if nextCursor != "" {
			opts.WithPageCursor(nextCursor)
		}

		resp, httpResp, err := api.ListDeploymentGates(auth, *opts)
		closeDatadogResponseBody(httpResp)
		if err != nil {
			return nil, err
		}

		gates := resp.GetData()
		for _, gate := range gates {
			gateID := gate.GetId()
			if gateID == "" {
				continue
			}
			ids = append(ids, gateID)
		}

		meta := resp.GetMeta()
		page := meta.GetPage()
		nextCursor = page.GetNextCursor()
		if nextCursor == "" {
			break
		}
	}

	return ids, nil
}
