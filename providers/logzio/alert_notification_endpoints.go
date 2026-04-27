// SPDX-License-Identifier: Apache-2.0

package logzio

import (
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/logzio/logzio_terraform_client/endpoints"
)

type AlertNotificationEndpointsGenerator struct {
	LogzioService
}

// Generate Terraform Resources from Logzio API,
func (g *AlertNotificationEndpointsGenerator) InitResources() error {
	var client *endpoints.EndpointsClient
	client, _ = endpoints.New(g.Args["api_token"].(string), g.Args["base_url"].(string))

	endpoints, err := client.ListEndpoints()
	if err != nil {
		return err
	}
	for _, endpoint := range endpoints {
		endpointID := strconv.FormatInt(int64(endpoint.Id), 10)
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			endpointID,
			createSlug(endpoint.Title+"-"+endpoint.Type+"-"+endpointID),
			"logzio_endpoint",
			"logzio",
			[]string{},
		))
	}
	return nil
}
