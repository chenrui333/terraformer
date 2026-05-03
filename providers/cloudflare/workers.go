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

type WorkersGenerator struct {
	CloudflareService
}

func (g *WorkersGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	zones, err := cloudflareZones(ctx, api)
	if err != nil {
		return err
	}
	for _, zone := range zones {
		routes, err := listWorkerRoutes(ctx, api, zone.ID)
		if err != nil {
			return err
		}
		for _, route := range routes {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				route.ID,
				cloudflareResourceName(zone.Name, route.Pattern, route.ID),
				"cloudflare_workers_route",
				"cloudflare",
				map[string]string{"zone_id": zone.ID},
				[]string{},
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func listWorkerRoutes(ctx context.Context, api *cf.API, zoneID string) ([]cf.WorkerRoute, error) {
	var routes []cf.WorkerRoute
	page, cursor := 1, ""
	for {
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/zones/%s/workers/routes?%s", zoneID, cloudflarePaginationQuery(page, cursor)),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pageRoutes []cf.WorkerRoute
		if err := json.Unmarshal(response.Result, &pageRoutes); err != nil {
			return nil, err
		}
		routes = append(routes, pageRoutes...)
		if !cloudflareAdvancePagination(response.ResultInfo, &page, &cursor) {
			break
		}
	}
	return routes, nil
}
