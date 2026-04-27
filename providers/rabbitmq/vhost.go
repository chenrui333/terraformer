// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"encoding/json"

	"github.com/chenrui333/terraformer/terraformutils"
)

type VhostGenerator struct {
	RBTService
}

type Vhost struct {
	Name string `json:"name"`
}

type Vhosts []Vhost

var VhostAllowEmptyValues = []string{}

func (g VhostGenerator) createResources(vhosts Vhosts) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, vhost := range vhosts {
		resources = append(resources, terraformutils.NewSimpleResource(
			vhost.Name,
			"vhost_"+normalizeResourceName(vhost.Name),
			"rabbitmq_vhost",
			"rabbitmq",
			VhostAllowEmptyValues,
		))
	}
	return resources
}

func (g *VhostGenerator) InitResources() error {
	body, err := g.generateRequest("/api/vhosts?columns=name")
	if err != nil {
		return err
	}
	var vhosts Vhosts
	err = json.Unmarshal(body, &vhosts)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(vhosts)
	return nil
}
