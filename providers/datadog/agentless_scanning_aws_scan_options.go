// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

// AgentlessScanningAwsScanOptionsGenerator ...
type AgentlessScanningAwsScanOptionsGenerator struct {
	DatadogService
}

func (g *AgentlessScanningAwsScanOptionsGenerator) createResource(data datadogV2.AwsScanOptionsData) (terraformutils.Resource, error) {
	id := data.GetId()
	if id == "" {
		return terraformutils.Resource{}, fmt.Errorf("agentless scanning AWS scan options missing id")
	}

	return terraformutils.NewResource(
		id,
		fmt.Sprintf("agentless_scanning_aws_scan_options_%s", id),
		"datadog_agentless_scanning_aws_scan_options",
		"datadog",
		map[string]string{
			"aws_account_id": id,
		},
		[]string{"lambda", "sensitive_data", "vuln_containers_os", "vuln_host_os"},
		map[string]interface{}{},
	), nil
}

func (g *AgentlessScanningAwsScanOptionsGenerator) createResources(items []datadogV2.AwsScanOptionsData) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	for _, item := range items {
		resource, err := g.createResource(item)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// InitResources Generate TerraformResources from Datadog API.
func (g *AgentlessScanningAwsScanOptionsGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewAgentlessScanningApi(datadogClient)

	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("agentless_scanning_aws_scan_options") {
			var resources []terraformutils.Resource
			for _, value := range filter.AcceptableValues {
				resp, httpResp, err := api.GetAwsScanOptions(auth, value)
				if httpResp != nil && httpResp.Body != nil {
					httpResp.Body.Close()
				}
				if err != nil {
					return err
				}
				resource, err := g.createResource(resp.GetData())
				if err != nil {
					return err
				}
				resources = append(resources, resource)
			}
			g.Resources = resources
			return nil
		}
	}

	resp, httpResp, err := api.ListAwsScanOptions(auth)
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		return err
	}

	resources, err := g.createResources(resp.GetData())
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}
