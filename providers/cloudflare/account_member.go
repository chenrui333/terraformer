// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"

	cf "github.com/chenrui333/terraformer/providers/cloudflare/internal/cloudflarev7"
	"github.com/chenrui333/terraformer/terraformutils"
)

type AccountMemberGenerator struct {
	CloudflareService
}

func (g *AccountMemberGenerator) createAccountMemberResources(api *cf.API) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	pageOpt := cf.PaginationOptions{
		Page:    1,
		PerPage: 10}

	for {
		members, info, err := api.AccountMembers(context.Background(), g.accountID(), pageOpt)
		if err != nil {
			return resources, err
		}

		for _, member := range members {
			resource := terraformutils.NewResource(
				member.ID,
				member.ID,
				"cloudflare_account_member",
				"cloudflare",
				map[string]string{
					"account_id": g.accountID(),
					"email":      member.User.Email,
				},
				[]string{},
				accountMemberAdditionalFields(member),
			)
			setCloudflareImportID(&resource, g.accountID()+"/"+member.ID)
			resources = append(resources, resource)
		}

		if pageOpt.Page < info.TotalPages {
			pageOpt.Page++
		} else {
			break
		}
	}

	return resources, nil
}

func accountMemberAdditionalFields(member cf.AccountMember) map[string]interface{} {
	if len(member.Policies) > 0 {
		return map[string]interface{}{
			"policies": accountMemberPolicyAdditionalFields(member.Policies),
		}
	}

	roleIDs := make([]string, 0, len(member.Roles))
	for _, role := range member.Roles {
		roleIDs = append(roleIDs, role.ID)
	}
	if len(roleIDs) == 0 {
		return map[string]interface{}{}
	}

	return map[string]interface{}{
		"roles": roleIDs,
	}
}

func accountMemberPolicyAdditionalFields(policies []cf.Policy) []map[string]interface{} {
	fields := make([]map[string]interface{}, 0, len(policies))
	for _, policy := range policies {
		field := map[string]interface{}{
			"access":            policy.Access,
			"permission_groups": accountMemberPermissionGroupAdditionalFields(policy.PermissionGroups),
			"resource_groups":   accountMemberResourceGroupAdditionalFields(policy.ResourceGroups),
		}
		fields = append(fields, field)
	}
	return fields
}

func accountMemberPermissionGroupAdditionalFields(permissionGroups []cf.PermissionGroup) []map[string]interface{} {
	fields := make([]map[string]interface{}, 0, len(permissionGroups))
	for _, group := range permissionGroups {
		fields = append(fields, map[string]interface{}{
			"id": group.ID,
		})
	}
	return fields
}

func accountMemberResourceGroupAdditionalFields(resourceGroups []cf.ResourceGroup) []map[string]interface{} {
	fields := make([]map[string]interface{}, 0, len(resourceGroups))
	for _, group := range resourceGroups {
		fields = append(fields, map[string]interface{}{
			"id": group.ID,
		})
	}
	return fields
}

func (g *AccountMemberGenerator) InitResources() error {
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	resources, err := g.createAccountMemberResources(api)
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, resources...)

	return nil
}
