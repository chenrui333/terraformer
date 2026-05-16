// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/smithy-go"
)

var IamAllowEmptyValues = []string{"tags."}

var IamAdditionalFields = map[string]interface{}{}

type IamGenerator struct {
	AWSService
}

type iamOptionalResourceLoader struct {
	name string
	load func() error
}

func (g *IamGenerator) loadOptionalResources(loaders []iamOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			if iamResourceMissing(err) {
				continue
			}
			log.Printf("Skipping IAM %s: %v", loader.name, err)
		}
	}
}

func (g *IamGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := iam.NewFromConfig(config)
	g.Resources = []terraformutils.Resource{}
	err := g.getUsers(svc)
	if err != nil {
		log.Println(err)
	}

	err = g.getGroups(svc)
	if err != nil {
		log.Println(err)
	}

	err = g.getPolicies(svc)
	if err != nil {
		log.Println(err)
	}

	err = g.getRoles(svc)
	if err != nil {
		log.Println(err)
	}

	err = g.getInstanceProfiles(svc)
	if err != nil {
		log.Println(err)
	}

	g.loadOptionalResources([]iamOptionalResourceLoader{
		{name: "account alias", load: func() error { return g.getAccountAlias(svc) }},
		{name: "account password policy", load: func() error { return g.getAccountPasswordPolicy(svc) }},
		{name: "OpenID Connect providers", load: func() error { return g.getOpenIDConnectProviders(svc) }},
		{name: "SAML providers", load: func() error { return g.getSAMLProviders(svc) }},
		{name: "service-linked roles", load: func() error { return g.getServiceLinkedRoles(svc) }},
		{name: "server certificates", load: func() error { return g.getServerCertificates(svc) }},
	})

	return nil
}

func (g *IamGenerator) getAccountAlias(svc *iam.Client) error {
	p := iam.NewListAccountAliasesPaginator(svc, &iam.ListAccountAliasesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, alias := range page.AccountAliases {
			if alias == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				alias,
				alias,
				"aws_iam_account_alias",
				"aws",
				map[string]string{"account_alias": alias},
				IamAllowEmptyValues,
				IamAdditionalFields,
			))
		}
	}
	return nil
}

func (g *IamGenerator) getAccountPasswordPolicy(svc *iam.Client) error {
	if _, err := svc.GetAccountPasswordPolicy(context.TODO(), &iam.GetAccountPasswordPolicyInput{}); err != nil {
		if iamResourceMissing(err) {
			return nil
		}
		return err
	}
	g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
		"iam-account-password-policy",
		"account_password_policy",
		"aws_iam_account_password_policy",
		"aws",
		IamAllowEmptyValues,
	))
	return nil
}

func (g *IamGenerator) getOpenIDConnectProviders(svc *iam.Client) error {
	output, err := svc.ListOpenIDConnectProviders(context.TODO(), &iam.ListOpenIDConnectProvidersInput{})
	if err != nil {
		return err
	}
	for _, provider := range output.OpenIDConnectProviderList {
		providerARN := StringValue(provider.Arn)
		if providerARN == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			providerARN,
			iamResourceName("oidc", iamOpenIDConnectProviderName(providerARN)),
			"aws_iam_openid_connect_provider",
			"aws",
			IamAllowEmptyValues,
		))
	}
	return nil
}

func (g *IamGenerator) getSAMLProviders(svc *iam.Client) error {
	output, err := svc.ListSAMLProviders(context.TODO(), &iam.ListSAMLProvidersInput{})
	if err != nil {
		return err
	}
	for _, provider := range output.SAMLProviderList {
		providerARN := StringValue(provider.Arn)
		providerName := arnLastSegment(providerARN, "/")
		if providerARN == "" || providerName == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			providerARN,
			iamResourceName("saml", providerName),
			"aws_iam_saml_provider",
			"aws",
			IamAllowEmptyValues,
		))
	}
	return nil
}

