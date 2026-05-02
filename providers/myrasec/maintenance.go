//nolint:revive // lint triage: legacy provider/API/security baseline is tracked in #175.
package myrasec

import (
	"fmt"
	"strconv"

	mgo "github.com/Myra-Security-GmbH/myrasec-go/v2"
	"github.com/chenrui333/terraformer/terraformutils"
)

// MaintenanceGenerator
type MaintenanceGenerator struct {
	MyrasecService
}

// createMaintenanceResources
func (g *MaintenanceGenerator) createMaintenanceResources(api *mgo.API, domainId int, vhost mgo.VHost) error {
	page := 1
	pageSize := 250
	params := map[string]string{
		"page":     strconv.Itoa(page),
		"pageSize": strconv.Itoa(pageSize),
	}

	for {
		params["page"] = strconv.Itoa(page)

		maintenance, err := api.ListMaintenances(domainId, vhost.Label, params)
		if err != nil {
			return err
		}

		for _, m := range maintenance {
			r := terraformutils.NewResource(
				strconv.Itoa(m.ID),
				fmt.Sprintf("%s_%d", vhost.Label, m.ID),
				"myrasec_maintenance",
				"myrasec",
				map[string]string{
					"subdomain_name": vhost.Label,
				},
				[]string{},
				map[string]interface{}{},
			)
			g.appendResource(r)
		}
		if len(maintenance) < pageSize {
			break
		}
		page++
	}
	return nil
}

// InitResources
func (g *MaintenanceGenerator) InitResources() error {
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}

	funcs := []func(*mgo.API, int, mgo.VHost) error{
		g.createMaintenanceResources,
	}

	err = createResourcesPerSubDomain(api, funcs, false)
	if err != nil {
		return err
	}

	return nil
}
