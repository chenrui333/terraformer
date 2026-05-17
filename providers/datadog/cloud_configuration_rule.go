// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"strconv"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// CloudConfigurationRuleAllowEmptyValues ...
	CloudConfigurationRuleAllowEmptyValues = []string{"filter.*.query", "tags."}
)

// CloudConfigurationRuleGenerator ...
type CloudConfigurationRuleGenerator struct {
	DatadogService
}

func (g *CloudConfigurationRuleGenerator) createResources(rules []datadogV2.SecurityMonitoringRuleResponse) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, rule := range rules {
		standardRule := rule.SecurityMonitoringStandardRuleResponse
		if standardRule == nil || standardRule.GetIsDefault() || standardRule.GetType() != datadogV2.SECURITYMONITORINGRULETYPEREAD_CLOUD_CONFIGURATION {
			continue
		}
		resources = append(resources, g.createResource(standardRule.GetId(), standardRule.GetIsEnabled()))
	}

	return resources
}

func (g *CloudConfigurationRuleGenerator) createResource(ruleID string, ruleEnabled bool) terraformutils.Resource {
	return terraformutils.NewResource(
		ruleID,
		fmt.Sprintf("cloud_configuration_rule_%s", ruleID),
		"datadog_cloud_configuration_rule",
		"datadog",
		map[string]string{
			"enabled": strconv.FormatBool(ruleEnabled),
		},
		CloudConfigurationRuleAllowEmptyValues,
		map[string]interface{}{},
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each cloud_configuration_rule create 1 TerraformResource.
// Need Cloud Configuration Rule ID as ID for terraform resource.
func (g *CloudConfigurationRuleGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewSecurityMonitoringApi(datadogClient)

	rules, err := listSecurityMonitoringRules(auth, api)
	if err != nil {
		return err
	}

	g.Resources = g.createResources(rules)
	return nil
}
