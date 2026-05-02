// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v16"
)

type CustomRoleGenerator struct {
	LaunchDarklyService
}

func (g *CustomRoleGenerator) loadCustomRoles(ctx context.Context, client *ldapi.APIClient) error {
	var allRoles []ldapi.CustomRole
	for offset := int64(0); ; offset += pageSize {
		roles, _, err := client.CustomRolesApi.GetCustomRoles(ctx).
			Limit(pageSize).
			Offset(offset).
			Execute()
		if err != nil {
			return err
		}
		if roles == nil {
			break
		}
		allRoles = append(allRoles, roles.Items...)
		if len(roles.Items) < pageSize {
			break
		}
	}
	for _, role := range allRoles {
		resource := terraformutils.NewResource(
			role.Key,
			role.Key,
			"launchdarkly_custom_role",
			"launchdarkly",
			map[string]string{
				"key": role.Key,
			},
			[]string{},
			map[string]interface{}{})
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *CustomRoleGenerator) InitResources() error {
	return g.loadCustomRoles(g.GetArgs()["ctx"].(context.Context), g.GetArgs()["client"].(*ldapi.APIClient))
}
