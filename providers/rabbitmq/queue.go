// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"encoding/json"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
)

type QueueGenerator struct {
	RBTService
}

type Queue struct {
	Name  string `json:"name"`
	Vhost string `json:"vhost"`
}

type Queues []Queue

var QueueAllowEmptyValues = []string{}
var QueueAdditionalFields = map[string]interface{}{}

func (g QueueGenerator) createResources(queues Queues) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, queue := range queues {
		resources = append(resources, terraformutils.NewResource(
			fmt.Sprintf("%s@%s", queue.Name, queue.Vhost),
			fmt.Sprintf("queue_%s_%s", normalizeResourceName(queue.Vhost), normalizeResourceName(queue.Name)),
			"rabbitmq_queue",
			"rabbitmq",
			map[string]string{
				"name":  queue.Name,
				"vhost": queue.Vhost,
			},
			QueueAllowEmptyValues,
			QueueAdditionalFields,
		))
	}
	return resources
}

func (g *QueueGenerator) InitResources() error {
	body, err := g.generateRequest("/api/queues?columns=name,vhost")
	if err != nil {
		return err
	}
	var queues Queues
	err = json.Unmarshal(body, &queues)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(queues)
	return nil
}
