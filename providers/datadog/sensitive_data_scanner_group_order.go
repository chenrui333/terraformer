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
	// SensitiveDataScannerGroupOrderAllowEmptyValues ...
	SensitiveDataScannerGroupOrderAllowEmptyValues = []string{"group_ids"}
)

// SensitiveDataScannerGroupOrderGenerator ...
type SensitiveDataScannerGroupOrderGenerator struct {
	DatadogService
}

// InitResources Generate TerraformResources from Datadog API,
// from the sensitive_data_scanner_group_order singleton create 1 TerraformResource.
func (g *SensitiveDataScannerGroupOrderGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewSensitiveDataScannerApi(datadogClient)

	config, err := listSensitiveDataScannerConfig(auth, api)
	if err != nil {
		return err
	}

	resource := g.createResource(config)
	g.Resources = append(g.Resources, resource)
	return nil
}

func (g *SensitiveDataScannerGroupOrderGenerator) createResource(config datadogV2.SensitiveDataScannerGetConfigResponse) terraformutils.Resource {
	configID, groupIDs := sensitiveDataScannerGroupOrderFromConfig(config)
	attributes := map[string]string{
		"group_ids.#": strconv.Itoa(len(groupIDs)),
	}
	for index, groupID := range groupIDs {
		attributes[fmt.Sprintf("group_ids.%d", index)] = groupID
	}

	return terraformutils.NewResource(
		configID,
		"sensitive_data_scanner_group_order",
		"datadog_sensitive_data_scanner_group_order",
		"datadog",
		attributes,
		SensitiveDataScannerGroupOrderAllowEmptyValues,
		map[string]interface{}{},
	)
}

func (g *SensitiveDataScannerGroupOrderGenerator) PostConvertHook() error {
	for i := range g.Resources {
		resource := &g.Resources[i]
		if resource.Item == nil {
			resource.Item = map[string]interface{}{}
		}
		if _, ok := resource.Item["group_ids"]; ok {
			continue
		}
		if !sensitiveDataScannerStateHasEmptyList(resource, "group_ids") {
			continue
		}
		resource.Item["group_ids"] = []interface{}{}
	}
	return nil
}

func sensitiveDataScannerGroupOrderFromConfig(config datadogV2.SensitiveDataScannerGetConfigResponse) (string, []string) {
	configID := "order"
	groupIDs := []string{}

	data, ok := config.GetDataOk()
	if !ok {
		return configID, groupIDs
	}
	if data.GetId() != "" {
		configID = data.GetId()
	}
	relationships, ok := data.GetRelationshipsOk()
	if !ok {
		return configID, groupIDs
	}
	groups, ok := relationships.GetGroupsOk()
	if !ok {
		return configID, groupIDs
	}
	for _, group := range groups.GetData() {
		if group.GetId() == "" {
			continue
		}
		groupIDs = append(groupIDs, group.GetId())
	}

	return configID, groupIDs
}
