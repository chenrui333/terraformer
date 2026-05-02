// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"

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
		response, err := api.ListWorkerRoutes(ctx, cf.ZoneIdentifier(zone.ID), cf.ListWorkerRoutesParams{})
		if err != nil {
			return err
		}
		for _, route := range response.Routes {
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
