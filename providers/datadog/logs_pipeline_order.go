// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"fmt"
	"time"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// LogsPipelineOrderAllowEmptyValues ...
	LogsPipelineOrderAllowEmptyValues = []string{}
)

// LogsPipelineOrderGenerator ...
type LogsPipelineOrderGenerator struct {
	DatadogService
}

// InitResources Generate TerraformResources
func (g *LogsPipelineOrderGenerator) InitResources() error {
	currentDate := time.Now().Format("20060102150405")
	resourceName := fmt.Sprintf("logs_pipeline_order_%s", currentDate)
	g.Resources = append(g.Resources, terraformutils.NewResource(
		resourceName,
		resourceName,
		"datadog_logs_pipeline_order",
		"datadog",
		map[string]string{
			"name": resourceName,
		},
		LogsPipelineOrderAllowEmptyValues,
		map[string]interface{}{},
	))
	return nil
}
