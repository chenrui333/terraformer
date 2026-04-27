// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/codedeploy"
	"github.com/chenrui333/terraformer/terraformutils"
)

var codedeployAllowEmptyValues = []string{"tags."}

type CodeDeployGenerator struct {
	AWSService
}

func (g *CodeDeployGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := codedeploy.NewFromConfig(config)
	p := codedeploy.NewListApplicationsPaginator(svc, &codedeploy.ListApplicationsInput{})
	var resources []terraformutils.Resource
	for p.HasMorePages() {
		page, e := p.NextPage(context.TODO())
		if e != nil {
			return e
		}
		for _, application := range page.Applications {
			resources = append(resources, terraformutils.NewSimpleResource(
				fmt.Sprintf(":%s", application),
				application,
				"aws_codedeploy_app",
				"aws",
				codedeployAllowEmptyValues))
		}
	}
	g.Resources = resources
	return nil
}
