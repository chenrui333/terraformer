// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var cloudtrailAllowEmptyValues = []string{"tags."}

type CloudTrailGenerator struct {
	AWSService
}

func (g *CloudTrailGenerator) createResources(trailList []types.Trail) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, trail := range trailList {
		resourceName := StringValue(trail.Name)
		resources = append(resources, terraformutils.NewSimpleResource(
			resourceName,
			resourceName,
			"aws_cloudtrail",
			"aws",
			cloudtrailAllowEmptyValues))
	}
	return resources
}

func (g *CloudTrailGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := cloudtrail.NewFromConfig(config)
	output, err := svc.DescribeTrails(context.TODO(), &cloudtrail.DescribeTrailsInput{})
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output.TrailList)
	return nil
}