func iamOpenIDConnectProviderName(providerARN string) string {
	if _, providerName, ok := strings.Cut(providerARN, ":oidc-provider/"); ok {
		return providerName
	}
	return arnLastSegment(providerARN, "/")
}

func iamResourceName(parts ...string) string {
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			segments = append(segments, part)
		}
	}
	if len(segments) == 0 {
		return "iam_resource"
	}
	return strings.Join(segments, "/")
}

func iamResourceMissing(err error) bool {
	var noSuchEntity *types.NoSuchEntityException
	if errors.As(err, &noSuchEntity) {
		return true
	}
	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && strings.Contains(strings.ToLower(apiErr.ErrorCode()), "nosuchentity")
}

func (g *IamGenerator) getRoles(svc *iam.Client) error {
	p := iam.NewListRolesPaginator(svc, &iam.ListRolesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, role := range page.Roles {
			roleName := StringValue(role.RoleName)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				roleName,
				roleName,
				"aws_iam_role",
				"aws",
				IamAllowEmptyValues))
			rolePoliciesPage := iam.NewListRolePoliciesPaginator(svc, &iam.ListRolePoliciesInput{RoleName: role.RoleName})
			for rolePoliciesPage.HasMorePages() {
				rolePoliciesNextPage, err := rolePoliciesPage.NextPage(context.TODO())
				if err != nil {
					log.Println(err)
					continue
				}
				for _, policyName := range rolePoliciesNextPage.PolicyNames {
					g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
						roleName+":"+policyName,
						roleName+"_"+policyName,
						"aws_iam_role_policy",
						"aws",
						IamAllowEmptyValues))
				}
			}
			roleAttachedPoliciesPage := iam.NewListAttachedRolePoliciesPaginator(svc, &iam.ListAttachedRolePoliciesInput{
				RoleName: &roleName,
			})
			for roleAttachedPoliciesPage.HasMorePages() {
				roleAttachedPoliciesNextPage, err := roleAttachedPoliciesPage.NextPage(context.TODO())
				if err != nil {
					log.Println(err)
					continue
				}
				for _, attachedPolicy := range roleAttachedPoliciesNextPage.AttachedPolicies {
					g.Resources = append(g.Resources, terraformutils.NewResource(
						roleName+"/"+*attachedPolicy.PolicyArn,
						roleName+"_"+*attachedPolicy.PolicyName,
						"aws_iam_role_policy_attachment",
						"aws",
						map[string]string{
							"role":       roleName,
							"policy_arn": *attachedPolicy.PolicyArn,
						},
						IamAllowEmptyValues,
						map[string]interface{}{}))
				}
			}
		}
	}
	return nil
}

func (g *IamGenerator) getUsers(svc *iam.Client) error {
	p := iam.NewListUsersPaginator(svc, &iam.ListUsersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, user := range page.Users {
			resourceName := StringValue(user.UserName)
			g.Resources = append(g.Resources, terraformutils.NewResource(
				resourceName,
				StringValue(user.UserId),
				"aws_iam_user",
				"aws",
				map[string]string{
					"force_destroy": "false",
				},
				IamAllowEmptyValues,
				map[string]interface{}{}))
			err := g.getUserPolices(svc, user.UserName)
			if err != nil {
				log.Println(err)
			}
			err = g.getUserPolicyAttachment(svc, user.UserName)
			if err != nil {
				log.Println(err)
			}
			err = g.getUserGroup(svc, user.UserName)
			if err != nil {
				log.Println(err)
			}
			err = g.getUserAccessKey(svc, user.UserName, StringValue(user.UserId))
			if err != nil {
				log.Println(err)
			}
		}
	}
	return nil
}

func (g *IamGenerator) getUserGroup(svc *iam.Client, userName *string) error {
	p := iam.NewListGroupsForUserPaginator(svc, &iam.ListGroupsForUserInput{UserName: userName})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, group := range page.Groups {
			userGroupMembership := *userName + "/" + *group.GroupName
			g.Resources = append(g.Resources, terraformutils.NewResource(
				userGroupMembership,
				userGroupMembership,
				"aws_iam_user_group_membership",
				"aws",
				map[string]string{
					"user":     *userName,
					"groups.#": "1",
					"groups.0": *group.GroupName,
				},
				IamAllowEmptyValues,
				IamAdditionalFields,
			))
		}
	}
	return nil
}

