// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"fmt"

	datadogCommunity "github.com/zorkian/go-datadog-api"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// IntegrationPagerdutyAllowEmptyValues ...
	IntegrationPagerdutyAllowEmptyValues = []string{"tags."}
)

// IntegrationPagerdutyGenerator ...
type IntegrationPagerdutyGenerator struct {
	DatadogService
}

func (g *IntegrationPagerdutyGenerator) createResources(pdSubdomain string) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	resources = append(resources, g.createResource(pdSubdomain))

	return resources
}

func (g *IntegrationPagerdutyGenerator) createResource(serviceName string) terraformutils.Resource {
	resource := terraformutils.NewResource(
		serviceName,
		fmt.Sprintf("integration_pagerduty_%s", serviceName),
		"datadog_integration_pagerduty",
		"datadog",
		map[string]string{
			"individual_services": "true",
		},
		IntegrationPagerdutyAllowEmptyValues,
		map[string]interface{}{},
	)
	// Ignore services in favor of individual_services
	resource.IgnoreKeys = append(resource.IgnoreKeys, "^services$")

	return resource
}

// InitResources Generate TerraformResources from Datadog API,
// from PD Service create 1 TerraformResource.
// Need IntegrationPagerduty Subdomain as ID for terraform resource
func (g *IntegrationPagerdutyGenerator) InitResources() error {
	client := datadogCommunity.NewClient(g.Args["api-key"].(string), g.Args["app-key"].(string))

	integration, err := client.GetIntegrationPD()
	if err != nil {
		return err
	}
	g.Resources = g.createResources(integration.GetSubdomain())
	return nil
}
