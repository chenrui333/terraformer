// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloud9"
	"github.com/aws/aws-sdk-go-v2/service/cloud9/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var cloud9AllowEmptyValues = []string{"tags."}

type Cloud9Generator struct {
	AWSService
}

func (g *Cloud9Generator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := cloud9.NewFromConfig(config)
	output, err := svc.ListEnvironments(context.TODO(), &cloud9.ListEnvironmentsInput{})
	if err != nil {
		return err
	}
	for _, environmentID := range output.EnvironmentIds {
		details, _ := svc.DescribeEnvironmentStatus(context.TODO(), &cloud9.DescribeEnvironmentStatusInput{
			EnvironmentId: &environmentID,
		})
		if details.Status == types.EnvironmentStatusError ||
			details.Status == types.EnvironmentStatusDeleting {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			environmentID,
			environmentID,
			"aws_cloud9_environment_ec2",
			"aws",
			cloud9AllowEmptyValues))
	}
	return nil
}