func (g *IamGenerator) getUserPolices(svc *iam.Client, userName *string) error {
	p := iam.NewListUserPoliciesPaginator(svc, &iam.ListUserPoliciesInput{UserName: userName})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, policy := range page.PolicyNames {
			resourceName := StringValue(userName) + "_" + policy
			resourceName = strings.ReplaceAll(resourceName, "@", "")
			policyID := StringValue(userName) + ":" + policy
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				policyID,
				resourceName,
				"aws_iam_user_policy",
				"aws",
				IamAllowEmptyValues))
		}
	}
	return nil
}

func (g *IamGenerator) getUserPolicyAttachment(svc *iam.Client, userName *string) error {
	p := iam.NewListAttachedUserPoliciesPaginator(svc, &iam.ListAttachedUserPoliciesInput{
		UserName: userName,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, attachedPolicy := range page.AttachedPolicies {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				*userName+"/"+*attachedPolicy.PolicyArn,
				*userName+"_"+*attachedPolicy.PolicyName,
				"aws_iam_user_policy_attachment",
				"aws",
				map[string]string{
					"user":       *userName,
					"policy_arn": *attachedPolicy.PolicyArn,
				},
				IamAllowEmptyValues,
				map[string]interface{}{}))
		}
	}
	return nil
}

func (g *IamGenerator) getPolicies(svc *iam.Client) error {
	p := iam.NewListPoliciesPaginator(svc, &iam.ListPoliciesInput{Scope: types.PolicyScopeTypeLocal})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, policy := range page.Policies {
			resourceName := StringValue(policy.PolicyName)
			policyARN := StringValue(policy.Arn)

			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				policyARN,
				resourceName,
				"aws_iam_policy",
				"aws",
				IamAllowEmptyValues))
		}
	}
	return nil
}

func (g *IamGenerator) getGroups(svc *iam.Client) error {
	p := iam.NewListGroupsPaginator(svc, &iam.ListGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, group := range page.Groups {
			resourceName := StringValue(group.GroupName)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_iam_group",
				"aws",
				IamAllowEmptyValues))
			g.getGroupPolicies(svc, group)
			g.getAttachedGroupPolicies(svc, group)
		}
	}
	return nil
}

func (g *IamGenerator) getGroupPolicies(svc *iam.Client, group types.Group) {
	groupPoliciesPage := iam.NewListGroupPoliciesPaginator(svc, &iam.ListGroupPoliciesInput{GroupName: group.GroupName})
	for groupPoliciesPage.HasMorePages() {
		groupPoliciesNextPage, err := groupPoliciesPage.NextPage(context.TODO())
		if err != nil {
			log.Println(err)
			continue
		}
		for _, policy := range groupPoliciesNextPage.PolicyNames {
			id := *group.GroupName + ":" + policy
			groupPolicyName := *group.GroupName + "_" + policy
			g.Resources = append(g.Resources, terraformutils.NewResource(
				id,
				groupPolicyName,
				"aws_iam_group_policy",
				"aws",
				map[string]string{},
				IamAllowEmptyValues,
				IamAdditionalFields))
		}
	}
}

