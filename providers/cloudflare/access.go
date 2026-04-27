// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type AccessGenerator struct {
	CloudflareService
}

func (g *AccessGenerator) createAccessApplications(api *cf.API, zoneID string) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	accessApplications, _, err := api.ListAccessApplications(context.Background(), cf.ZoneIdentifier(zoneID), cf.ListAccessApplicationsParams{})
	if err != nil {
		return []terraformutils.Resource{}, err
	}

	for _, app := range accessApplications {
		resources = append(resources, terraformutils.NewResource(
			app.ID,
			fmt.Sprintf("%s_%s", app.Name, app.ID),
			"cloudflare_access_application",
			"cloudflare",
			map[string]string{
				"zone_id": zoneID,
				"name":    app.Name,
			},
			[]string{},
			map[string]interface{}{},
		))
	}

	return resources, nil
}

func (g *AccessGenerator) InitResources() error {
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}

	zones, err := api.ListZones(context.Background())
	if err != nil {
		return err
	}

	for _, zone := range zones {
		tmpRes, err := g.createAccessApplications(api, zone.ID)
		if err != nil {
			return err
		}

		g.Resources = append(g.Resources, tmpRes...)
	}

	return nil
}
