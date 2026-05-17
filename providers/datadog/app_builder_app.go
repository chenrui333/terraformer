// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/google/uuid"

	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	datadogAppBuilderAppServiceName = "app_builder_app"
	datadogAppBuilderAppPageLimit   = int64(100)

	appBuilderAppActionQueryNamesToConnectionIDsKey = "action_query_names_to_connection_ids"
)

var (
	// AppBuilderAppAllowEmptyValues ...
	AppBuilderAppAllowEmptyValues = []string{appBuilderAppActionQueryNamesToConnectionIDsKey + "."}
)

// AppBuilderAppGenerator ...
type AppBuilderAppGenerator struct {
	DatadogService
}

func (g *AppBuilderAppGenerator) createResource(appID string) (terraformutils.Resource, error) {
	return newDatadogIDResource(datadogAppBuilderAppServiceName, appID, AppBuilderAppAllowEmptyValues)
}

func (g *AppBuilderAppGenerator) createResources(appIDs []string) ([]terraformutils.Resource, error) {
	return datadogIDResources(datadogAppBuilderAppServiceName, appIDs, AppBuilderAppAllowEmptyValues)
}

func (g *AppBuilderAppGenerator) PostConvertHook() error {
	for i := range g.Resources {
		if err := preserveAppBuilderAppEmptyActionQueryMap(&g.Resources[i]); err != nil {
			return err
		}
	}
	return nil
}

func preserveAppBuilderAppEmptyActionQueryMap(resource *terraformutils.Resource) error {
	hasEmptyMap, err := appBuilderAppStateHasEmptyActionQueryMap(resource)
	if err != nil {
		return err
	}
	if !hasEmptyMap {
		return nil
	}
	if resource.Item == nil {
		resource.Item = map[string]interface{}{}
	}
	if value, ok := resource.Item[appBuilderAppActionQueryNamesToConnectionIDsKey]; !ok || !appBuilderAppValueHasValue(value) {
		resource.Item[appBuilderAppActionQueryNamesToConnectionIDsKey] = map[string]interface{}{}
	}
	return preserveAppBuilderAppEmptyActionQueryMapState(resource)
}

func appBuilderAppStateHasEmptyActionQueryMap(resource *terraformutils.Resource) (bool, error) {
	if resource == nil || resource.InstanceState == nil {
		return false, nil
	}
	if resource.InstanceState.Attributes != nil {
		if count, ok := resource.InstanceState.Attributes[appBuilderAppActionQueryNamesToConnectionIDsKey+".%"]; ok && count == "0" {
			return true, nil
		}
	}
	if len(resource.InstanceState.TypedAttributes) == 0 {
		return false, nil
	}
	typedAttributes := map[string]json.RawMessage{}
	if err := json.Unmarshal(resource.InstanceState.TypedAttributes, &typedAttributes); err != nil {
		return false, err
	}
	rawValue, ok := typedAttributes[appBuilderAppActionQueryNamesToConnectionIDsKey]
	if !ok {
		return false, nil
	}
	return appBuilderAppRawMessageIsEmptyMap(rawValue)
}

func preserveAppBuilderAppEmptyActionQueryMapState(resource *terraformutils.Resource) error {
	if resource == nil || resource.InstanceState == nil {
		return nil
	}
	if resource.InstanceState.Attributes == nil {
		resource.InstanceState.Attributes = map[string]string{}
	}
	resource.InstanceState.Attributes[appBuilderAppActionQueryNamesToConnectionIDsKey+".%"] = "0"
	if len(resource.InstanceState.TypedAttributes) == 0 {
		return nil
	}

	typedAttributes := map[string]json.RawMessage{}
	if err := json.Unmarshal(resource.InstanceState.TypedAttributes, &typedAttributes); err != nil {
		return err
	}
	rawValue, ok := typedAttributes[appBuilderAppActionQueryNamesToConnectionIDsKey]
	if ok {
		emptyMap, err := appBuilderAppRawMessageIsEmptyMap(rawValue)
		if err != nil {
			return err
		}
		if !emptyMap {
			return nil
		}
	}
	typedAttributes[appBuilderAppActionQueryNamesToConnectionIDsKey] = json.RawMessage("{}")
	rawAttributes, err := json.Marshal(typedAttributes)
	if err != nil {
		return err
	}
	resource.InstanceState.SetTypedAttributes(rawAttributes)
	return nil
}

func appBuilderAppRawMessageIsEmptyMap(rawValue json.RawMessage) (bool, error) {
	if len(bytes.TrimSpace(rawValue)) == 0 {
		return false, nil
	}
	var value map[string]interface{}
	if err := json.Unmarshal(rawValue, &value); err != nil {
		return false, err
	}
	return len(value) == 0, nil
}

func appBuilderAppValueHasValue(value interface{}) bool {
	switch typedValue := value.(type) {
	case nil:
		return false
	case map[string]interface{}:
		return len(typedValue) > 0
	case map[string]string:
		return len(typedValue) > 0
	default:
		return true
	}
}

// InitResources Generate TerraformResources from Datadog API,
// from each app_builder_app create 1 TerraformResource.
func (g *AppBuilderAppGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewAppBuilderApi(datadogClient)

	resources, hasIDFilter, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if hasIDFilter {
		g.Resources = resources
		return nil
	}

	appIDs, err := g.listAppBuilderAppIDs(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(appIDs)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}

func (g *AppBuilderAppGenerator) filteredResources(auth context.Context, api *datadogV2.AppBuilderApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	hasIDFilter := false
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || filter.ServiceName != datadogAppBuilderAppServiceName {
			continue
		}
		hasIDFilter = true
		for _, value := range filter.AcceptableValues {
			appID, err := g.getAppBuilderAppID(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(appID)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}
	return resources, hasIDFilter, nil
}

func (g *AppBuilderAppGenerator) getAppBuilderAppID(auth context.Context, api *datadogV2.AppBuilderApi, appID string) (string, error) {
	parsedAppID, err := uuid.Parse(appID)
	if err != nil {
		return "", err
	}
	resp, httpResp, err := api.GetApp(auth, parsedAppID)
	closeDatadogResponseBody(httpResp)
	if err != nil {
		return "", err
	}
	data := resp.GetData()
	responseID := data.GetId()
	if responseID == uuid.Nil {
		return appID, nil
	}
	return responseID.String(), nil
}

func (g *AppBuilderAppGenerator) listAppBuilderAppIDs(auth context.Context, api *datadogV2.AppBuilderApi) ([]string, error) {
	ids := []string{}
	pageNumber := int64(0)
	appsSeen := int64(0)

	for {
		opts := datadogV2.NewListAppsOptionalParameters().
			WithLimit(datadogAppBuilderAppPageLimit).
			WithPage(pageNumber)

		resp, httpResp, err := api.ListApps(auth, *opts)
		closeDatadogResponseBody(httpResp)
		if err != nil {
			return nil, err
		}

		apps := resp.GetData()
		for _, app := range apps {
			appID := app.GetId()
			if appID == uuid.Nil {
				continue
			}
			ids = append(ids, appID.String())
		}
		if len(apps) == 0 {
			break
		}
		appsSeen += int64(len(apps))

		meta := resp.GetMeta()
		pageMeta := meta.GetPage()
		if totalFiltered, ok := pageMeta.GetTotalFilteredCountOk(); ok {
			if appsSeen >= *totalFiltered {
				break
			}
		} else if int64(len(apps)) < datadogAppBuilderAppPageLimit {
			break
		}
		pageNumber++
	}

	return ids, nil
}