func (g *IamGenerator) getAttachedGroupPolicies(svc *iam.Client, group types.Group) {
	groupAttachedPoliciesPage := iam.NewListAttachedGroupPoliciesPaginator(svc,
		&iam.ListAttachedGroupPoliciesInput{GroupName: group.GroupName})
	for groupAttachedPoliciesPage.HasMorePages() {
		groupAttachedPoliciesNextPage, err := groupAttachedPoliciesPage.NextPage(context.TODO())
		if err != nil {
			log.Println(err)
			continue
		}
		for _, attachedPolicy := range groupAttachedPoliciesNextPage.AttachedPolicies {
			if !strings.Contains(*attachedPolicy.PolicyArn, "arn:aws:iam::aws") {
				continue // map only AWS managed policies since others should be managed by
			}
			id := *group.GroupName + "/" + *attachedPolicy.PolicyArn
			g.Resources = append(g.Resources, terraformutils.NewResource(
				id,
				*group.GroupName+"_"+*attachedPolicy.PolicyName,
				"aws_iam_group_policy_attachment",
				"aws",
				map[string]string{
					"group":      *group.GroupName,
					"policy_arn": *attachedPolicy.PolicyArn,
				},
				IamAllowEmptyValues,
				IamAdditionalFields))
		}
	}
}

func (g *IamGenerator) getInstanceProfiles(svc *iam.Client) error {
	p := iam.NewListInstanceProfilesPaginator(svc, &iam.ListInstanceProfilesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, instanceProfile := range page.InstanceProfiles {
			resourceName := *instanceProfile.InstanceProfileName

			g.Resources = append(g.Resources, terraformutils.NewResource(
				resourceName,
				resourceName,
				"aws_iam_instance_profile",
				"aws",
				map[string]string{
					"name": resourceName,
				},
				IamAllowEmptyValues,
				IamAdditionalFields))
		}
	}
	return nil
}

func (g *IamGenerator) getUserAccessKey(svc *iam.Client, userName *string, userID string) error {
	p := iam.NewListAccessKeysPaginator(svc, &iam.ListAccessKeysInput{UserName: userName})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, key := range page.AccessKeyMetadata {
			accessKeyID := StringValue(key.AccessKeyId)
			g.Resources = append(g.Resources, terraformutils.NewResource(
				accessKeyID,
				accessKeyID,
				"aws_iam_access_key",
				"aws",
				map[string]string{
					"user": *userName,
				},
				IamAllowEmptyValues,
				map[string]interface{}{
					"depends_on": []string{"aws_iam_user.tfer--" + userID},
				}))
		}
	}
	return nil
}

func (g *IamGenerator) getServiceLinkedRoles(svc *iam.Client) error {
	slrPathPrefix := "/aws-service-role/"
	p := iam.NewListRolesPaginator(svc, &iam.ListRolesInput{
		PathPrefix: &slrPathPrefix,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, role := range page.Roles {
			roleARN := StringValue(role.Arn)
			if roleARN == "" {
				continue
			}
			roleName := StringValue(role.RoleName)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				roleARN,
				iamResourceName("slr", roleName),
				"aws_iam_service_linked_role",
				"aws",
				IamAllowEmptyValues))
		}
	}
	return nil
}

func (g *IamGenerator) getServerCertificates(svc *iam.Client) error {
	p := iam.NewListServerCertificatesPaginator(svc, &iam.ListServerCertificatesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, cert := range page.ServerCertificateMetadataList {
			certName := StringValue(cert.ServerCertificateName)
			if certName == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				certName,
				iamResourceName("cert", certName),
				"aws_iam_server_certificate",
				"aws",
				IamAllowEmptyValues))
		}
	}
	return nil
}

// PostGenerateHook for add policy json as heredoc
func (g *IamGenerator) PostConvertHook() error {
	for i, resource := range g.Resources {
		switch resource.InstanceInfo.Type {
		case "aws_iam_policy", "aws_iam_user_policy", "aws_iam_group_policy", "aws_iam_role_policy":
			policy := g.escapeAwsInterpolation(resource.Item["policy"].(string))
			resource.Item["policy"] = fmt.Sprintf(`<<POLICY
%s
POLICY`, policy)
		case "aws_iam_role":
			policy := g.escapeAwsInterpolation(resource.Item["assume_role_policy"].(string))
			g.Resources[i].Item["assume_role_policy"] = fmt.Sprintf(`<<POLICY
%s
POLICY`, policy)
		case "aws_iam_instance_profile":
			delete(resource.Item, "roles")
		}
	}
	return nil
}
