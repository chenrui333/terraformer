// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/codepipeline"
	"github.com/chenrui333/terraformer/terraformutils"
)

var codepipelineAllowEmptyValues = []string{"tags."}

type CodePipelineGenerator struct {
	AWSService
}

func (g *CodePipelineGenerator) loadPipelines(svc *codepipeline.Client) error {
	p := codepipeline.NewListPipelinesPaginator(svc, &codepipeline.ListPipelinesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, pipeline := range page.Pipelines {
			resourceName := StringValue(pipeline.Name)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_codepipeline",
				"aws",
				codepipelineAllowEmptyValues))
		}
	}
	return nil
}

func (g *CodePipelineGenerator) loadWebhooks(svc *codepipeline.Client) error {
	p := codepipeline.NewListWebhooksPaginator(svc, &codepipeline.ListWebhooksInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, webhook := range page.Webhooks {
			resourceArn := StringValue(webhook.Arn)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				resourceArn,
				resourceArn,
				"aws_codepipeline_webhook",
				"aws",
				codepipelineAllowEmptyValues))
		}
	}
	return nil
}

func (g *CodePipelineGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := codepipeline.NewFromConfig(config)

	if err := g.loadPipelines(svc); err != nil {
		return err
	}
	if err := g.loadWebhooks(svc); err != nil {
		return err
	}

	return nil
}
