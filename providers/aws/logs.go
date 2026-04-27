// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/chenrui333/terraformer/terraformutils"
)

var logsAllowEmptyValues = []string{"tags."}

type LogsGenerator struct {
	AWSService
}

func (g *LogsGenerator) createResources(logGroups *cloudwatchlogs.DescribeLogGroupsOutput) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, logGroup := range logGroups.LogGroups {
		resourceName := StringValue(logGroup.LogGroupName)

		attributes := map[string]string{}

		if logGroup.RetentionInDays != nil {
			attributes["retention_in_days"] = strconv.FormatInt(int64(*logGroup.RetentionInDays), 10)
		}

		if logGroup.KmsKeyId != nil {
			attributes["kms_key_id"] = *logGroup.KmsKeyId
		}

		resources = append(resources, terraformutils.NewResource(
			resourceName,
			resourceName,
			"aws_cloudwatch_log_group",
			"aws",
			attributes,
			logsAllowEmptyValues,
			map[string]interface{}{}))
	}
	return resources
}

// Generate TerraformResources from AWS API
func (g *LogsGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := cloudwatchlogs.NewFromConfig(config)

	p := cloudwatchlogs.NewDescribeLogGroupsPaginator(svc, &cloudwatchlogs.DescribeLogGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, g.createResources(page)...)
	}
	return nil
}

// remove retention_in_days if it is 0 (it gets added by the "refresh" stage)
func (g *LogsGenerator) PostConvertHook() error {
	for _, resource := range g.Resources {
		if resource.Item["retention_in_days"] == "0" {
			delete(resource.Item, "retention_in_days")
		}
	}
	return nil
}
