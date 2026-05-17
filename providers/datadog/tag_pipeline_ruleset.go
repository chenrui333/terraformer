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
	TagPipelineRulesetAllowEmptyValues = []string{}
)

type TagPipelineRulesetGenerator struct {
	DatadogService
}

func (g *TagPipelineRulesetGenerator) createResource(ruleset datadogV2.RulesetRespData) terraformutils.Resource {
	id := ruleset.GetId()
	resourceName := fmt.Sprintf("tag_pipeline_ruleset_%s", id)
	attrs := ruleset.GetAttributes()
	if name := attrs.GetName(); name != "" {
		resourceName = fmt.Sprintf("tag_pipeline_ruleset_%s", name)
	}

	return terraformutils.NewSimpleResource(
		id,
		resourceName,
		"datadog_tag_pipeline_ruleset",
		"datadog",
		TagPipelineRulesetAllowEmptyValues,
	)
}

func (g *TagPipelineRulesetGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewCloudCostManagementApi(datadogClient)

	resp, httpResp, err := api.ListTagPipelinesRulesets(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}

	resources := []terraformutils.Resource{}
	for _, ruleset := range resp.GetData() {
		resources = append(resources, g.createResource(ruleset))
	}
	g.Resources = resources
	return nil
}
