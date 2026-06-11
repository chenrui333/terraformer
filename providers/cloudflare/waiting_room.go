// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	cf "github.com/chenrui333/terraformer/providers/cloudflare/internal/cloudflarev7"
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
		waitingRooms, err := listWaitingRooms(ctx, api, zone.ID)
		if err != nil {
			return err
		}
		for _, waitingRoom := range waitingRooms {
			waitingRoomResource := terraformutils.NewResource(
				waitingRoom.ID,
				cloudflareResourceName(zone.Name, waitingRoom.Name, waitingRoom.ID),
				"cloudflare_waiting_room",
				"cloudflare",
				map[string]string{"zone_id": zone.ID},
				[]string{},
				map[string]interface{}{},
			)
			setCloudflareImportID(&waitingRoomResource, zone.ID+"/"+waitingRoom.ID)
			g.Resources = append(g.Resources, waitingRoomResource)

			rules, err := api.ListWaitingRoomRules(ctx, cf.ZoneIdentifier(zone.ID), cf.ListWaitingRoomRuleParams{WaitingRoomID: waitingRoom.ID})
			if err != nil {
				return err
			}
			if len(rules) > 0 {
				waitingRoomRulesResource := terraformutils.NewResource(
					waitingRoom.ID,
					cloudflareResourceName(zone.Name, waitingRoom.Name, waitingRoom.ID, "rules"),
					"cloudflare_waiting_room_rules",
					"cloudflare",
					map[string]string{"zone_id": zone.ID, "waiting_room_id": waitingRoom.ID},
					[]string{},
					map[string]interface{}{},
				)
				setCloudflareImportID(&waitingRoomRulesResource, zone.ID+"/"+waitingRoom.ID)
				g.Resources = append(g.Resources, waitingRoomRulesResource)
			}

			events, err := listWaitingRoomEvents(ctx, api, zone.ID, waitingRoom.ID)
			if err != nil {
				return err
			}
			for _, event := range events {
				waitingRoomEventResource := terraformutils.NewResource(
					event.ID,
					cloudflareResourceName(zone.Name, waitingRoom.Name, event.Name, event.ID),
					"cloudflare_waiting_room_event",
					"cloudflare",
					map[string]string{"zone_id": zone.ID, "waiting_room_id": waitingRoom.ID},
					[]string{},
					map[string]interface{}{},
				)
				setCloudflareImportID(&waitingRoomEventResource, zone.ID+"/"+waitingRoom.ID+"/"+event.ID)
				g.Resources = append(g.Resources, waitingRoomEventResource)
			}
		}
	}
	return nil
}

func listWaitingRooms(ctx context.Context, api *cf.API, zoneID string) ([]cf.WaitingRoom, error) {
	var waitingRooms []cf.WaitingRoom
	page, cursor := 1, ""
	for {
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/zones/%s/waiting_rooms?%s", zoneID, cloudflarePaginationQuery(page, cursor)),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pageWaitingRooms []cf.WaitingRoom
		if err := json.Unmarshal(response.Result, &pageWaitingRooms); err != nil {
			return nil, err
		}
		waitingRooms = append(waitingRooms, pageWaitingRooms...)
		if !cloudflareAdvancePagination(response.ResultInfo, &page, &cursor) {
			break
		}
	}
	return waitingRooms, nil
}

func listWaitingRoomEvents(ctx context.Context, api *cf.API, zoneID, waitingRoomID string) ([]cf.WaitingRoomEvent, error) {
	var events []cf.WaitingRoomEvent
	page, cursor := 1, ""
	for {
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/zones/%s/waiting_rooms/%s/events?%s", zoneID, waitingRoomID, cloudflarePaginationQuery(page, cursor)),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pageEvents []cf.WaitingRoomEvent
		if err := json.Unmarshal(response.Result, &pageEvents); err != nil {
			return nil, err
		}
		events = append(events, pageEvents...)
		if !cloudflareAdvancePagination(response.ResultInfo, &page, &cursor) {
			break
		}
	}
	return events, nil
}
