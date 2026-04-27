// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var cloudFormationAllowEmptyValues = []string{"tags."}

type CloudFormationGenerator struct {
	AWSService
}

func (g *CloudFormationGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := cloudformation.NewFromConfig(config)
	p := cloudformation.NewListStacksPaginator(svc, &cloudformation.ListStacksInput{})
	for p.HasMorePages() {
		page, e := p.NextPage(context.TODO())
		if e != nil {
			return e
		}
		for _, stackSummary := range page.StackSummaries {
			if stackSummary.StackStatus == types.StackStatusDeleteComplete {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				*stackSummary.StackId,
				*stackSummary.StackName,
				"aws_cloudformation_stack",
				"aws",
				cloudFormationAllowEmptyValues,
			))
		}
	}
	stackSets, err := svc.ListStackSets(context.TODO(), &cloudformation.ListStackSetsInput{})
	if err != nil {
		return err
	}
	for _, stackSetSummary := range stackSets.Summaries {
		if stackSetSummary.Status == types.StackSetStatusDeleted {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			*stackSetSummary.StackSetId,
			*stackSetSummary.StackSetName,
			"aws_cloudformation_stack_set",
			"aws",
			cloudFormationAllowEmptyValues,
		))

		stackSetInstances, err := svc.ListStackInstances(context.TODO(), &cloudformation.ListStackInstancesInput{
			StackSetName: stackSetSummary.StackSetName,
		})
		if err != nil {
			return err
		}
		for _, stackSetI := range stackSetInstances.Summaries {
			id := StringValue(stackSetI.StackSetId) + "," + StringValue(stackSetI.Account) + "," + StringValue(stackSetI.Region)

			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				id,
				id,
				"aws_cloudformation_stack_set_instance",
				"aws",
				cloudFormationAllowEmptyValues,
			))
		}
	}

	return nil
}

func (g *CloudFormationGenerator) PostConvertHook() error {
	for _, resource := range g.Resources {
		if resource.InstanceInfo.Type == "aws_cloudformation_stack" {
			delete(resource.Item, "outputs")
			if templateBody, ok := resource.InstanceState.Attributes["template_body"]; ok {
				resource.Item["template_body"] = g.escapeAwsInterpolation(templateBody)
			}
		}
	}
	return nil
}
