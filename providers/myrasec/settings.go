//nolint:revive // lint triage: legacy provider/API/security baseline is tracked in #175.
package myrasec

import (
	"fmt"
	"strconv"

	mgo "github.com/Myra-Security-GmbH/myrasec-go/v2"
	"github.com/chenrui333/terraformer/terraformutils"
)

// SettingGenerator
type SettingsGenerator struct {
	MyrasecService
}

// createSettingResources
func (g *SettingsGenerator) createSettingResources(api *mgo.API, domainId int, vhost mgo.VHost) error {
	params := map[string]string{}

	s, err := api.ListSettings(domainId, vhost.Label, params)
	if err != nil {
		return err
	}

	r := terraformutils.NewResource(
		strconv.Itoa(vhost.ID),
		fmt.Sprintf("%s_%d", vhost.Label, vhost.ID),
		"myrasec_settings",
		"myrasec",
		map[string]string{
			"subdomain_name": vhost.Label,
			"only_https":     strconv.FormatBool(s.OnlyHTTPS),
		},
		[]string{},
		map[string]interface{}{},
	)
	g.appendResource(r)
	return nil
}

// InitResources
func (g *SettingsGenerator) InitResources() error {
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}

	funcs := []func(*mgo.API, int, mgo.VHost) error{
		g.createSettingResources,
	}

	err = createResourcesPerSubDomain(api, funcs, true)
	if err != nil {
		return err
	}

	return nil
}
