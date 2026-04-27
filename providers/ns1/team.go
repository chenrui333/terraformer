// SPDX-License-Identifier: Apache-2.0

package ns1

import (
	"net/http"
	"time"

	"github.com/chenrui333/terraformer/terraformutils"
	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
)

type TeamGenerator struct {
	Ns1Service
}

func (g *TeamGenerator) createTeamResources(client *ns1.Client) error {
	teams, resp, err := client.Teams.List()
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}

	for _, t := range teams {
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			t.ID,
			t.ID,
			"ns1_team",
			"ns1",
			[]string{}))
	}

	return nil
}

func (g *TeamGenerator) InitResources() error {
	httpClient := &http.Client{Timeout: time.Second * 10}
	client := ns1.NewClient(httpClient, ns1.SetAPIKey(g.Args["api_key"].(string)))

	if err := g.createTeamResources(client); err != nil {
		return err
	}

	return nil
}
