// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"encoding/json"
	"fmt"
	"net/http"

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
	integration, err := getPagerDutyIntegration(
		g.Args["api-key"].(string),
		g.Args["app-key"].(string),
		g.Args["api-url"].(string),
	)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(integration.Subdomain)
	return nil
}

type pagerDutyIntegration struct {
	Subdomain string               `json:"subdomain"`
	Services  []pagerDutyServicePD `json:"services"`
}

type pagerDutyServicePD struct {
	ServiceName string `json:"service_name"`
	ServiceKey  string `json:"service_key"`
}

func getPagerDutyIntegration(apiKey, appKey, apiURL string) (*pagerDutyIntegration, error) {
	url := apiURL + "/api/v1/integration/pagerduty"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("DD-API-KEY", apiKey)
	req.Header.Set("DD-APPLICATION-KEY", appKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("datadog API returned %d for PagerDuty integration", resp.StatusCode)
	}

	var integration pagerDutyIntegration
	if err := json.NewDecoder(resp.Body).Decode(&integration); err != nil {
		return nil, err
	}
	return &integration, nil
}
