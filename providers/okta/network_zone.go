// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v5/okta"
)

type NetworkZoneGenerator struct {
	OktaService
}

func (g *NetworkZoneGenerator) createResources(networkZoneList []okta.ListNetworkZones200ResponseInner) []terraformutils.Resource {
	var resources []terraformutils.Resource

	for _, networkZone := range networkZoneList {
		var id, name, zoneType *string

		// Handle each type of network zone
		switch {
		case networkZone.DynamicNetworkZone != nil:
			id = networkZone.DynamicNetworkZone.Id
			name = &networkZone.DynamicNetworkZone.Name
			zoneType = &networkZone.DynamicNetworkZone.Type
		case networkZone.EnhancedDynamicNetworkZone != nil:
			id = networkZone.EnhancedDynamicNetworkZone.Id
			name = &networkZone.EnhancedDynamicNetworkZone.Name
			zoneType = &networkZone.EnhancedDynamicNetworkZone.Type
		case networkZone.IPNetworkZone != nil:
			id = networkZone.IPNetworkZone.Id
			name = &networkZone.IPNetworkZone.Name
			zoneType = &networkZone.IPNetworkZone.Type
		default:
			fmt.Println("Unknown or unsupported network zone type encountered")
			continue
		}

		// Ensure all required fields are present before creating the resource
		if id != nil && *name != "" && *zoneType != "" {
			resource := terraformutils.NewSimpleResource(
				*id,
				normalizeResourceName(*id+"_"+*name),
				"okta_network_zone",
				"okta",
				[]string{},
			)
			resources = append(resources, resource)
		}
	}
	return resources
}

func (g *NetworkZoneGenerator) InitResources() error {
	ctx, client, err := g.ClientV5()
	if err != nil {
		return fmt.Errorf("failed to create Okta client: %w", err)
	}

	networkZoneList, resp, err := client.NetworkZoneAPI.ListNetworkZones(ctx).Execute()
	if err != nil {
		return fmt.Errorf("error listing network zones: %w", err)
	}

	allZones := networkZoneList

	for resp.HasNextPage() {
		var nextZoneSet []okta.ListNetworkZones200ResponseInner
		resp, err = resp.Next(&nextZoneSet)
		if err != nil {
			return fmt.Errorf("error fetching next page of network zones: %w", err)
		}
		allZones = append(allZones, nextZoneSet...)
	}

	g.Resources = g.createResources(allZones)
	return nil
}
