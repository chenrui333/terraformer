// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

const datadogSyntheticsSuitePageSize = int64(100)

const (
	syntheticsSuiteMessageKey = "message"
	syntheticsSuiteTagsKey    = "tags"
)

var (
	// SyntheticsSuiteAllowEmptyValues ...
	SyntheticsSuiteAllowEmptyValues = []string{"message", "tags."}
)

// SyntheticsSuiteGenerator ...
type SyntheticsSuiteGenerator struct {
	DatadogService
}

func (g *SyntheticsSuiteGenerator) createResource(suiteID string) (terraformutils.Resource, error) {
	if suiteID == "" {
		return terraformutils.Resource{}, fmt.Errorf("synthetics suite missing id")
	}

	return terraformutils.NewSimpleResource(
		suiteID,
		fmt.Sprintf("synthetics_suite_%s", suiteID),
		"datadog_synthetics_suite",
		"datadog",
		SyntheticsSuiteAllowEmptyValues,
	), nil
}

func (g *SyntheticsSuiteGenerator) PostConvertHook() error {
	for i := range g.Resources {
		resource := &g.Resources[i]
		if err := preserveSyntheticsSuiteEmptyTags(resource); err != nil {
			return err
		}
		if err := preserveSyntheticsSuiteEmptyMessage(resource); err != nil {
			return err
		}
	}
	return nil
}

func preserveSyntheticsSuiteEmptyMessage(resource *terraformutils.Resource) error {
	hasEmptyMessage, err := syntheticsSuiteStateHasEmptyMessage(resource)
	if err != nil {
		return err
	}
	if !hasEmptyMessage {
		return nil
	}
	if resource.Item == nil {
		resource.Item = map[string]interface{}{}
	}
	if value, ok := resource.Item[syntheticsSuiteMessageKey]; !ok || !syntheticsSuiteValueHasValue(value) {
		resource.Item[syntheticsSuiteMessageKey] = ""
	}
	return preserveSyntheticsSuiteEmptyMessageState(resource)
}

func syntheticsSuiteStateHasEmptyMessage(resource *terraformutils.Resource) (bool, error) {
	if resource == nil || resource.InstanceState == nil {
		return false, nil
	}
	if resource.InstanceState.Attributes != nil {
		if message, ok := resource.InstanceState.Attributes[syntheticsSuiteMessageKey]; ok && message == "" {
			return true, nil
		}
	}
	if len(resource.InstanceState.TypedAttributes) == 0 {
		return false, nil
	}

	typedAttributes := map[string]json.RawMessage{}
	if err := json.Unmarshal(resource.InstanceState.TypedAttributes, &typedAttributes); err != nil {
		return false, fmt.Errorf("decode synthetics suite typed attributes: %w", err)
	}
	rawMessage, ok := typedAttributes[syntheticsSuiteMessageKey]
	if !ok {
		return false, nil
	}
	messageIsEmpty, err := syntheticsSuiteRawMessageIsEmptyString(rawMessage)
	if err != nil {
		return false, fmt.Errorf("decode synthetics suite message state: %w", err)
	}
	return messageIsEmpty, nil
}

func preserveSyntheticsSuiteEmptyMessageState(resource *terraformutils.Resource) error {
	if resource == nil || resource.InstanceState == nil {
		return nil
	}
	if resource.InstanceState.Attributes == nil {
		resource.InstanceState.Attributes = map[string]string{}
	}
	resource.InstanceState.Attributes[syntheticsSuiteMessageKey] = ""
	if len(resource.InstanceState.TypedAttributes) == 0 {
		return nil
	}

	typedAttributes := map[string]json.RawMessage{}
	if err := json.Unmarshal(resource.InstanceState.TypedAttributes, &typedAttributes); err != nil {
		return fmt.Errorf("decode synthetics suite typed attributes: %w", err)
	}
	rawMessage, ok := typedAttributes[syntheticsSuiteMessageKey]
	messageHasValue := false
	if ok {
		var err error
		messageHasValue, err = syntheticsSuiteRawMessageHasValue(rawMessage)
		if err != nil {
			return fmt.Errorf("decode synthetics suite message state: %w", err)
		}
	}
	if messageHasValue {
		return nil
	}
	typedAttributes[syntheticsSuiteMessageKey] = json.RawMessage("\"\"")
	rawAttributes, err := json.Marshal(typedAttributes)
	if err != nil {
		return fmt.Errorf("encode synthetics suite typed attributes: %w", err)
	}
	resource.InstanceState.SetTypedAttributes(rawAttributes)
	return nil
}

func preserveSyntheticsSuiteEmptyTags(resource *terraformutils.Resource) error {
	hasEmptyTags, err := syntheticsSuiteStateHasEmptyTags(resource)
	if err != nil {
		return err
	}
	if !hasEmptyTags {
		return nil
	}
	if resource.Item == nil {
		resource.Item = map[string]interface{}{}
	}
	if value, ok := resource.Item[syntheticsSuiteTagsKey]; !ok || !syntheticsSuiteValueHasValue(value) {
		resource.Item[syntheticsSuiteTagsKey] = []interface{}{}
	}
	return preserveSyntheticsSuiteEmptyTagsState(resource)
}

