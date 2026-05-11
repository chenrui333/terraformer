// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appstream"
	appstreamtypes "github.com/aws/aws-sdk-go-v2/service/appstream/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	appStreamFleetResourceType                 = "aws_appstream_fleet"
	appStreamStackResourceType                 = "aws_appstream_stack"
	appStreamFleetStackAssociationResourceType = "aws_appstream_fleet_stack_association"
	appStreamUserResourceType                  = "aws_appstream_user"
	appStreamUserStackAssociationResourceType  = "aws_appstream_user_stack_association"

	appStreamFleetStackAssociationIDSeparator = "/"
	appStreamUserIDSeparator                  = "/"
	appStreamUserStackAssociationIDSeparator  = "/"
)

var appStreamAllowEmptyValues = []string{"tags."}

type AppStreamGenerator struct {
	AWSService
}

func (g *AppStreamGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := appstream.NewFromConfig(config)

	fleetNames, err := g.loadFleets(svc)
	if err != nil {
		return err
	}
	if err := g.loadStacks(svc); err != nil {
		return err
	}
	if err := g.loadFleetStackAssociations(svc, fleetNames); err != nil {
		return err
	}
	if err := g.loadUsers(svc); err != nil {
		return err
	}
	return g.loadUserStackAssociations(svc)
}

func (g *AppStreamGenerator) loadFleets(svc *appstream.Client) ([]string, error) {
	var fleetNames []string
	input := &appstream.DescribeFleetsInput{}
	for {
		page, err := svc.DescribeFleets(context.TODO(), input)
		if err != nil {
			return nil, err
		}
		for _, fleet := range page.Fleets {
			resource, ok := newAppStreamFleetResource(fleet)
			if !ok {
				continue
			}
			g.Resources = append(g.Resources, resource)
			fleetNames = append(fleetNames, StringValue(fleet.Name))
		}
		nextToken := appStreamNextToken(page.NextToken)
		if nextToken == nil {
			break
		}
		input.NextToken = nextToken
	}
	return fleetNames, nil
}

