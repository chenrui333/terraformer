//nolint:revive // lint triage: legacy provider/API/security baseline is tracked in #175.
package myrasec

import (
	"fmt"
	"strconv"

	mgo "github.com/Myra-Security-GmbH/myrasec-go/v2"
	"github.com/chenrui333/terraformer/terraformutils"
)

// CacheSettingGenerator
type CacheSettingGenerator struct {
	MyrasecService
}

// createCacheSettingResources
func (g *CacheSettingGenerator) createCacheSettingResources(api *mgo.API, domainId int, vhost mgo.VHost) error {
	page := 1
	pageSize := 250
	params := map[string]string{
		"pageSize": strconv.Itoa(pageSize),
		"page":     strconv.Itoa(page),
	}

	for {
		params["page"] = strconv.Itoa(page)

		settings, err := api.ListCacheSettings(domainId, vhost.Label, params)

		if err != nil {
			return err
		}

		for _, s := range settings {
			r := terraformutils.NewResource(
				strconv.Itoa(s.ID),
				fmt.Sprintf("%s_%d", vhost.Label, s.ID),
				"myrasec_cache_setting",
				"myrasec",
				map[string]string{
					"subdomain_name": vhost.Label,
				},
				[]string{},
				map[string]interface{}{},
			)
			r.IgnoreKeys = append(r.IgnoreKeys, "^Metadata")
			g.appendResource(r)
		}
		if len(settings) < pageSize {
			break
		}
		page++
	}
	return nil
}

// InitResources
func (g *CacheSettingGenerator) InitResources() error {
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}

	funcs := []func(*mgo.API, int, mgo.VHost) error{
		g.createCacheSettingResources,
	}
	err = createResourcesPerSubDomain(api, funcs, true)
	if err != nil {
		return err
	}

	return nil
}
