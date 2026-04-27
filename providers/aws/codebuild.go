// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	"github.com/chenrui333/terraformer/terraformutils"
)

var codebuildAllowEmptyValues = []string{"tags."}

type CodeBuildGenerator struct {
	AWSService
}

func (g *CodeBuildGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := codebuild.NewFromConfig(config)
	p := codebuild.NewListProjectsPaginator(svc, &codebuild.ListProjectsInput{})
	for p.HasMorePages() {
		page, e := p.NextPage(context.TODO())
		if e != nil {
			return e
		}
		for _, project := range page.Projects {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				project,
				project,
				"aws_codebuild_project",
				"aws",
				codebuildAllowEmptyValues))
		}
	}
	return nil
}

func (g *CodeBuildGenerator) PostConvertHook() error {
	for _, r := range g.Resources {
		if r.InstanceInfo.Type != "aws_codebuild_project" {
			continue
		}
		if r.InstanceState.Attributes["concurrent_build_limit"] == "0" {
			delete(r.Item, "concurrent_build_limit")
		}
	}
	return nil
}
