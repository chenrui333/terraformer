// SPDX-License-Identifier: Apache-2.0

package logzio

import (
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"

	alerts "github.com/logzio/logzio_terraform_client/alerts_v2"
)

type AlertsGenerator struct {
	LogzioService
}

// Generate Terraform Resources from Logzio API,
func (g *AlertsGenerator) InitResources() error {
	client, err := alerts.New(g.Args["api_token"].(string), g.Args["base_url"].(string))
	if err != nil {
		return err
	}

	alerts, err := client.ListAlerts()
	if err != nil {
		return err
	}
	allowedEmptyValues := []string{"alert_notification_endpoints.#", "notification_emails.#"}
	for _, alert := range alerts {
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			strconv.FormatInt(alert.AlertId, 10),
			createSlug(alert.Title+"-"+strconv.FormatInt(alert.AlertId, 10)),
			"logzio_alert",
			"logzio",
			allowedEmptyValues,
		))
	}
	return nil
}
