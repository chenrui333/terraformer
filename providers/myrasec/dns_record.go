//nolint:revive // lint triage: legacy provider/API/security baseline is tracked in #175.
package myrasec

import (
	"fmt"
	"strconv"

	mgo "github.com/Myra-Security-GmbH/myrasec-go/v2"
	"github.com/chenrui333/terraformer/terraformutils"
)

// DNSGenerator
type DNSGenerator struct {
	MyrasecService
}

// createDnsResources
func (g *DNSGenerator) createDnsResources(api *mgo.API, domain mgo.Domain) error {
	page := 1
	pageSize := 250
	params := map[string]string{
		"pageSize": strconv.Itoa(pageSize),
		"page":     strconv.Itoa(page),
	}

	for {
		params["page"] = strconv.Itoa(page)

		records, err := api.ListDNSRecords(domain.ID, params)
		if err != nil {
			return err
		}

		for _, d := range records {
			r := terraformutils.NewResource(
				strconv.Itoa(d.ID),
				fmt.Sprintf("%s_%d", domain.Name, d.ID),
				"myrasec_dns_record",
				"myrasec",
				map[string]string{
					"domain_name": domain.Name,
				},
				[]string{},
				map[string]interface{}{},
			)

			r.IgnoreKeys = append(r.IgnoreKeys, "^metadata")
			g.appendResource(r)
		}
		if len(records) < pageSize {
			break
		}
		page++
	}

	return nil
}

// InitResources
func (g *DNSGenerator) InitResources() error {
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}

	funcs := []func(*mgo.API, mgo.Domain) error{
		g.createDnsResources,
	}

	err = createResourcesPerDomain(api, funcs)
	if err != nil {
		return err
	}

	return nil
}
