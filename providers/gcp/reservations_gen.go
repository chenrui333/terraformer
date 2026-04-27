// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"log"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var reservationsAllowEmptyValues = []string{""}

var reservationsAdditionalFields = map[string]interface{}{}

type ReservationsGenerator struct {
	GCPService
}

// Run on reservationsList and create for each TerraformResource
func (g ReservationsGenerator) createResources(ctx context.Context, reservationsList *compute.ReservationsListCall, zone string) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := reservationsList.Pages(ctx, func(page *compute.ReservationList) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				zone+"/"+obj.Name,
				zone+"/"+obj.Name,
				"google_compute_reservation",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
					"zone":    zone,
				},
				reservationsAllowEmptyValues,
				reservationsAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Generate TerraformResources from GCP API,
// from each reservations create 1 TerraformResource
// Need reservations name as ID for terraform resource
func (g *ReservationsGenerator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}

	for _, zoneLink := range g.GetArgs()["region"].(compute.Region).Zones {
		t := strings.Split(zoneLink, "/")
		zone := t[len(t)-1]
		reservationsList := computeService.Reservations.List(g.GetArgs()["project"].(string), zone)
		g.Resources = append(g.Resources, g.createResources(ctx, reservationsList, zone)...)
	}

	return nil

}
