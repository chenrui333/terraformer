// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"

	"github.com/auth0/go-auth0/v2/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	LogStreamAllowEmptyValues = []string{}
)

type LogStreamGenerator struct {
	Auth0Service
}

func (g LogStreamGenerator) createResources(logStreams []*management.LogStreamResponseSchema) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, logStream := range logStreams {
		if logStream == nil {
			return nil, auth0MissingResource("auth0_log_stream")
		}
		id, name := auth0LogStreamResourceValues(logStream)
		resourceName, err := auth0RequiredString("auth0_log_stream", "id", id)
		if err != nil {
			return nil, err
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			auth0ResourceName(name, resourceName),
			"auth0_log_stream",
			"auth0",
			LogStreamAllowEmptyValues,
		))
	}
	return resources, nil
}

func auth0LogStreamResourceValues(logStream *management.LogStreamResponseSchema) (*string, *string) {
	if logStream == nil {
		return nil, nil
	}
	if v := logStream.GetLogStreamHTTPResponseSchema(); v != nil {
		return v.ID, v.Name
	}
	if v := logStream.GetLogStreamEventBridgeResponseSchema(); v != nil {
		return v.ID, v.Name
	}
	if v := logStream.GetLogStreamEventGridResponseSchema(); v != nil {
		return v.ID, v.Name
	}
	if v := logStream.GetLogStreamDatadogResponseSchema(); v != nil {
		return v.ID, v.Name
	}
	if v := logStream.GetLogStreamSplunkResponseSchema(); v != nil {
		return v.ID, v.Name
	}
	if v := logStream.GetLogStreamSumoResponseSchema(); v != nil {
		return v.ID, v.Name
	}
	if v := logStream.GetLogStreamSegmentResponseSchema(); v != nil {
		return v.ID, v.Name
	}
	if v := logStream.GetLogStreamMixpanelResponseSchema(); v != nil {
		return v.ID, v.Name
	}
	return nil, nil
}

func (g *LogStreamGenerator) InitResources() error {
	m, err := g.generateClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	list, err := m.LogStreams.List(ctx)
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
