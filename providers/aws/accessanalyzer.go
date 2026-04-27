// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/accessanalyzer"
	"github.com/chenrui333/terraformer/terraformutils"
)

var accessanalyzerAllowEmptyValues = []string{"tags."}

type AccessAnalyzerGenerator struct {
	AWSService
}

func (g *AccessAnalyzerGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := accessanalyzer.NewFromConfig(config)
	p := accessanalyzer.NewListAnalyzersPaginator(svc, &accessanalyzer.ListAnalyzersInput{})
	var resources []terraformutils.Resource
	for p.HasMorePages() {
		page, e := p.NextPage(context.TODO())
		if e != nil {
			return e
		}
		for _, analyzer := range page.Analyzers {
			resourceName := *analyzer.Name
			resources = append(resources, terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_accessanalyzer_analyzer",
				"aws",
				accessanalyzerAllowEmptyValues))
		}
	}
	g.Resources = resources
	return nil
}
