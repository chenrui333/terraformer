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
	// SecurityMonitoringRuleAllowEmptyValues ...
	SecurityMonitoringRuleAllowEmptyValues = []string{"tags."}
)

// SecurityMonitoringRuleGenerator ...
type SecurityMonitoringRuleGenerator struct {
	DatadogService
}

func (g *SecurityMonitoringRuleGenerator) createResources(rulesResponse []datadogV2.SecurityMonitoringRuleResponse) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, rule := range rulesResponse {
		if rule.SecurityMonitoringSignalRuleResponse != nil {
			if !rule.SecurityMonitoringSignalRuleResponse.GetIsDefault() {
				resourceName := rule.SecurityMonitoringSignalRuleResponse.GetId()
				resources = append(resources, g.createResource(resourceName, rule.SecurityMonitoringSignalRuleResponse.GetIsEnabled()))
			}
		}
		if rule.SecurityMonitoringStandardRuleResponse != nil {
			if !rule.SecurityMonitoringStandardRuleResponse.GetIsDefault() && rule.SecurityMonitoringStandardRuleResponse.GetType() != datadogV2.SECURITYMONITORINGRULETYPEREAD_CLOUD_CONFIGURATION {
				resourceName := rule.SecurityMonitoringStandardRuleResponse.GetId()
				resources = append(resources, g.createResource(resourceName, rule.SecurityMonitoringStandardRuleResponse.GetIsEnabled()))
			}
		}
	}

	return resources
}

func (g *SecurityMonitoringRuleGenerator) createResource(ruleID string, ruleEnabled bool) terraformutils.Resource {
	return terraformutils.NewResource(
		ruleID,
		fmt.Sprintf("security_monitoring_rule_%s", ruleID),
		"datadog_security_monitoring_rule",
		"datadog",
		map[string]string{
			"enabled": strconv.FormatBool(ruleEnabled),
		},
		SecurityMonitoringRuleAllowEmptyValues,
		map[string]interface{}{},
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each SecurityMonitoringRule create 1 TerraformResource.
// Need SecurityMonitoringRule ID as ID for terraform resource
func (g *SecurityMonitoringRuleGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewSecurityMonitoringApi(datadogClient)

	securityMonitoringRuleResponses, err := listSecurityMonitoringRules(auth, api)
	if err != nil {
		return err
	}

	g.Resources = g.createResources(securityMonitoringRuleResponses)
	return nil
}
