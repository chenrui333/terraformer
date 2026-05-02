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

type organizationClient interface {
	ListAccountsForParent(context.Context, *organizations.ListAccountsForParentInput, ...func(*organizations.Options)) (*organizations.ListAccountsForParentOutput, error)
	ListOrganizationalUnitsForParent(context.Context, *organizations.ListOrganizationalUnitsForParentInput, ...func(*organizations.Options)) (*organizations.ListOrganizationalUnitsForParentOutput, error)
	ListTargetsForPolicy(context.Context, *organizations.ListTargetsForPolicyInput, ...func(*organizations.Options)) (*organizations.ListTargetsForPolicyOutput, error)
}

func (g *OrganizationGenerator) traverseNode(svc organizationClient, parentID string) error {
	var accountNextToken *string
	for {
		accountsForParent, err := svc.ListAccountsForParent(context.TODO(),
			&organizations.ListAccountsForParentInput{
				ParentId:  aws.String(parentID),
				NextToken: accountNextToken,
			})
		if err != nil {
			return fmt.Errorf("list organization accounts for parent %s: %w", parentID, err)
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
		accountNextToken = accountsForParent.NextToken
		if accountNextToken == nil {
			break
		}
	}

	var unitNextToken *string
	for {
		unitsForParent, err := svc.ListOrganizationalUnitsForParent(context.TODO(),
			&organizations.ListOrganizationalUnitsForParentInput{
				ParentId:  aws.String(parentID),
				NextToken: unitNextToken,
			})
		if err != nil {
			return fmt.Errorf("list organization units for parent %s: %w", parentID, err)
		}
		for _, unit := range unitsForParent.OrganizationalUnits {
			unitID := StringValue(unit.Id)
			g.Resources = append(g.Resources, terraformutils.NewResource(
				unitID,
				StringValue(unit.Name),
				"aws_organizations_organizational_unit",
				"aws",
				map[string]string{
					"id":  unitID,
					"arn": StringValue(unit.Arn),
				},
				organizationAllowEmptyValues,
				map[string]interface{}{},
			))
			if err := g.traverseNode(svc, unitID); err != nil {
				return err
			}
		}
		unitNextToken = unitsForParent.NextToken
		if unitNextToken == nil {
			break
		}
	}
	return nil
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
		if err := g.traverseNode(svc, nodeID); err != nil {
			return err
		}
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

			if err := g.addPolicyAttachments(svc, policyID, policyName); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *OrganizationGenerator) addPolicyAttachments(svc organizationClient, policyID, policyName string) error {
	var nextToken *string
	for {
		targetsForPolicy, err := svc.ListTargetsForPolicy(context.TODO(),
			&organizations.ListTargetsForPolicyInput{
				PolicyId:  aws.String(policyID),
				NextToken: nextToken,
			})
		if err != nil {
			return fmt.Errorf("list organization targets for policy %s: %w", policyID, err)
		}
		for _, target := range targetsForPolicy.Targets {
			targetID := StringValue(target.TargetId)
			g.Resources = append(g.Resources, terraformutils.NewResource(
				targetID+":"+policyID,
				"pa-"+targetID+":"+policyName,
				"aws_organizations_policy_attachment",
				"aws",
				map[string]string{
					"policy_id": policyID,
					"target_id": targetID,
				},
				organizationAllowEmptyValues,
				map[string]interface{}{},
			))
		}
		nextToken = targetsForPolicy.NextToken
		if nextToken == nil {
			break
		}
	}
	return nil
}
