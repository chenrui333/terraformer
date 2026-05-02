// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v16"
)

type TeamMemberGenerator struct {
	LaunchDarklyService
}

func (g *TeamMemberGenerator) loadTeamMembers(ctx context.Context, client *ldapi.APIClient) error {
	var allMembers []ldapi.Member
	for offset := int64(0); ; offset += pageSize {
		members, _, err := client.AccountMembersApi.GetMembers(ctx).
			Limit(pageSize).
			Offset(offset).
			Execute()
		if err != nil {
			return err
		}
		if members == nil {
			break
		}
		allMembers = append(allMembers, members.Items...)
		if members.TotalCount == nil || int64(len(allMembers)) >= int64(*members.TotalCount) {
			break
		}
	}
	for _, member := range allMembers {
		resource := terraformutils.NewResource(
			member.Id,
			resourceName(member.Email, member.Id),
			"launchdarkly_team_member",
			"launchdarkly",
			map[string]string{},
			[]string{},
			map[string]interface{}{})
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *TeamMemberGenerator) InitResources() error {
	return g.loadTeamMembers(g.GetArgs()["ctx"].(context.Context), g.GetArgs()["client"].(*ldapi.APIClient))
}
