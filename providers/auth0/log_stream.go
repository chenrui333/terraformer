// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"gopkg.in/auth0.v5/management"
)

var (
	LogStreamAllowEmptyValues = []string{}
)

type LogStreamGenerator struct {
	Auth0Service
}

func (g LogStreamGenerator) createResources(logStreams []*management.LogStream) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, LogStream := range logStreams {
		resourceName := *LogStream.ID
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			resourceName+"_"+*LogStream.Name,
			"auth0_log_stream",
			"auth0",
			LogStreamAllowEmptyValues,
		))
	}
	return resources
}

func (g *LogStreamGenerator) InitResources() error {
	m := g.generateClient()
	list, err := m.LogStream.List()
	if err != nil {
		return err
	}

	g.Resources = g.createResources(list)
	return nil
}
