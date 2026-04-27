// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"fmt"
	"time"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// LogsIndexOrderAllowEmptyValues ...
	LogsIndexOrderAllowEmptyValues = []string{}
)

// LogsIndexOrderGenerator ...
type LogsIndexOrderGenerator struct {
	DatadogService
}

// InitResources Generate TerraformResources
func (g *LogsIndexOrderGenerator) InitResources() error {
	currentDate := time.Now().Format("20060102150405")
	resourceName := fmt.Sprintf("logs_index_order_%s", currentDate)
	g.Resources = append(g.Resources, terraformutils.NewResource(
		resourceName,
		resourceName,
		"datadog_logs_index_order",
		"datadog",
		map[string]string{
			"name": resourceName,
		},
		LogsIndexOrderAllowEmptyValues,
		map[string]interface{}{},
	))
	return nil
}
