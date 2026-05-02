// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"

	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	cloudInventorySyncConfigByIDPath = "/api/v2/cloudinventoryservice/syncconfigs/%s"
)

var (
	// CloudInventorySyncConfigAllowEmptyValues ...
	CloudInventorySyncConfigAllowEmptyValues = []string{}
)

// CloudInventorySyncConfigGenerator ...
type CloudInventorySyncConfigGenerator struct {
	DatadogService
}

type cloudInventorySyncConfigResponse struct {
	Data cloudInventorySyncConfigResponseDataList `json:"data"`
}

type cloudInventorySyncConfigResponseDataList []cloudInventorySyncConfigResponseData

type cloudInventorySyncConfigResponseData struct {
	ID         string                                     `json:"id"`
	Attributes cloudInventorySyncConfigResponseAttributes `json:"attributes,omitempty"`
}

type cloudInventorySyncConfigResponseAttributes struct {
	CloudProvider string `json:"cloud_provider,omitempty"`
}

const cloudInventorySyncConfigMaxErrorBody = 512

func (d *cloudInventorySyncConfigResponseDataList) UnmarshalJSON(data []byte) error {
	var list []cloudInventorySyncConfigResponseData
	if err := json.Unmarshal(data, &list); err == nil {
		*d = list
		return nil
	}

	var single cloudInventorySyncConfigResponseData
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	*d = []cloudInventorySyncConfigResponseData{single}
	return nil
}

func (g *CloudInventorySyncConfigGenerator) createResource(syncConfig cloudInventorySyncConfigResponseData) terraformutils.Resource {
	resourceName := syncConfig.ID
	if syncConfig.Attributes.CloudProvider != "" {
		resourceName = fmt.Sprintf("%s_%s", syncConfig.Attributes.CloudProvider, syncConfig.ID)
	}

	return terraformutils.NewSimpleResource(
		syncConfig.ID,
		fmt.Sprintf("cloud_inventory_sync_config_%s", resourceName),
		"datadog_cloud_inventory_sync_config",
		"datadog",
		CloudInventorySyncConfigAllowEmptyValues,
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each cloud inventory sync config create 1 TerraformResource.
// Need Cloud Inventory Sync Config ID as ID for terraform resource.
func (g *CloudInventorySyncConfigGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)

	resources := []terraformutils.Resource{}
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("cloud_inventory_sync_config") {
			for _, value := range filter.AcceptableValues {
				syncConfig, err := getCloudInventorySyncConfig(auth, datadogClient, value)
				if err != nil {
					return err
				}
				resources = append(resources, g.createResource(syncConfig))
			}
		}
	}

	if len(resources) > 0 {
		g.Resources = resources
		return nil
	}

	log.Print("Filter(Cloud Inventory Sync Config IDs) is required for importing datadog_cloud_inventory_sync_config resource")
	return nil
}

func getCloudInventorySyncConfig(ctx context.Context, client *datadog.APIClient, id string) (cloudInventorySyncConfigResponseData, error) {
	body, err := sendCloudInventorySyncConfigRequest(ctx, client, fmt.Sprintf(cloudInventorySyncConfigByIDPath, url.PathEscape(id)))
	if err != nil {
		return cloudInventorySyncConfigResponseData{}, err
	}

	var response cloudInventorySyncConfigResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return cloudInventorySyncConfigResponseData{}, err
	}
	if len(response.Data) == 0 {
		return cloudInventorySyncConfigResponseData{}, fmt.Errorf("cloud inventory sync config %q not found", id)
	}
	return response.Data[0], nil
}

// The Datadog Go SDK does not generate a typed Cloud Inventory sync config API,
// so Terraformer mirrors the upstream Terraform provider's raw endpoint request.
func sendCloudInventorySyncConfigRequest(ctx context.Context, client *datadog.APIClient, path string) ([]byte, error) {
	basePath, err := client.GetConfig().ServerURLWithContext(ctx, "")
	if err != nil {
		return nil, err
	}

	headerParams := map[string]string{
		"Accept": "application/json",
	}
	if client.GetConfig().DelegatedTokenConfig != nil {
		if err := datadog.UseDelegatedTokenAuth(ctx, &headerParams, client.GetConfig().DelegatedTokenConfig); err != nil {
			return nil, err
		}
	} else {
		datadog.SetAuthKeys(ctx, &headerParams, [2]string{"apiKeyAuth", "DD-API-KEY"}, [2]string{"appKeyAuth", "DD-APPLICATION-KEY"})
	}

	request, err := client.PrepareRequest(ctx, basePath+path, http.MethodGet, nil, headerParams, url.Values{}, url.Values{}, nil)
	if err != nil {
		return nil, err
	}

	response, err := client.CallAPI(request)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("cloud inventory sync config request failed: empty response")
	}

	body, err := datadog.ReadBody(response)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("cloud inventory sync config request failed: %s: %s", response.Status, truncateCloudInventorySyncConfigErrorBody(body))
	}
	return body, nil
}

func truncateCloudInventorySyncConfigErrorBody(body []byte) string {
	if len(body) <= cloudInventorySyncConfigMaxErrorBody {
		return string(body)
	}
	return string(body[:cloudInventorySyncConfigMaxErrorBody]) + "...(truncated)"
}
