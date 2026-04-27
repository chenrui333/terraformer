// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"encoding/json"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
)

type ExchangeGenerator struct {
	RBTService
}

type Exchange struct {
	Name  string `json:"name"`
	Vhost string `json:"vhost"`
}

type Exchanges []Exchange

var ExchangeAllowEmptyValues = []string{}
var ExchangeAdditionalFields = map[string]interface{}{}

func (g ExchangeGenerator) createResources(exchanges Exchanges) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, exchange := range exchanges {
		if len(exchange.Name) == 0 {
			continue
		}
		resources = append(resources, terraformutils.NewResource(
			fmt.Sprintf("%s@%s", exchange.Name, exchange.Vhost),
			fmt.Sprintf("exchange_%s_%s", normalizeResourceName(exchange.Vhost), normalizeResourceName(exchange.Name)),
			"rabbitmq_exchange",
			"rabbitmq",
			map[string]string{
				"name":  exchange.Name,
				"vhost": exchange.Vhost,
			},
			ExchangeAllowEmptyValues,
			ExchangeAdditionalFields,
		))
	}
	return resources
}

func (g *ExchangeGenerator) InitResources() error {
	body, err := g.generateRequest("/api/exchanges?columns=name,vhost")
	if err != nil {
		return err
	}
	var exchanges Exchanges
	err = json.Unmarshal(body, &exchanges)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(exchanges)
	return nil
}