func syntheticsSuiteStateHasEmptyTags(resource *terraformutils.Resource) (bool, error) {
	if resource == nil || resource.InstanceState == nil {
		return false, nil
	}
	if resource.InstanceState.Attributes != nil {
		if count, ok := resource.InstanceState.Attributes[syntheticsSuiteTagsKey+".#"]; ok && count == "0" {
			return true, nil
		}
	}
	if len(resource.InstanceState.TypedAttributes) == 0 {
		return false, nil
	}
	typedAttributes := map[string]json.RawMessage{}
	if err := json.Unmarshal(resource.InstanceState.TypedAttributes, &typedAttributes); err != nil {
		return false, fmt.Errorf("decode synthetics suite typed attributes: %w", err)
	}
	rawTags, ok := typedAttributes[syntheticsSuiteTagsKey]
	if !ok {
		return false, nil
	}
	tagsHaveValue, err := syntheticsSuiteRawMessageHasValue(rawTags)
	if err != nil {
		return false, fmt.Errorf("decode synthetics suite tags state: %w", err)
	}
	return !tagsHaveValue, nil
}

func preserveSyntheticsSuiteEmptyTagsState(resource *terraformutils.Resource) error {
	if resource == nil || resource.InstanceState == nil {
		return nil
	}
	if resource.InstanceState.Attributes == nil {
		resource.InstanceState.Attributes = map[string]string{}
	}
	resource.InstanceState.Attributes[syntheticsSuiteTagsKey+".#"] = "0"
	if len(resource.InstanceState.TypedAttributes) == 0 {
		return nil
	}

	typedAttributes := map[string]json.RawMessage{}
	if err := json.Unmarshal(resource.InstanceState.TypedAttributes, &typedAttributes); err != nil {
		return fmt.Errorf("decode synthetics suite typed attributes: %w", err)
	}
	rawTags, ok := typedAttributes[syntheticsSuiteTagsKey]
	tagsHaveValue := false
	if ok {
		var err error
		tagsHaveValue, err = syntheticsSuiteRawMessageHasValue(rawTags)
		if err != nil {
			return fmt.Errorf("decode synthetics suite tags state: %w", err)
		}
	}
	if tagsHaveValue {
		return nil
	}
	typedAttributes[syntheticsSuiteTagsKey] = json.RawMessage("[]")
	rawAttributes, err := json.Marshal(typedAttributes)
	if err != nil {
		return fmt.Errorf("encode synthetics suite typed attributes: %w", err)
	}
	resource.InstanceState.SetTypedAttributes(rawAttributes)
	return nil
}

func syntheticsSuiteRawMessageHasValue(rawValue json.RawMessage) (bool, error) {
	if len(bytes.TrimSpace(rawValue)) == 0 {
		return false, nil
	}

	var value interface{}
	decoder := json.NewDecoder(bytes.NewReader(rawValue))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return false, err
	}
	return syntheticsSuiteValueHasValue(value), nil
}

func syntheticsSuiteRawMessageIsEmptyString(rawValue json.RawMessage) (bool, error) {
	if len(bytes.TrimSpace(rawValue)) == 0 {
		return false, nil
	}

	var value *string
	if err := json.Unmarshal(rawValue, &value); err != nil {
		return false, err
	}
	return value != nil && *value == "", nil
}

func syntheticsSuiteValueHasValue(value interface{}) bool {
	switch value := value.(type) {
	case nil:
		return false
	case string:
		return value != ""
	case []interface{}:
		for _, item := range value {
			if syntheticsSuiteValueHasValue(item) {
				return true
			}
		}
		return false
	case map[string]interface{}:
		for _, item := range value {
			if syntheticsSuiteValueHasValue(item) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func (g *SyntheticsSuiteGenerator) createResources(suites []datadogV2.SyntheticsSuite) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, suite := range suites {
		resource, err := g.createResource(suite.GetPublicId())
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each Synthetics suite create 1 TerraformResource.
func (g *SyntheticsSuiteGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewSyntheticsApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	suites, err := listSyntheticsSuites(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(suites)
	if err != nil {
		return err
	}

	g.Resources = resources
	return nil
}

func (g *SyntheticsSuiteGenerator) filteredResources(auth context.Context, api *datadogV2.SyntheticsApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("synthetics_suite") {
			continue
		}

		filtered = true
		for _, value := range filter.AcceptableValues {
			suite, err := getSyntheticsSuite(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			data := suite.GetData()
			resource, err := g.createResource(data.GetId())
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, filtered, nil
}

func getSyntheticsSuite(auth context.Context, api *datadogV2.SyntheticsApi, suiteID string) (datadogV2.SyntheticsSuiteResponse, error) {
	resp, httpResp, err := api.GetSyntheticsSuite(auth, suiteID)
	closeDatadogResponseBody(httpResp)
	if err != nil {
		return datadogV2.SyntheticsSuiteResponse{}, err
	}
	data := resp.GetData()
	if data.GetId() == "" {
		return datadogV2.SyntheticsSuiteResponse{}, fmt.Errorf("synthetics suite %q not found", suiteID)
	}
	return resp, nil
}

func listSyntheticsSuites(auth context.Context, api *datadogV2.SyntheticsApi) ([]datadogV2.SyntheticsSuite, error) {
	suites := []datadogV2.SyntheticsSuite{}
	start := int64(0)

	for {
		optionalParams := datadogV2.NewSearchSuitesOptionalParameters().
			WithStart(start).
			WithCount(datadogSyntheticsSuitePageSize)

		resp, httpResp, err := api.SearchSuites(auth, *optionalParams)
		closeDatadogResponseBody(httpResp)
		if err != nil {
			return nil, err
		}

		data := resp.GetData()
		attrs := data.GetAttributes()
		pageSuites := attrs.GetSuites()
		suites = append(suites, pageSuites...)

		if len(pageSuites) == 0 || len(pageSuites) < int(datadogSyntheticsSuitePageSize) {
			break
		}
		if total := attrs.GetTotal(); total > 0 && len(suites) >= int(total) {
			break
		}
		start += int64(len(pageSuites))
	}

	return suites, nil
}
