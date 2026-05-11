// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/detective"
	detectivetypes "github.com/aws/aws-sdk-go-v2/service/detective/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var detectiveAllowEmptyValues = []string{"tags."}

const (
	detectiveGraphResourceType                    = "aws_detective_graph"
	detectiveMemberResourceType                   = "aws_detective_member"
	detectiveOrganizationAdminAccountResourceType = "aws_detective_organization_admin_account"
	detectiveMemberResourceIDSeparator            = "/"
	detectiveResourceNameSeparator                = ":"
)

type DetectiveGenerator struct {
	AWSService
}

func (g *DetectiveGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := detective.NewFromConfig(config)

	if err := g.addGraphs(svc); err != nil {
		return err
	}
	if err := g.addOrganizationAdminAccounts(svc); err != nil {
		if !detectiveOptionalResourceUnavailable(err) {
			return err
		}
	}
	return nil
}

func (g *DetectiveGenerator) addGraphs(svc *detective.Client) error {
	p := detective.NewListGraphsPaginator(svc, &detective.ListGraphsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, graph := range page.GraphList {
			graphARN := StringValue(graph.Arn)
			if graphARN == "" {
				continue
			}
			g.Resources = append(g.Resources, newDetectiveGraphResource(graphARN))
			if err := g.addMembers(svc, graphARN); err != nil {
				if detectiveOptionalResourceUnavailable(err) {
					continue
				}
				return err
			}
		}
	}
	return nil
}

func (g *DetectiveGenerator) addMembers(svc *detective.Client, graphARN string) error {
	p := detective.NewListMembersPaginator(svc, &detective.ListMembersInput{
		GraphArn: &graphARN,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, member := range page.MemberDetails {
			resource, ok := newDetectiveMemberResource(graphARN, member)
			if !ok {
				continue
			}
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *DetectiveGenerator) addOrganizationAdminAccounts(svc *detective.Client) error {
	p := detective.NewListOrganizationAdminAccountsPaginator(svc, &detective.ListOrganizationAdminAccountsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, administrator := range page.Administrators {
			accountID := StringValue(administrator.AccountId)
			if accountID == "" {
				continue
			}
			g.Resources = append(g.Resources, newDetectiveOrganizationAdminAccountResource(accountID))
		}
	}
	return nil
}

func newDetectiveGraphResource(graphARN string) terraformutils.Resource {
	return terraformutils.NewResource(
		graphARN,
		detectiveResourceName("graph", graphARN),
		detectiveGraphResourceType,
		"aws",
		map[string]string{
			"graph_arn": graphARN,
		},
		detectiveAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newDetectiveMemberResource(graphARN string, member detectivetypes.MemberDetail) (terraformutils.Resource, bool) {
	accountID := StringValue(member.AccountId)
	emailAddress := StringValue(member.EmailAddress)
	if graphARN == "" || accountID == "" || emailAddress == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		detectiveMemberResourceID(graphARN, accountID),
		detectiveResourceName("member", graphARN, accountID),
		detectiveMemberResourceType,
		"aws",
		map[string]string{
			"account_id":    accountID,
			"email_address": emailAddress,
			"graph_arn":     graphARN,
		},
		detectiveAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newDetectiveOrganizationAdminAccountResource(accountID string) terraformutils.Resource {
	return terraformutils.NewResource(
		accountID,
		detectiveResourceName("organization_admin_account", accountID),
		detectiveOrganizationAdminAccountResourceType,
		"aws",
		map[string]string{
			"account_id": accountID,
		},
		detectiveAllowEmptyValues,
		map[string]interface{}{},
	)
}

func detectiveMemberResourceID(graphARN, accountID string) string {
	return strings.Join([]string{graphARN, accountID}, detectiveMemberResourceIDSeparator)
}

func detectiveResourceName(parts ...string) string {
	return strings.Join(parts, detectiveResourceNameSeparator)
}

func detectiveOptionalResourceUnavailable(err error) bool {
	if err == nil {
		return false
	}
	var notFound *detectivetypes.ResourceNotFoundException
	if errors.As(err, &notFound) {
		return true
	}
	var accessDenied *detectivetypes.AccessDeniedException
	if errors.As(err, &accessDenied) {
		return detectiveAdminUnavailableMessage(accessDenied.ErrorMessage())
	}
	var validation *detectivetypes.ValidationException
	if errors.As(err, &validation) {
		return detectiveAdminUnavailableMessage(validation.ErrorMessage())
	}
	return false
}

func detectiveAdminUnavailableMessage(message string) bool {
	message = strings.ToLower(message)
	return strings.Contains(message, "not a member of an organization") ||
		strings.Contains(message, "not an administrator") ||
		strings.Contains(message, "not the administrator") ||
		strings.Contains(message, "delegated administrator")
}
