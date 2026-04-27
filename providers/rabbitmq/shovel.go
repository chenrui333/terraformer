// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"encoding/json"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
)

type ShovelGenerator struct {
	RBTService
}

type Shovel struct {
	Name  string `json:"name"`
	Vhost string `json:"vhost"`
}

type Shovels []Shovel

var ShovelAllowEmptyValues = []string{}
var ShovelAdditionalFields = map[string]interface{}{}

func (g ShovelGenerator) createResources(shovels Shovels) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, shovel := range shovels {
		if len(shovel.Name) == 0 {
			continue
		}
		resources = append(resources, terraformutils.NewResource(
			fmt.Sprintf("%s@%s", shovel.Name, shovel.Vhost),
			fmt.Sprintf("shovel_%s_%s", normalizeResourceName(shovel.Vhost), normalizeResourceName(shovel.Name)),
			"rabbitmq_shovel",
			"rabbitmq",
			map[string]string{
				"name":  shovel.Name,
				"vhost": shovel.Vhost,
			},
			ShovelAllowEmptyValues,
			ShovelAdditionalFields,
		))
	}
	return resources
}

func (g *ShovelGenerator) InitResources() error {
	body, err := g.generateRequest("/api/shovels?columns=name,vhost")
	if err != nil {
		return err
	}
	var shovels Shovels
	err = json.Unmarshal(body, &shovels)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(shovels)
	return nil
}
