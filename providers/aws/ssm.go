// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

var ssmAllowEmptyValues = []string{"tags."}

type SsmGenerator struct {
	AWSService
}

func (g *SsmGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := ssm.NewFromConfig(config)
	p := ssm.NewDescribeParametersPaginator(svc, &ssm.DescribeParametersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, parameter := range page.Parameters {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				StringValue(parameter.Name),
				StringValue(parameter.Name),
				"aws_ssm_parameter",
				"aws",
				ssmAllowEmptyValues,
			))
		}
	}

	return nil
}
