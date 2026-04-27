// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"fmt"

	datadogCommunity "github.com/zorkian/go-datadog-api"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// IntegrationPagerdutyServiceObjectAllowEmptyValues ...
	IntegrationPagerdutyServiceObjectAllowEmptyValues = []string{"tags."}
)

// IntegrationPagerdutyServiceObjectGenerator ...
type IntegrationPagerdutyServiceObjectGenerator struct {
	DatadogService
}

func (g *IntegrationPagerdutyServiceObjectGenerator) createResources(serviceNames []string) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, name := range serviceNames {
		resourceName := name
		resources = append(resources, g.createResource(resourceName))
	}

	return resources
}

func (g *IntegrationPagerdutyServiceObjectGenerator) createResource(serviceName string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		serviceName,
		fmt.Sprintf("integration_pagerduty_service_object_%s", serviceName),
		"datadog_integration_pagerduty_service_object",
		"datadog",
		IntegrationPagerdutyServiceObjectAllowEmptyValues,
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each PD Service create 1 TerraformResource.
// Need IntegrationPagerdutyServiceObject ServiceName as ID for terraform resource
func (g *IntegrationPagerdutyServiceObjectGenerator) InitResources() error {
	client := datadogCommunity.NewClient(g.Args["api-key"].(string), g.Args["app-key"].(string))

	pdIntegration, err := client.GetIntegrationPD()
	if err != nil {
		return err
	}

	var serviceNames []string
	for _, service := range pdIntegration.Services {
		serviceNames = append(serviceNames, *service.ServiceName)
	}

	g.Resources = g.createResources(serviceNames)
	return nil
}
