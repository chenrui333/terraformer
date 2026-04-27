// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"encoding/json"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
)

type PolicyGenerator struct {
	RBTService
}

type Policy struct {
	Name  string `json:"name"`
	Vhost string `json:"vhost"`
}

type Policies []Policy

var PolicyAllowEmptyValues = []string{}
var PolicyAdditionalFields = map[string]interface{}{}

func (g PolicyGenerator) createResources(policies Policies) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, policy := range policies {
		resources = append(resources, terraformutils.NewResource(
			fmt.Sprintf("%s@%s", policy.Name, policy.Vhost),
			fmt.Sprintf("policy_%s_%s", normalizeResourceName(policy.Vhost), normalizeResourceName(policy.Name)),
			"rabbitmq_policy",
			"rabbitmq",
			map[string]string{
				"name":  policy.Name,
				"vhost": policy.Vhost,
			},
			PolicyAllowEmptyValues,
			PolicyAdditionalFields,
		))
	}
	return resources
}

func (g *PolicyGenerator) InitResources() error {
	body, err := g.generateRequest("/api/policies?columns=name,vhost")
	if err != nil {
		return err
	}
	var policies Policies
	err = json.Unmarshal(body, &policies)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(policies)
	return nil
}