func (g *AppStreamGenerator) loadStacks(svc *appstream.Client) error {
	input := &appstream.DescribeStacksInput{}
	for {
		page, err := svc.DescribeStacks(context.TODO(), input)
		if err != nil {
			return err
		}
		for _, stack := range page.Stacks {
			if resource, ok := newAppStreamStackResource(stack); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
		nextToken := appStreamNextToken(page.NextToken)
		if nextToken == nil {
			break
		}
		input.NextToken = nextToken
	}
	return nil
}

func (g *AppStreamGenerator) loadFleetStackAssociations(svc *appstream.Client, fleetNames []string) error {
	for _, fleetName := range fleetNames {
		if err := g.loadFleetStackAssociationsForFleet(svc, fleetName); err != nil {
			return err
		}
	}
	return nil
}

func (g *AppStreamGenerator) loadFleetStackAssociationsForFleet(svc *appstream.Client, fleetName string) error {
	if fleetName == "" {
		return nil
	}
	input := &appstream.ListAssociatedStacksInput{FleetName: aws.String(fleetName)}
	for {
		page, err := svc.ListAssociatedStacks(context.TODO(), input)
		if appStreamResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, stackName := range page.Names {
			if resource, ok := newAppStreamFleetStackAssociationResource(fleetName, stackName); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
		nextToken := appStreamNextToken(page.NextToken)
		if nextToken == nil {
			break
		}
		input.NextToken = nextToken
	}
	return nil
}

func (g *AppStreamGenerator) loadUsers(svc *appstream.Client) error {
	input := &appstream.DescribeUsersInput{
		AuthenticationType: appstreamtypes.AuthenticationTypeUserpool,
	}
	for {
		page, err := svc.DescribeUsers(context.TODO(), input)
		if appStreamResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, user := range page.Users {
			if resource, ok := newAppStreamUserResource(user); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
		nextToken := appStreamNextToken(page.NextToken)
		if nextToken == nil {
			break
		}
		input.NextToken = nextToken
	}
	return nil
}

func (g *AppStreamGenerator) loadUserStackAssociations(svc *appstream.Client) error {
	input := &appstream.DescribeUserStackAssociationsInput{
		AuthenticationType: appstreamtypes.AuthenticationTypeUserpool,
	}
	for {
		page, err := svc.DescribeUserStackAssociations(context.TODO(), input)
		if appStreamResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, association := range page.UserStackAssociations {
			if resource, ok := newAppStreamUserStackAssociationResource(association); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
		nextToken := appStreamNextToken(page.NextToken)
		if nextToken == nil {
			break
		}
		input.NextToken = nextToken
	}
	return nil
}

func newAppStreamFleetResource(fleet appstreamtypes.Fleet) (terraformutils.Resource, bool) {
	name := StringValue(fleet.Name)
	if name == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		name,
		appStreamResourceName("fleet", name),
		appStreamFleetResourceType,
		"aws",
		appStreamAllowEmptyValues,
	), true
}

func newAppStreamStackResource(stack appstreamtypes.Stack) (terraformutils.Resource, bool) {
	name := StringValue(stack.Name)
	if name == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		name,
		appStreamResourceName("stack", name),
		appStreamStackResourceType,
		"aws",
		appStreamAllowEmptyValues,
	), true
}

func newAppStreamFleetStackAssociationResource(fleetName, stackName string) (terraformutils.Resource, bool) {
	if fleetName == "" || stackName == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		appStreamFleetStackAssociationImportID(fleetName, stackName),
		appStreamResourceName("fleet-stack-association", fleetName, stackName),
		appStreamFleetStackAssociationResourceType,
		"aws",
		appStreamAllowEmptyValues,
	), true
}

func newAppStreamUserResource(user appstreamtypes.User) (terraformutils.Resource, bool) {
	userName := StringValue(user.UserName)
	authType := user.AuthenticationType
	if userName == "" || authType == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		appStreamUserImportID(userName, authType),
		appStreamResourceName("user", string(authType), userName),
		appStreamUserResourceType,
		"aws",
		map[string]string{
			"authentication_type": string(authType),
			"user_name":           userName,
		},
		appStreamAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newAppStreamUserStackAssociationResource(association appstreamtypes.UserStackAssociation) (terraformutils.Resource, bool) {
	userName := StringValue(association.UserName)
	stackName := StringValue(association.StackName)
	authType := association.AuthenticationType
	if userName == "" || stackName == "" || authType == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		appStreamUserStackAssociationImportID(userName, authType, stackName),
		appStreamResourceName("user-stack-association", string(authType), userName, stackName),
		appStreamUserStackAssociationResourceType,
		"aws",
		map[string]string{
			"authentication_type": string(authType),
			"stack_name":          stackName,
			"user_name":           userName,
		},
		appStreamAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func appStreamFleetStackAssociationImportID(fleetName, stackName string) string {
	return strings.Join([]string{fleetName, stackName}, appStreamFleetStackAssociationIDSeparator)
}

func appStreamUserImportID(userName string, authType appstreamtypes.AuthenticationType) string {
	return strings.Join([]string{userName, string(authType)}, appStreamUserIDSeparator)
}

func appStreamUserStackAssociationImportID(userName string, authType appstreamtypes.AuthenticationType, stackName string) string {
	return strings.Join([]string{userName, string(authType), stackName}, appStreamUserStackAssociationIDSeparator)
}

func appStreamResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d/%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return "appstream-resource"
	}
	return strings.Join(cleanParts, "/")
}

func appStreamNextToken(token *string) *string {
	if StringValue(token) == "" {
		return nil
	}
	return token
}

func appStreamResourceNotFound(err error) bool {
	var notFound *appstreamtypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
