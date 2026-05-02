// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type DNSGenerator struct {
	CloudflareService
}

func (*DNSGenerator) createZonesResource(api *cf.API, zoneID, _ string) ([]terraformutils.Resource, error) {
	zoneDetails, err := api.ZoneDetails(context.Background(), zoneID)
	if err != nil {
		return []terraformutils.Resource{}, err
	}

	resource := terraformutils.NewResource(
		zoneDetails.ID,
		zoneDetails.Name,
		"cloudflare_zone",
		"cloudflare",
		map[string]string{
			"id": zoneDetails.ID,
		},
		[]string{},
		map[string]interface{}{},
	)
	resource.IgnoreKeys = append(resource.IgnoreKeys, "^meta$")

	return []terraformutils.Resource{resource}, nil
}

func (*DNSGenerator) createRecordsResources(api *cf.API, zoneID, zoneName string) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	records, _, err := api.ListDNSRecords(context.Background(), cf.ZoneIdentifier(zoneID), cf.ListDNSRecordsParams{})
	if err != nil {
		return resources, err
	}

	for _, record := range records {
		r := terraformutils.NewResource(
			record.ID,
			fmt.Sprintf("%s_%s_%s", record.Type, zoneName, record.ID),
			"cloudflare_dns_record",
			"cloudflare",
			map[string]string{
				"zone_id": zoneID,
				"name":    record.Name,
			},
			[]string{},
			map[string]interface{}{},
		)

		setCloudflareImportID(&r, zoneID+"/"+record.ID)
		r.IgnoreKeys = append(r.IgnoreKeys, "^metadata")
		resources = append(resources, r)
	}

	return resources, nil
}

func (g *DNSGenerator) InitResources() error {
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}

	zones, err := api.ListZones(context.Background())
	if err != nil {
		return err
	}

	funcs := []func(*cf.API, string, string) ([]terraformutils.Resource, error){
		g.createZonesResource,
		g.createRecordsResources,
	}

	for _, zone := range zones {
		for _, f := range funcs {
			tmpRes, err := f(api, zone.ID, zone.Name)
			if err != nil {
				return err
			}
			g.Resources = append(g.Resources, tmpRes...)
		}
	}
	return nil
}

func (g *DNSGenerator) PostConvertHook() error {
	// 'record' resource have 'data' and 'content' is mutual-exclude
	// delete which one have empty value
	for i, resource := range g.Resources {
		if resource.InstanceInfo.Type == "cloudflare_dns_record" {
			if val, ok := resource.Item["data"].(map[string]interface{}); ok && len(val) == 0 {
				delete(g.Resources[i].Item, "data")
			}
			if val, ok := resource.Item["content"].(string); ok && val == "" {
				delete(g.Resources[i].Item, "content")
			}
		}
	}

	return nil
}
