// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/chenrui333/terraformer/terraformutils"
)

var ElbAllowEmptyValues = []string{"tags."}

type ElbGenerator struct {
	AWSService
}

// Generate TerraformResources from AWS API,
// from each ELB create 1 TerraformResource.
// Need only ELB name as ID for terraform resource
// AWS api support paging
func (g *ElbGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := elasticloadbalancing.NewFromConfig(config)
	p := elasticloadbalancing.NewDescribeLoadBalancersPaginator(svc, &elasticloadbalancing.DescribeLoadBalancersInput{})
	for p.HasMorePages() {
		page, e := p.NextPage(context.TODO())
		if e != nil {
			return e
		}
		for _, loadBalancer := range page.LoadBalancerDescriptions {
			resourceName := StringValue(loadBalancer.LoadBalancerName)
			resource := terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_elb",
				"aws",
				ElbAllowEmptyValues,
			)
			resource.IgnoreKeys = append(resource.IgnoreKeys, "^instances\\.(.*)") // don't import current connect instances to ELB
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}
