// SPDX-License-Identifier: Apache-2.0

package ns1

import (
	"net/http"
	"time"

	"github.com/chenrui333/terraformer/terraformutils"
	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
)

type MonitoringJobGenerator struct {
	Ns1Service
}

func (g *MonitoringJobGenerator) createMonitoringJobResources(client *ns1.Client) error {
	jobs, resp, err := client.Jobs.List()
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}

	for _, j := range jobs {
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			j.ID,
			j.ID,
			"ns1_monitoringjob",
			"ns1",
			[]string{}))
	}

	return nil
}

func (g *MonitoringJobGenerator) InitResources() error {
	httpClient := &http.Client{Timeout: time.Second * 10}
	client := ns1.NewClient(httpClient, ns1.SetAPIKey(g.Args["api_key"].(string)))

	if err := g.createMonitoringJobResources(client); err != nil {
		return err
	}

	return nil
}
