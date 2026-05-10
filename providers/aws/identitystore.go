// SPDX-License-Identifier: Apache-2.0

//nolint:revive // lint triage: legacy provider/API/security baseline is tracked in #175.
package aws

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/identitystore"
	identitystoretypes "github.com/aws/aws-sdk-go-v2/service/identitystore/types"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
)

var identityStoreAllowEmptyValues = []string{"tags."}

const (
	identityStoreGroupResourceType           = "aws_identitystore_group"
	identityStoreGroupMembershipResourceType = "aws_identitystore_group_membership"
	identityStoreUserResourceType            = "aws_identitystore_user"
	identityStoreResourceIDSeparator         = "/"
	identityStoreResourceNameSeparator       = ":"
)

type IdentityStoreGenerator struct {
	AWSService
}

func (g *IdentityStoreGenerator) GetIdentityStoreIds() ([]string, error) {
	config, e := g.generateConfig()
	if e != nil {
		return nil, e
	}
	svc := ssoadmin.NewFromConfig(config)
	instances, err := listSSOAdminInstances(svc)
	if err != nil {
		return nil, err
	}
	var identityStoreIds []string
	for _, instance := range instances {
		identityStoreId := StringValue(instance.IdentityStoreId)
		if identityStoreId != "" {
			identityStoreIds = append(identityStoreIds, identityStoreId)
		}
	}
	return identityStoreIds, nil
}

func (g *IdentityStoreGenerator) InitGroupResources(identityStoreId string) error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := identitystore.NewFromConfig(config)
	p := identitystore.NewListGroupsPaginator(svc, &identitystore.ListGroupsInput{
		IdentityStoreId: aws.String(identityStoreId),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, group := range page.Groups {
			groupId := StringValue(group.GroupId)
			if groupId == "" {
				continue
			}
			resourceStart := len(g.Resources)
			g.Resources = append(g.Resources, newIdentityStoreGroupResource(identityStoreId, group))
			err = g.InitGroupMembershipResources(identityStoreId, groupId)
			if err != nil {
				if identityStoreResourceNotFound(err) {
					g.Resources = g.Resources[:resourceStart]
					continue
				}
				return err
			}
		}
	}
	return nil
}

func (g *IdentityStoreGenerator) InitGroupMembershipResources(identityStoreId string, groupId string) error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := identitystore.NewFromConfig(config)
	p := identitystore.NewListGroupMembershipsPaginator(svc, &identitystore.ListGroupMembershipsInput{
		GroupId:         aws.String(groupId),
		IdentityStoreId: aws.String(identityStoreId),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, user := range page.GroupMemberships {
			var memberId string
			switch v := user.MemberId.(type) {
			case *identitystoretypes.MemberIdMemberUserId:
				memberId = v.Value // Value is string
			default:
				continue
			}
			membershipId := StringValue(user.MembershipId)
			if membershipId == "" || memberId == "" {
				continue
			}
			g.Resources = append(g.Resources, newIdentityStoreGroupMembershipResource(identityStoreId, groupId, memberId, membershipId))
		}
	}
	return nil
}

func (g *IdentityStoreGenerator) InitUserResources(identityStoreId string) error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := identitystore.NewFromConfig(config)
	p := identitystore.NewListUsersPaginator(svc, &identitystore.ListUsersInput{
		IdentityStoreId: aws.String(identityStoreId),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, user := range page.Users {
			userId := StringValue(user.UserId)
			if userId == "" {
				continue
			}
			g.Resources = append(g.Resources, newIdentityStoreUserResource(identityStoreId, user))
		}
	}
	return nil
}

func (g *IdentityStoreGenerator) InitResources() error {
	identityStoreIds, e := g.GetIdentityStoreIds()
	if e != nil {
		return e
	}
	if len(identityStoreIds) == 0 {
		return nil
	}

	for _, identityStoreId := range identityStoreIds {
		e = g.InitUserResources(identityStoreId)
		if e != nil {
			if identityStoreResourceNotFound(e) {
				continue
			}
			return e
		}

		e = g.InitGroupResources(identityStoreId)
		if e != nil {
			if identityStoreResourceNotFound(e) {
				continue
			}
			return e
		}
	}

	return nil
}

func newIdentityStoreGroupResource(identityStoreId string, group identitystoretypes.Group) terraformutils.Resource {
	groupId := StringValue(group.GroupId)
	attributes := map[string]string{
		"group_id":          groupId,
		"identity_store_id": identityStoreId,
	}
	if description := StringValue(group.Description); description != "" {
		attributes["description"] = description
	}
	if displayName := StringValue(group.DisplayName); displayName != "" {
		attributes["display_name"] = displayName
	}
	return terraformutils.NewResource(
		identityStoreResourceID(identityStoreId, groupId),
		identityStoreResourceName(StringValue(group.DisplayName), groupId, identityStoreId),
		identityStoreGroupResourceType,
		"aws",
		attributes,
		identityStoreAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newIdentityStoreGroupMembershipResource(identityStoreId, groupId, memberId, membershipId string) terraformutils.Resource {
	return terraformutils.NewResource(
		identityStoreResourceID(identityStoreId, membershipId),
		identityStoreResourceName(groupId, memberId, membershipId, identityStoreId),
		identityStoreGroupMembershipResourceType,
		"aws",
		map[string]string{
			"group_id":          groupId,
			"identity_store_id": identityStoreId,
			"member_id":         memberId,
			"membership_id":     membershipId,
		},
		identityStoreAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newIdentityStoreUserResource(identityStoreId string, user identitystoretypes.User) terraformutils.Resource {
	userId := StringValue(user.UserId)
	attributes := map[string]string{
		"identity_store_id": identityStoreId,
		"user_id":           userId,
	}
	if displayName := StringValue(user.DisplayName); displayName != "" {
		attributes["display_name"] = displayName
	}
	if userName := StringValue(user.UserName); userName != "" {
		attributes["user_name"] = userName
	}
	return terraformutils.NewResource(
		identityStoreResourceID(identityStoreId, userId),
		identityStoreResourceName(StringValue(user.UserName), userId, identityStoreId),
		identityStoreUserResourceType,
		"aws",
		attributes,
		identityStoreAllowEmptyValues,
		map[string]interface{}{},
	)
}

func identityStoreResourceID(identityStoreId, resourceId string) string {
	return strings.Join([]string{identityStoreId, resourceId}, identityStoreResourceIDSeparator)
}

func identityStoreResourceName(parts ...string) string {
	var nonEmptyParts []string
	for _, part := range parts {
		if part != "" {
			nonEmptyParts = append(nonEmptyParts, part)
		}
	}
	return strings.Join(nonEmptyParts, identityStoreResourceNameSeparator)
}

func identityStoreResourceNotFound(err error) bool {
	var notFound *identitystoretypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
