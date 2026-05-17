// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

// CSMThreatsAgentRuleGenerator ...
type CSMThreatsAgentRuleGenerator struct {
	DatadogService
}

func (g *CSMThreatsAgentRuleGenerator) createResource(ruleID, policyID string) (terraformutils.Resource, error) {
	if ruleID == "" {
		return terraformutils.Resource{}, fmt.Errorf("CSM threats agent rule missing id")
	}

	attrs := map[string]string{}
	resourceName := fmt.Sprintf("csm_threats_agent_rule_%s", ruleID)
	if policyID != "" {
		attrs["policy_id"] = policyID
		resourceName = fmt.Sprintf("csm_threats_agent_rule_%s_%s", policyID, ruleID)
	}

	return terraformutils.NewResource(
		ruleID,
		resourceName,
		"datadog_csm_threats_agent_rule",
		"datadog",
		attrs,
		[]string{},
		map[string]interface{}{},
	), nil
}

// InitResources Generate TerraformResources from Datadog API.
func (g *CSMThreatsAgentRuleGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewCSMThreatsApi(datadogClient)

	for filterIndex, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("csm_threats_agent_rule") {
			var resources []terraformutils.Resource
			var ruleIDs []string
			for _, value := range filter.AcceptableValues {
				policyID, ruleID := parseCSMThreatsAgentRuleFilterID(value)
				resource, err := g.createResource(ruleID, policyID)
				if err != nil {
					return err
				}
				resources = append(resources, resource)
				ruleIDs = append(ruleIDs, ruleID)
			}
			g.Filter[filterIndex].AcceptableValues = ruleIDs
			g.Resources = resources
			return nil
		}
	}

	// List all rules (includes unscoped rules not tied to any policy)
	rulesResp, httpResp, err := api.ListCSMThreatsAgentRules(auth)
	if httpResp != nil && httpResp.Body != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		return err
	}

	var resources []terraformutils.Resource
	for _, rule := range rulesResp.GetData() {
		ruleID := rule.GetId()
		if ruleID == "" {
			continue
		}
		policies := csmThreatsAgentRulePolicies(rule)
		if len(policies) == 0 {
			resource, err := g.createResource(ruleID, "")
			if err != nil {
				return err
			}
			resources = append(resources, resource)
		} else {
			for _, policyID := range policies {
				resource, err := g.createResource(ruleID, policyID)
				if err != nil {
					return err
				}
				resources = append(resources, resource)
			}
		}
	}

	g.Resources = resources
	return nil
}

func csmThreatsAgentRulePolicies(rule datadogV2.CloudWorkloadSecurityAgentRuleData) []string {
	attrs := rule.GetAttributes()
	seen := map[string]struct{}{}
	var policies []string
	for _, id := range attrs.GetBlocking() {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			policies = append(policies, id)
		}
	}
	for _, id := range attrs.GetMonitoring() {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			policies = append(policies, id)
		}
	}
	for _, id := range attrs.GetDisabled() {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			policies = append(policies, id)
		}
	}
	return policies
}

func parseCSMThreatsAgentRuleFilterID(value string) (policyID, ruleID string) {
	if idx := strings.IndexByte(value, ':'); idx >= 0 {
		return value[:idx], value[idx+1:]
	}
	return "", value
}
