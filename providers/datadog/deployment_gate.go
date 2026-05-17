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

const (
	datadogDeploymentGateServiceName = "deployment_gate"
	datadogDeploymentGatePageSize    = int64(100)
)

var (
	// DeploymentGateAllowEmptyValues ...
	DeploymentGateAllowEmptyValues = []string{}
)

// DeploymentGateGenerator ...
type DeploymentGateGenerator struct {
	DatadogService
}

func (g *DeploymentGateGenerator) createResource(gateID string, rules []datadogV2.DeploymentRuleResponseDataAttributes) (terraformutils.Resource, error) {
	if gateID == "" {
		return terraformutils.Resource{}, fmt.Errorf("%s missing id", datadogDeploymentGateServiceName)
	}

	attributes := map[string]string{
		"id": gateID,
	}
	if err := addDeploymentGateRuleAttributes(attributes, rules); err != nil {
		return terraformutils.Resource{}, err
	}

	return terraformutils.NewResource(
		gateID,
		fmt.Sprintf("%s_%s", datadogDeploymentGateServiceName, gateID),
		"datadog_deployment_gate",
		"datadog",
		attributes,
		DeploymentGateAllowEmptyValues,
		map[string]interface{}{},
	), nil
}

func (g *DeploymentGateGenerator) createResources(auth context.Context, api *datadogV2.DeploymentGatesApi, gateIDs []string) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, gateID := range gateIDs {
		rules, err := g.getDeploymentGateRules(auth, api, gateID)
		if err != nil {
			return nil, err
		}
		resource, err := g.createResource(gateID, rules)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

func addDeploymentGateRuleAttributes(attributes map[string]string, rules []datadogV2.DeploymentRuleResponseDataAttributes) error {
	if len(rules) == 0 {
		return nil
	}

	attributes["rule.#"] = strconv.Itoa(len(rules))
	for ruleIndex, rule := range rules {
		prefix := fmt.Sprintf("rule.%d", ruleIndex)
		ruleID, ok := deploymentGateRuleID(rule)
		if !ok {
			return fmt.Errorf("deployment gate rule missing id")
		}
		name, ok := rule.GetNameOk()
		if !ok || name == nil || *name == "" {
			return fmt.Errorf("deployment gate rule %s missing required name", ruleID)
		}
		ruleType, ok := rule.GetTypeOk()
		if !ok || ruleType == nil || string(*ruleType) == "" {
			return fmt.Errorf("deployment gate rule %s missing required type", ruleID)
		}

		attributes[prefix+".id"] = ruleID
		attributes[prefix+".name"] = *name
		attributes[prefix+".type"] = string(*ruleType)
		if dryRun, ok := rule.GetDryRunOk(); ok && dryRun != nil {
			attributes[prefix+".dry_run"] = strconv.FormatBool(*dryRun)
		}
		if err := addDeploymentGateRuleOptionsAttributes(attributes, prefix, string(*ruleType), rule.GetOptions()); err != nil {
			return err
		}
	}
	return nil
}

func deploymentGateRuleID(rule datadogV2.DeploymentRuleResponseDataAttributes) (string, bool) {
	ruleID, ok := rule.AdditionalProperties["id"].(string)
	return ruleID, ok && ruleID != ""
}

func addDeploymentGateRuleOptionsAttributes(attributes map[string]string, rulePrefix, ruleType string, options datadogV2.DeploymentRulesOptions) error {
	optionsPrefix := rulePrefix + ".options"
	hasOptions := false

	switch ruleType {
	case "faulty_deployment_detection":
		if fddOptions := options.DeploymentRuleOptionsFaultyDeploymentDetection; fddOptions != nil {
			hasOptions = addDeploymentGateFDDOptionsAttributes(attributes, optionsPrefix, fddOptions)
		} else if unparsedOptions, ok := options.UnparsedObject.(map[string]interface{}); ok {
			hasOptions = addDeploymentGateFDDUnparsedOptionsAttributes(attributes, optionsPrefix, unparsedOptions)
		}
	case "monitor":
		if monitorOptions := options.DeploymentRuleOptionsMonitor; monitorOptions != nil {
			hasOptions = addDeploymentGateMonitorOptionsAttributes(attributes, optionsPrefix, monitorOptions)
		} else if unparsedOptions, ok := options.UnparsedObject.(map[string]interface{}); ok {
			hasOptions = addDeploymentGateMonitorUnparsedOptionsAttributes(attributes, optionsPrefix, unparsedOptions)
		}
	default:
		return fmt.Errorf("deployment gate rule has unsupported type %q", ruleType)
	}

	if !hasOptions {
		return fmt.Errorf("deployment gate rule %s missing reconstructable options", attributes[rulePrefix+".id"])
	}
	return nil
}

func addDeploymentGateFDDOptionsAttributes(attributes map[string]string, optionsPrefix string, options *datadogV2.DeploymentRuleOptionsFaultyDeploymentDetection) bool {
	hasOptions := false
	if duration, ok := options.GetDurationOk(); ok && duration != nil {
		attributes[optionsPrefix+".duration"] = strconv.FormatInt(*duration, 10)
		hasOptions = true
	}
	if excludedResources, ok := options.GetExcludedResourcesOk(); ok && excludedResources != nil && len(*excludedResources) > 0 {
		addDeploymentGateExcludedResourcesAttributes(attributes, optionsPrefix, *excludedResources)
		hasOptions = true
	}
	return hasOptions
}

func addDeploymentGateMonitorOptionsAttributes(attributes map[string]string, optionsPrefix string, options *datadogV2.DeploymentRuleOptionsMonitor) bool {
	hasOptions := false
	if duration, ok := options.GetDurationOk(); ok && duration != nil {
		attributes[optionsPrefix+".duration"] = strconv.FormatInt(*duration, 10)
		hasOptions = true
	}
	if query, ok := options.GetQueryOk(); ok && query != nil {
		attributes[optionsPrefix+".query"] = *query
		hasOptions = true
	}
	return hasOptions
}

func addDeploymentGateFDDUnparsedOptionsAttributes(attributes map[string]string, optionsPrefix string, options map[string]interface{}) bool {
	hasOptions := addDeploymentGateUnparsedDurationAttribute(attributes, optionsPrefix, options)
	if excludedResources, ok := deploymentGateUnparsedStringSlice(options["excluded_resources"]); ok && len(excludedResources) > 0 {
		addDeploymentGateExcludedResourcesAttributes(attributes, optionsPrefix, excludedResources)
		hasOptions = true
	}
	return hasOptions
}

func addDeploymentGateMonitorUnparsedOptionsAttributes(attributes map[string]string, optionsPrefix string, options map[string]interface{}) bool {
	hasOptions := addDeploymentGateUnparsedDurationAttribute(attributes, optionsPrefix, options)
	if query, ok := options["query"].(string); ok {
		attributes[optionsPrefix+".query"] = query
		hasOptions = true
	}
	return hasOptions
}

func addDeploymentGateUnparsedDurationAttribute(attributes map[string]string, optionsPrefix string, options map[string]interface{}) bool {
	switch duration := options["duration"].(type) {
	case int:
		attributes[optionsPrefix+".duration"] = strconv.Itoa(duration)
		return true
	case int64:
		attributes[optionsPrefix+".duration"] = strconv.FormatInt(duration, 10)
		return true
	case float64:
		attributes[optionsPrefix+".duration"] = strconv.FormatInt(int64(duration), 10)
		return true
	default:
		return false
	}
}

func deploymentGateUnparsedStringSlice(value interface{}) ([]string, bool) {
	switch typedValue := value.(type) {
	case []string:
		return typedValue, true
	case []interface{}:
		values := []string{}
		for _, item := range typedValue {
			itemValue, ok := item.(string)
			if !ok {
				return nil, false
			}
			values = append(values, itemValue)
		}
		return values, true
	default:
		return nil, false
	}
}

func addDeploymentGateExcludedResourcesAttributes(attributes map[string]string, optionsPrefix string, excludedResources []string) {
	attributes[optionsPrefix+".excluded_resources.#"] = strconv.Itoa(len(excludedResources))
	for excludedResourceIndex, excludedResource := range excludedResources {
		attributes[fmt.Sprintf("%s.excluded_resources.%d", optionsPrefix, excludedResourceIndex)] = excludedResource
	}
}

// InitResources Generate TerraformResources from Datadog API,
// from each deployment_gate create 1 TerraformResource.
func (g *DeploymentGateGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	datadogClient.GetConfig().SetUnstableOperationEnabled("v2.GetDeploymentGate", true)
	datadogClient.GetConfig().SetUnstableOperationEnabled("v2.GetDeploymentGateRules", true)
	datadogClient.GetConfig().SetUnstableOperationEnabled("v2.ListDeploymentGates", true)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewDeploymentGatesApi(datadogClient)

	resources, hasIDFilter, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if hasIDFilter {
		g.Resources = resources
		return nil
	}

	gateIDs, err := g.listDeploymentGateIDs(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(auth, api, gateIDs)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}

func (g *DeploymentGateGenerator) filteredResources(auth context.Context, api *datadogV2.DeploymentGatesApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	hasIDFilter := false
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || filter.ServiceName != datadogDeploymentGateServiceName {
			continue
		}
		hasIDFilter = true
		for _, value := range filter.AcceptableValues {
			gateID, err := g.getDeploymentGateID(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			rules, err := g.getDeploymentGateRules(auth, api, gateID)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(gateID, rules)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}
	return resources, hasIDFilter, nil
}

func (g *DeploymentGateGenerator) getDeploymentGateID(auth context.Context, api *datadogV2.DeploymentGatesApi, gateID string) (string, error) {
	resp, httpResp, err := api.GetDeploymentGate(auth, gateID)
	closeDatadogResponseBody(httpResp)
	if err != nil {
		return "", err
	}
	data := resp.GetData()
	responseID := data.GetId()
	if responseID == "" {
		return gateID, nil
	}
	return responseID, nil
}

func (g *DeploymentGateGenerator) getDeploymentGateRules(auth context.Context, api *datadogV2.DeploymentGatesApi, gateID string) ([]datadogV2.DeploymentRuleResponseDataAttributes, error) {
	resp, httpResp, err := api.GetDeploymentGateRules(auth, gateID)
	closeDatadogResponseBody(httpResp)
	if err != nil {
		return nil, err
	}
	data := resp.GetData()
	attributes := data.GetAttributes()
	rules, ok := attributes.GetRulesOk()
	if !ok || rules == nil {
		return nil, nil
	}
	return *rules, nil
}

func (g *DeploymentGateGenerator) listDeploymentGateIDs(auth context.Context, api *datadogV2.DeploymentGatesApi) ([]string, error) {
	ids := []string{}
	nextCursor := ""

	for {
		opts := datadogV2.NewListDeploymentGatesOptionalParameters().
			WithPageSize(datadogDeploymentGatePageSize)
		if nextCursor != "" {
			opts.WithPageCursor(nextCursor)
		}

		resp, httpResp, err := api.ListDeploymentGates(auth, *opts)
		closeDatadogResponseBody(httpResp)
		if err != nil {
			return nil, err
		}

		gates := resp.GetData()
		for _, gate := range gates {
			gateID := gate.GetId()
			if gateID == "" {
				continue
			}
			ids = append(ids, gateID)
		}

		meta := resp.GetMeta()
		page := meta.GetPage()
		nextCursor = page.GetNextCursor()
		if nextCursor == "" {
			break
		}
	}

	return ids, nil
}
