// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/workspaces"
	"github.com/chenrui333/terraformer/terraformutils"
)

var workspacesAllowEmptyValues = []string{"tags."}

type WorkspacesGenerator struct {
	AWSService
}

func (g *WorkspacesGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := workspaces.NewFromConfig(config)
	if err := g.loadWorkspaces(svc); err != nil {
		return err
	}
	if err := g.loadWorkspacesIPGroup(svc); err != nil {
		return err
	}
	return nil
}

func (g *WorkspacesGenerator) loadWorkspaces(svc *workspaces.Client) error {
	p := workspaces.NewDescribeWorkspacesPaginator(svc, &workspaces.DescribeWorkspacesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, workspace := range page.Workspaces {
			directoryID := StringValue(workspace.DirectoryId)
			workspaceID := StringValue(workspace.WorkspaceId)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				directoryID,
				directoryID,
				"aws_workspaces_directory",
				"aws",
				workspacesAllowEmptyValues))
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				workspaceID,
				workspaceID,
				"aws_workspaces_workspace",
				"aws",
				workspacesAllowEmptyValues))
		}
	}
	return nil
}

func (g *WorkspacesGenerator) loadWorkspacesIPGroup(svc *workspaces.Client) error {
	var nextToken *string
	for {
		response, err := svc.DescribeIpGroups(context.TODO(), &workspaces.DescribeIpGroupsInput{NextToken: nextToken})
		if err != nil {
			return err
		}
		for _, ipGroup := range response.Result {
			groupID := StringValue(ipGroup.GroupId)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				groupID,
				groupID,
				"aws_workspaces_ip_group",
				"aws",
				workspacesAllowEmptyValues))
		}
		nextToken = response.NextToken
		if !awsHasMorePages(nextToken) {
			break
		}
	}
	return nil
}
