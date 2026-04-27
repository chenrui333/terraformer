// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// LogsArchiveOrderAllowEmptyValues ...
	LogsArchiveOrderAllowEmptyValues = []string{}
)

// LogsArchiveOrderGenerator ...
type LogsArchiveOrderGenerator struct {
	DatadogService
}

// InitResources Generate TerraformResources
func (g *LogsArchiveOrderGenerator) InitResources() error {
	g.Resources = append(g.Resources, terraformutils.NewResource(
		"archiveOrderID",
		"archiveOrderID",
		"datadog_logs_archive_order",
		"datadog",
		map[string]string{},
		LogsArchiveOrderAllowEmptyValues,
		map[string]interface{}{},
	))
	return nil
}
