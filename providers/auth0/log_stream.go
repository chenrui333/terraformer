// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"

	"github.com/auth0/go-auth0/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	LogStreamAllowEmptyValues = []string{}
)

type LogStreamGenerator struct {
	Auth0Service
}

func (g LogStreamGenerator) createResources(logStreams []*management.LogStream) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, logStream := range logStreams {
		if logStream == nil {
			return nil, auth0MissingResource("auth0_log_stream")
		}
		resourceName, err := auth0RequiredString("auth0_log_stream", "id", logStream.ID)
		if err != nil {
			return nil, err
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			auth0ResourceName(logStream.Name, resourceName),
			"auth0_log_stream",
			"auth0",
			LogStreamAllowEmptyValues,
		))
	}
	return resources, nil
}

func (g *LogStreamGenerator) InitResources() error {
	m, err := g.generateClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	list, err := m.LogStream.List(ctx)
	if err != nil {
		return err
	}

	resources, err := g.createResources(list)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}
