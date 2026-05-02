// SPDX-License-Identifier: Apache-2.0

package datadog

import "github.com/chenrui333/terraformer/terraformutils"

const apmRetentionFilterOrderID = "filtersOrderID"

var (
	// APMRetentionFilterOrderAllowEmptyValues ...
	APMRetentionFilterOrderAllowEmptyValues = []string{"filter_ids"}
)

// APMRetentionFilterOrderGenerator ...
type APMRetentionFilterOrderGenerator struct {
	DatadogService
}

func (g *APMRetentionFilterOrderGenerator) createResource() terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		apmRetentionFilterOrderID,
		"apm_retention_filter_order",
		"datadog_apm_retention_filter_order",
		"datadog",
		APMRetentionFilterOrderAllowEmptyValues,
	)
}

func (g *APMRetentionFilterOrderGenerator) PostConvertHook() error {
	for i := range g.Resources {
		resource := &g.Resources[i]
		if resource.Item == nil {
			resource.Item = map[string]interface{}{}
		}
		if _, ok := resource.Item["filter_ids"]; ok {
			continue
		}
		if !apmRetentionFilterOrderStateHasEmptyFilterIDs(resource) {
			continue
		}
		resource.Item["filter_ids"] = []interface{}{}
	}
	return nil
}

func apmRetentionFilterOrderStateHasEmptyFilterIDs(resource *terraformutils.Resource) bool {
	if resource == nil || resource.InstanceState == nil || resource.InstanceState.Attributes == nil {
		return false
	}
	return resource.InstanceState.Attributes["filter_ids.#"] == "0"
}

// InitResources Generate TerraformResources for the singleton APM retention filter order.
// The Datadog provider read path ignores the import ID and stores filtersOrderID.
func (g *APMRetentionFilterOrderGenerator) InitResources() error {
	for i, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("apm_retention_filter_order") {
			g.Filter[i].AcceptableValues = []string{apmRetentionFilterOrderID}
		}
	}

	g.Resources = []terraformutils.Resource{g.createResource()}
	return nil
}
