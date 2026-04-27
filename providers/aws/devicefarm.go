// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/devicefarm"
	"github.com/chenrui333/terraformer/terraformutils"
)

var devicefarmAllowEmptyValues = []string{"tags."}

type DeviceFarmGenerator struct {
	AWSService
}

func (g *DeviceFarmGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := devicefarm.NewFromConfig(config)
	p := devicefarm.NewListProjectsPaginator(svc, &devicefarm.ListProjectsInput{})
	var resources []terraformutils.Resource
	for p.HasMorePages() {
		page, e := p.NextPage(context.TODO())
		if e != nil {
			return e
		}
		for _, project := range page.Projects {
			projectArn := StringValue(project.Arn)
			projectName := StringValue(project.Name)
			resources = append(resources, terraformutils.NewSimpleResource(
				projectArn,
				projectName,
				"aws_devicefarm_project",
				"aws",
				devicefarmAllowEmptyValues))
		}
	}
	g.Resources = resources
	return nil
}
