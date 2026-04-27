// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"errors"

	"github.com/chenrui333/terraformer/terraformutils"
	"google.golang.org/api/iterator"

	"cloud.google.com/go/logging/logadmin"
)

var loggingAllowEmptyValues = []string{}

var loggingAdditionalFields = map[string]interface{}{}

type LoggingGenerator struct {
	GCPService
}

func (g *LoggingGenerator) loadLoggingMetrics(ctx context.Context, client *logadmin.Client) error {
	metricIterator := client.Metrics(ctx)

	for {
		metric, err := metricIterator.Next()

		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			metric.ID,
			metric.ID,
			"google_logging_metric",
			g.ProviderName,
			map[string]string{
				"name":    metric.ID,
				"project": g.GetArgs()["project"].(string),
			},
			loggingAllowEmptyValues,
			loggingAdditionalFields,
		))
	}
	return nil
}

// Generate TerraformResources from GCP API
func (g *LoggingGenerator) InitResources() error {
	project := g.GetArgs()["project"].(string)
	ctx := context.Background()
	client, err := logadmin.NewClient(ctx, project)
	if err != nil {
		return err
	}

	if err := g.loadLoggingMetrics(ctx, client); err != nil {
		return err
	}

	return nil
}
