// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	CustomAllocationRuleAllowEmptyValues = []string{}
)

type CustomAllocationRuleGenerator struct {
	DatadogService
}

func (g *CustomAllocationRuleGenerator) createResource(rule datadogV2.ArbitraryRuleResponseData) terraformutils.Resource {
	id := rule.GetId()
	resourceName := fmt.Sprintf("custom_allocation_rule_%s", id)
	attrs := rule.GetAttributes()
	if ruleName := attrs.GetRuleName(); ruleName != "" {
		resourceName = fmt.Sprintf("custom_allocation_rule_%s", ruleName)
	}

	return terraformutils.NewSimpleResource(
		id,
		resourceName,
		"datadog_custom_allocation_rule",
		"datadog",
		CustomAllocationRuleAllowEmptyValues,
	)
}

func (g *CustomAllocationRuleGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewCloudCostManagementApi(datadogClient)

	resp, httpResp, err := api.ListCustomAllocationRules(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}

	resources := []terraformutils.Resource{}
	for _, rule := range resp.GetData() {
		if rule.GetId() == "" {
			continue
		}
		resources = append(resources, g.createResource(rule))
	}
	g.Resources = resources
	return nil
}
