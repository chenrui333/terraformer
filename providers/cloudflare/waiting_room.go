// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
)

type WaitingRoomGenerator struct {
	CloudflareService
}

func (g *WaitingRoomGenerator) InitResources() error {
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
		waitingRooms, err := api.ListWaitingRooms(ctx, zone.ID)
		if err != nil {
			return err
		}
		for _, waitingRoom := range waitingRooms {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				waitingRoom.ID,
				cloudflareResourceName(zone.Name, waitingRoom.Name, waitingRoom.ID),
				"cloudflare_waiting_room",
				"cloudflare",
				map[string]string{"zone_id": zone.ID},
				[]string{},
				map[string]interface{}{},
			))
			g.Resources = append(g.Resources, terraformutils.NewResource(
				waitingRoom.ID,
				cloudflareResourceName(zone.Name, waitingRoom.Name, waitingRoom.ID, "rules"),
				"cloudflare_waiting_room_rules",
				"cloudflare",
				map[string]string{"zone_id": zone.ID, "waiting_room_id": waitingRoom.ID},
				[]string{},
				map[string]interface{}{},
			))
			events, err := api.ListWaitingRoomEvents(ctx, zone.ID, waitingRoom.ID)
			if err != nil {
				return err
			}
			for _, event := range events {
				g.Resources = append(g.Resources, terraformutils.NewResource(
					event.ID,
					cloudflareResourceName(zone.Name, waitingRoom.Name, event.Name, event.ID),
					"cloudflare_waiting_room_event",
					"cloudflare",
					map[string]string{"zone_id": zone.ID, "waiting_room_id": waitingRoom.ID},
					[]string{},
					map[string]interface{}{},
				))
			}
		}
	}
	return nil
}
