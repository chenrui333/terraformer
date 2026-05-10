// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/pipes"
	"github.com/chenrui333/terraformer/terraformutils"
)

var pipesAllowEmptyValues = []string{"tags."}

type PipesGenerator struct {
	AWSService
}

func (g *PipesGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := pipes.NewFromConfig(config)

	return g.loadPipes(svc)
}

func (g *PipesGenerator) loadPipes(svc *pipes.Client) error {
	p := pipes.NewListPipesPaginator(svc, &pipes.ListPipesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, pipe := range page.Pipes {
			if resource, ok := newPipesPipeResource(StringValue(pipe.Name)); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newPipesPipeResource(pipeName string) (terraformutils.Resource, bool) {
	if pipeName == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		pipeName,
		pipeName,
		"aws_pipes_pipe",
		"aws",
		pipesAllowEmptyValues), true
}
