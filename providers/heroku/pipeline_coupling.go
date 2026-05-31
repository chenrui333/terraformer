// SPDX-License-Identifier: Apache-2.0

package heroku

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	heroku "github.com/heroku/heroku-go/v6"
)

type PipelineCouplingGenerator struct {
	HerokuService
}

func (g PipelineCouplingGenerator) createResources(pipelineCouplingList []heroku.PipelineCoupling) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, pipelineCoupling := range pipelineCouplingList {
		resources = append(resources, terraformutils.NewSimpleResource(
			pipelineCoupling.ID,
			pipelineCoupling.ID,
			"heroku_pipeline_coupling",
			"heroku",
			[]string{}))
	}
	return resources
}

func (g *PipelineCouplingGenerator) InitResources() error {
	svc := g.generateService()
	output, err := svc.PipelineCouplingList(context.TODO(), &heroku.ListRange{Field: "id"})
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
