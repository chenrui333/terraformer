// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
)

var organizationAllowEmptyValues = []string{"tags."}

type OrganizationGenerator struct {
	AWSService
}

func (g *OrganizationGenerator) traverseNode(svc *organizations.Client, parentID string) {
	accountsForParent, err := svc.ListAccountsForParent(context.TODO(),
		&organizations.ListAccountsForParentInput{ParentId: aws.String(parentID)})
	if err != nil {
		return
	}
	for _, account := range accountsForParent.Accounts {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			StringValue(account.Id),
			StringValue(account.Name),
			"aws_organizations_organization",
			"aws",
			map[string]string{
				"id":  StringValue(account.Id),
				"arn": StringValue(account.Arn),
			},
			organizationAllowEmptyValues,
			map[string]interface{}{},
		))
		g.Resources = append(g.Resources, terraformutils.NewResource(
			StringValue(account.Id),
			StringValue(account.Name),
			"aws_organizations_account",
			"aws",
			map[string]string{
				"id":  StringValue(account.Id),
				"arn": StringValue(account.Arn),
			},
			organizationAllowEmptyValues,
			map[string]interface{}{},
		))
	}

	unitsForParent, err := svc.ListOrganizationalUnitsForParent(context.TODO(),
		&organizations.ListOrganizationalUnitsForParentInput{ParentId: aws.String(parentID)})
	if err != nil {
		return
	}
	for _, unit := range unitsForParent.OrganizationalUnits {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			StringValue(unit.Id),
			StringValue(unit.Name),
			"aws_organizations_organizational_unit",
			"aws",
			map[string]string{
				"id":  StringValue(unit.Id),
				"arn": StringValue(unit.Arn),
			},
			organizationAllowEmptyValues,
			map[string]interface{}{},
		))
		g.traverseNode(svc, StringValue(unit.Id))
	}
}

func (g *OrganizationGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := organizations.NewFromConfig(config)

	roots, err := svc.ListRoots(context.TODO(), &organizations.ListRootsInput{})
	if err != nil {
		return err
	}

	for _, root := range roots.Roots {
		nodeID := StringValue(root.Id)
		g.traverseNode(svc, nodeID)
	}

	p := organizations.NewListPoliciesPaginator(svc, &organizations.ListPoliciesInput{
		Filter: types.PolicyTypeServiceControlPolicy,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, policy := range page.Policies {
			policyID := StringValue(policy.Id)
			policyName := StringValue(policy.Name)
			g.Resources = append(g.Resources, terraformutils.NewResource(
				policyID,
				policyName,
				"aws_organizations_policy",
				"aws",
				map[string]string{
					"id":  policyID,
					"arn": StringValue(policy.Arn),
				},
				organizationAllowEmptyValues,
				map[string]interface{}{},
			))

			targetsForPolicy, err := svc.ListTargetsForPolicy(context.TODO(),
				&organizations.ListTargetsForPolicyInput{PolicyId: policy.Id})
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			for _, target := range targetsForPolicy.Targets {
				g.Resources = append(g.Resources, terraformutils.NewResource(
					StringValue(target.TargetId)+":"+policyID,
					"pa-"+StringValue(target.TargetId)+":"+policyName,
					"aws_organizations_policy_attachment",
					"aws",
					map[string]string{
						"policy_id": policyID,
						"target_id": StringValue(target.TargetId),
					},
					organizationAllowEmptyValues,
					map[string]interface{}{},
				))
			}
		}
	}

	return nil
}
