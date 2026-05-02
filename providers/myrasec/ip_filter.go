//nolint:revive // lint triage: legacy provider/API/security baseline is tracked in #175.
package myrasec

import (
	"fmt"
	"strconv"

	mgo "github.com/Myra-Security-GmbH/myrasec-go/v2"
	"github.com/chenrui333/terraformer/terraformutils"
)

// IPFilterGenerator
type IPFilterGenerator struct {
	MyrasecService
}

// createIPFilterResources
func (g *IPFilterGenerator) createIPFilterResources(api *mgo.API, domainId int, vhost mgo.VHost) error {
	page := 1
	pageSize := 250
	params := map[string]string{
		"page":     strconv.Itoa(page),
		"pageSize": strconv.Itoa(pageSize),
	}

	for {
		params["page"] = strconv.Itoa(page)

		filters, err := api.ListIPFilters(domainId, vhost.Label, params)
		if err != nil {
			return err
		}

		for _, f := range filters {
			r := terraformutils.NewResource(
				strconv.Itoa(f.ID),
				fmt.Sprintf("%s_%d", vhost.Label, f.ID),
				"myrasec_ip_filter",
				"myrasec",
				map[string]string{
					"subdomain_name": vhost.Label,
				},
				[]string{},
				map[string]interface{}{},
			)
			g.appendResource(r)
		}
		if len(filters) < pageSize {
			break
		}
		page++
	}
	return nil
}

// InitResources
func (g *IPFilterGenerator) InitResources() error {
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}

	funcs := []func(*mgo.API, int, mgo.VHost) error{
		g.createIPFilterResources,
	}

	err = createResourcesPerSubDomain(api, funcs, true)
	if err != nil {
		return err
	}

	return nil
}
