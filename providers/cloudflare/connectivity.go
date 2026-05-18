// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type ConnectivityGenerator struct {
	CloudflareService
}

type cloudflareConnectivityDirectoryService struct {
	ID        string `json:"id"`
	ServiceID string `json:"service_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
}

func (s cloudflareConnectivityDirectoryService) resourceID() string {
	if s.ServiceID != "" {
		return s.ServiceID
	}
	return s.ID
}

func (g *ConnectivityGenerator) appendConnectivityDirectoryServiceResources(ctx context.Context, api *cf.API, accountID string) error {
	services, err := listConnectivityDirectoryServices(ctx, api, accountID)
	if err != nil {
		return err
	}
	for _, service := range services {
		resource, ok := cloudflareConnectivityDirectoryServiceResource(accountID, service)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func listConnectivityDirectoryServices(ctx context.Context, api *cf.API, accountID string) ([]cloudflareConnectivityDirectoryService, error) {
	var services []cloudflareConnectivityDirectoryService
	page, cursor := 1, ""
	for {
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/accounts/%s/connectivity/directory/services?%s", accountID, cloudflarePaginationQuery(page, cursor)),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		if len(response.Result) == 0 || string(response.Result) == "null" {
			return services, nil
		}

		var pageServices []cloudflareConnectivityDirectoryService
		if err := json.Unmarshal(response.Result, &pageServices); err != nil {
			return nil, err
		}
		services = append(services, pageServices...)
		if !cloudflareAdvancePaginationWithItemCount(response.ResultInfo, &page, &cursor, len(pageServices)) {
			break
		}
	}
	return services, nil
}

func cloudflareConnectivityDirectoryServiceResource(
	accountID string,
	service cloudflareConnectivityDirectoryService,
) (terraformutils.Resource, bool) {
	serviceID := service.resourceID()
	if accountID == "" || serviceID == "" {
		return terraformutils.Resource{}, false
	}

	attributes := map[string]string{
		"account_id": accountID,
		"service_id": serviceID,
	}
	if service.Name != "" {
		attributes["name"] = service.Name
	}
	if service.Type != "" {
		attributes["type"] = service.Type
	}

	resource := terraformutils.NewResource(
		serviceID,
		cloudflareResourceName(accountID, "connectivity_directory_service", service.Name, serviceID),
		"cloudflare_connectivity_directory_service",
		"cloudflare",
		attributes,
		[]string{},
		map[string]interface{}{},
	)
	setCloudflareImportID(&resource, accountID+"/"+serviceID)
	return resource, true
}

func (g *ConnectivityGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	account, err := g.accountResourceContainer()
	if err != nil {
		return err
	}
	return g.appendConnectivityDirectoryServiceResources(ctx, api, account.Identifier)
}
