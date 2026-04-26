// Copyright 2018 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
