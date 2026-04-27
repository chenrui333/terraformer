// SPDX-License-Identifier: Apache-2.0

package heroku

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	heroku "github.com/heroku/heroku-go/v5"
)

type PipelineGenerator struct {
	HerokuService
}

func (g PipelineGenerator) createResources(pipelineList []heroku.Pipeline) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, pipeline := range pipelineList {
		resources = append(resources, terraformutils.NewSimpleResource(
			pipeline.ID,
			pipeline.Name,
			"heroku_pipeline",
			"heroku",
			[]string{}))
	}
	return resources
}

func (g *PipelineGenerator) InitResources() error {
	svc := g.generateService()
	output, err := svc.PipelineList(context.TODO(), &heroku.ListRange{Field: "id"})
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
