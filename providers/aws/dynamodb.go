// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/chenrui333/terraformer/terraformutils"
)

var dynamodbAllowEmptyValues = []string{"tags."}

type DynamoDbGenerator struct {
	AWSService
}

func (g *DynamoDbGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := dynamodb.NewFromConfig(config)
	p := dynamodb.NewListTablesPaginator(svc, &dynamodb.ListTablesInput{})
	for p.HasMorePages() {
		page, e := p.NextPage(context.TODO())
		if e != nil {
			return e
		}
		for _, tableName := range page.TableNames {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				tableName,
				tableName,
				"aws_dynamodb_table",
				"aws",
				dynamodbAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *DynamoDbGenerator) PostConvertHook() error {
	for _, r := range g.Resources {
		if r.InstanceInfo.Type != "aws_dynamodb_table" {
			continue
		}
		if val, ok := r.InstanceState.Attributes["ttl.0.enabled"]; ok && val == "false" {
			delete(r.Item, "ttl")
		}
	}
	return nil
}
