// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"encoding/json"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
)

type BindingGenerator struct {
	RBTService
}

type Binding struct {
	Source          string                 `json:"source"`
	Vhost           string                 `json:"vhost"`
	Destination     string                 `json:"destination"`
	DestinationType string                 `json:"destination_type"`
	PropertiesKey   string                 `json:"properties_key"`
	Arguments       map[string]interface{} `json:"arguments"`
}

type Bindings []Binding

var BindingAllowEmptyValues = []string{"source"}
var BindingAdditionalFields = map[string]interface{}{}

func (g BindingGenerator) createResources(bindings Bindings) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, binding := range bindings {
		argumentsJSON, errArgumentsJSON := json.Marshal(binding.Arguments)
		if errArgumentsJSON != nil {
			argumentsJSON = []byte("{}")
		}
		resources = append(resources, terraformutils.NewResource(
			fmt.Sprintf("%s/%s/%s/%s/%s", percentEncodeSlashes(binding.Vhost), binding.Source, binding.Destination, binding.DestinationType, binding.PropertiesKey),
			fmt.Sprintf("binding_%s_%s_%s_%s_%s", normalizeResourceName(binding.Source), normalizeResourceName(binding.Vhost), normalizeResourceName(binding.Destination), normalizeResourceName(binding.DestinationType), normalizeResourceName(binding.PropertiesKey)),
			"rabbitmq_binding",
			"rabbitmq",
			map[string]string{
				"source":           binding.Source,
				"vhost":            binding.Vhost,
				"destination":      binding.Destination,
				"destination_type": binding.DestinationType,
				"properties_key":   binding.PropertiesKey,
				"arguments_json":   string(argumentsJSON),
			},
			BindingAllowEmptyValues,
			BindingAdditionalFields,
		))
	}
	return resources
}

func (g *BindingGenerator) InitResources() error {
	body, err := g.generateRequest("/api/bindings?columns=source,vhost,destination,destination_type,properties_key,arguments")
	if err != nil {
		return err
	}
	var bindings Bindings
	err = json.Unmarshal(body, &bindings)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(bindings)
	return nil
}
