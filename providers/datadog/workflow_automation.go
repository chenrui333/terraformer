// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"log"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// WorkflowAutomationAllowEmptyValues ...
	WorkflowAutomationAllowEmptyValues = []string{"description"}
)

// WorkflowAutomationGenerator ...
type WorkflowAutomationGenerator struct {
	DatadogService
}

func (g *WorkflowAutomationGenerator) PostConvertHook() error {
	for i := range g.Resources {
		resource := &g.Resources[i]
		if resource.Item == nil {
			resource.Item = map[string]interface{}{}
		}
		if _, ok := resource.Item["tags"]; ok {
			continue
		}
		if !workflowAutomationStateHasEmptyTags(resource) {
			continue
		}
		resource.Item["tags"] = []interface{}{}
	}
	return nil
}

func workflowAutomationStateHasEmptyTags(resource *terraformutils.Resource) bool {
	if resource == nil || resource.InstanceState == nil || resource.InstanceState.Attributes == nil {
		return false
	}
	return resource.InstanceState.Attributes["tags.#"] == "0"
}

func (g *WorkflowAutomationGenerator) createResource(workflow datadogV2.WorkflowData) (terraformutils.Resource, error) {
	workflowID := workflow.GetId()
	if workflowID == "" {
		return terraformutils.Resource{}, fmt.Errorf("workflow automation missing id")
	}

	return terraformutils.NewSimpleResource(
		workflowID,
		fmt.Sprintf("workflow_automation_%s", workflowID),
		"datadog_workflow_automation",
		"datadog",
		WorkflowAutomationAllowEmptyValues,
	), nil
}

// InitResources Generate TerraformResources from Datadog API.
// The workflow automation API supports get-by-ID but does not expose a workflow list endpoint,
// so this generator requires an ID filter containing the workflow ID.
func (g *WorkflowAutomationGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewWorkflowAutomationApi(datadogClient)

	resources := []terraformutils.Resource{}
	matchedIDFilter := false

	for _, filter := range g.Filter {
		if filter.FieldPath != "id" {
			continue
		}
		if !filter.IsApplicable("workflow_automation") {
			continue
		}
		matchedIDFilter = true
		for _, value := range filter.AcceptableValues {
			workflow, httpResp, err := api.GetWorkflow(auth, value)
			closeDatadogResponseBody(httpResp)
			if err != nil {
				return err
			}
			resource, err := g.createResource(workflow.GetData())
			if err != nil {
				return err
			}
			resources = append(resources, resource)
		}
	}

	if matchedIDFilter {
		g.Resources = resources
		return nil
	}

	log.Print("Filter(resource id) is required to import datadog_workflow_automation resources because the Datadog API does not provide a workflow list endpoint")
	return nil
}
