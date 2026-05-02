// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v16"
)

type TeamGenerator struct {
	LaunchDarklyService
}

func (g *TeamGenerator) loadTeams(ctx context.Context, client *ldapi.APIClient) error {
	var allTeams []ldapi.Team
	for offset := int64(0); ; offset += pageSize {
		teams, resp, err := client.TeamsApi.GetTeams(ctx).
			Limit(pageSize).
			Offset(offset).
			Execute()
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if err != nil {
			return err
		}
		if teams == nil {
			break
		}
		allTeams = append(allTeams, teams.Items...)
		if teams.TotalCount == nil || int64(len(allTeams)) >= int64(*teams.TotalCount) {
			break
		}
	}
	for _, team := range allTeams {
		teamKey := team.GetKey()
		resource := terraformutils.NewResource(
			teamKey,
			resourceName(team.GetName(), teamKey),
			"launchdarkly_team",
			"launchdarkly",
			map[string]string{
				"key": teamKey,
			},
			[]string{},
			map[string]interface{}{})
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *TeamGenerator) InitResources() error {
	return g.loadTeams(g.GetArgs()["ctx"].(context.Context), g.GetArgs()["client"].(*ldapi.APIClient))
}
