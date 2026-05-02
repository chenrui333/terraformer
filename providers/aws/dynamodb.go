// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var dynamodbAllowEmptyValues = []string{"tags."}

type DynamoDbGenerator struct {
	AWSService
}

type dynamodbOptionalResourceLoader struct {
	name string
	load func() error
}

type dynamodbTableReference struct {
	name      string
	tableARN  string
	streamARN string
}

func (g *DynamoDbGenerator) loadOptionalResources(loaders []dynamodbOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			log.Printf("Skipping DynamoDB %s: %v", loader.name, err)
		}
	}
}

func (g *DynamoDbGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := dynamodb.NewFromConfig(config)

	tables, err := g.loadTables(svc)
	if err != nil {
		return err
	}

	loaders := []dynamodbOptionalResourceLoader{
		{name: "Kinesis streaming destinations", load: func() error { return g.loadKinesisStreamingDestinations(svc, tables) }},
		{name: "resource policies", load: func() error { return g.loadResourcePolicies(svc, tables) }},
		{name: "table exports", load: func() error { return g.loadTableExports(svc) }},
	}
	if account, err := g.getAccountNumber(config); err != nil {
		log.Printf("Skipping DynamoDB contributor insights: unable to get account ID: %v", err)
	} else if accountID := StringValue(account); accountID != "" {
		loaders = append(loaders, dynamodbOptionalResourceLoader{
			name: "contributor insights",
			load: func() error { return g.loadContributorInsights(svc, accountID) },
		})
	}
	g.loadOptionalResources(loaders)

	return nil
}

func (g *DynamoDbGenerator) loadTables(svc *dynamodb.Client) ([]dynamodbTableReference, error) {
	var tables []dynamodbTableReference
	p := dynamodb.NewListTablesPaginator(svc, &dynamodb.ListTablesInput{})
	for p.HasMorePages() {
		page, e := p.NextPage(context.TODO())
		if e != nil {
			return tables, e
		}
		for _, tableName := range page.TableNames {
			if tableName == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				tableName,
				tableName,
				"aws_dynamodb_table",
				"aws",
				dynamodbAllowEmptyValues,
			))
			tables = append(tables, g.tableReference(svc, tableName))
		}
	}
	return tables, nil
}

func (g *DynamoDbGenerator) tableReference(svc *dynamodb.Client, tableName string) dynamodbTableReference {
	table := dynamodbTableReference{name: tableName}
	output, err := svc.DescribeTable(context.TODO(), &dynamodb.DescribeTableInput{TableName: &tableName})
	if err != nil {
		log.Printf("Skipping DynamoDB table metadata for %s: %v", tableName, err)
		return table
	}
	if output == nil || output.Table == nil {
		return table
	}
	table.tableARN = StringValue(output.Table.TableArn)
	table.streamARN = StringValue(output.Table.LatestStreamArn)
	return table
}

func (g *DynamoDbGenerator) loadContributorInsights(svc *dynamodb.Client, accountID string) error {
	p := dynamodb.NewListContributorInsightsPaginator(svc, &dynamodb.ListContributorInsightsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, summary := range page.ContributorInsightsSummaries {
			tableName := StringValue(summary.TableName)
			indexName := StringValue(summary.IndexName)
			if tableName == "" || !dynamodbContributorInsightsImportable(summary) {
				continue
			}
			attributes := map[string]string{"table_name": tableName}
			if indexName != "" {
				attributes["index_name"] = indexName
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				dynamodbContributorInsightsImportID(tableName, indexName, accountID),
				dynamodbResourceName(tableName, indexName, "contributor-insights"),
				"aws_dynamodb_contributor_insights",
				"aws",
				attributes,
				dynamodbAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *DynamoDbGenerator) loadKinesisStreamingDestinations(svc *dynamodb.Client, tables []dynamodbTableReference) error {
	for _, table := range tables {
		output, err := svc.DescribeKinesisStreamingDestination(context.TODO(), &dynamodb.DescribeKinesisStreamingDestinationInput{
			TableName: &table.name,
		})
		if err != nil {
			log.Printf("Skipping DynamoDB Kinesis streaming destinations for %s: %v", table.name, err)
			continue
		}
		if output == nil {
			continue
		}
		for _, destination := range output.KinesisDataStreamDestinations {
			streamARN := StringValue(destination.StreamArn)
			if streamARN == "" || !dynamodbKinesisStreamingDestinationImportable(destination) {
				continue
			}
			attributes := map[string]string{
				"stream_arn": streamARN,
				"table_name": table.name,
			}
			if precision := string(destination.ApproximateCreationDateTimePrecision); precision != "" {
				attributes["approximate_creation_date_time_precision"] = precision
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				dynamodbKinesisStreamingDestinationImportID(table.name, streamARN),
				dynamodbResourceName(table.name, arnLastSegment(streamARN, "/")),
				"aws_dynamodb_kinesis_streaming_destination",
				"aws",
				attributes,
				dynamodbAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *DynamoDbGenerator) loadResourcePolicies(svc *dynamodb.Client, tables []dynamodbTableReference) error {
	for _, table := range tables {
		for _, resource := range table.resourcePolicyTargets() {
			output, err := svc.GetResourcePolicy(context.TODO(), &dynamodb.GetResourcePolicyInput{ResourceArn: &resource.arn})
			if err != nil {
				if dynamodbResourcePolicyMissing(err) {
					continue
				}
				log.Printf("Skipping DynamoDB resource policy for %s: %v", resource.name, err)
				continue
			}
			if output == nil {
				continue
			}
			policy := StringValue(output.Policy)
			if policy == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				resource.arn,
				dynamodbResourceName(resource.name, "policy"),
				"aws_dynamodb_resource_policy",
				"aws",
				map[string]string{
					"policy":       policy,
					"resource_arn": resource.arn,
				},
				dynamodbAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

type dynamodbResourcePolicyTarget struct {
	name string
	arn  string
}

func (table dynamodbTableReference) resourcePolicyTargets() []dynamodbResourcePolicyTarget {
	var targets []dynamodbResourcePolicyTarget
	if table.tableARN != "" {
		targets = append(targets, dynamodbResourcePolicyTarget{name: table.name, arn: table.tableARN})
	}
	if table.streamARN != "" {
		targets = append(targets, dynamodbResourcePolicyTarget{name: dynamodbResourceName(table.name, "stream"), arn: table.streamARN})
	}
	return targets
}

func (g *DynamoDbGenerator) loadTableExports(svc *dynamodb.Client) error {
	p := dynamodb.NewListExportsPaginator(svc, &dynamodb.ListExportsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, export := range page.ExportSummaries {
			exportARN := StringValue(export.ExportArn)
			if exportARN == "" || !dynamodbTableExportImportable(export) {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				exportARN,
				dynamodbResourceName(arnLastSegment(exportARN, "/")),
				"aws_dynamodb_table_export",
				"aws",
				dynamodbAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *DynamoDbGenerator) PostConvertHook() error {
	for _, r := range g.Resources {
		switch r.InstanceInfo.Type {
		case "aws_dynamodb_table":
			if val, ok := r.InstanceState.Attributes["ttl.0.enabled"]; ok && val == "false" {
				delete(r.Item, "ttl")
			}
		case "aws_dynamodb_resource_policy":
			if val, ok := r.Item["policy"]; ok {
				policy := g.escapeAwsInterpolation(val.(string))
				r.Item["policy"] = fmt.Sprintf("<<POLICY\n%s\nPOLICY", policy)
			}
		default:
			continue
		}
	}
	return nil
}

func dynamodbContributorInsightsImportID(tableName, indexName, accountID string) string {
	return fmt.Sprintf("name:%s/index:%s/%s", tableName, indexName, accountID)
}

func dynamodbKinesisStreamingDestinationImportID(tableName, streamARN string) string {
	return strings.Join([]string{tableName, streamARN}, ",")
}

func dynamodbResourceName(parts ...string) string {
	var cleanParts []string
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}
	if len(cleanParts) == 0 {
		return "dynamodb_resource"
	}
	return strings.Join(cleanParts, "/")
}

func dynamodbContributorInsightsImportable(summary dynamodbtypes.ContributorInsightsSummary) bool {
	return summary.ContributorInsightsStatus == dynamodbtypes.ContributorInsightsStatusEnabled
}

func dynamodbKinesisStreamingDestinationImportable(destination dynamodbtypes.KinesisDataStreamDestination) bool {
	return destination.DestinationStatus == dynamodbtypes.DestinationStatusActive
}

func dynamodbTableExportImportable(export dynamodbtypes.ExportSummary) bool {
	return export.ExportStatus != ""
}

func dynamodbResourcePolicyMissing(err error) bool {
	var policyNotFound *dynamodbtypes.PolicyNotFoundException
	var resourceNotFound *dynamodbtypes.ResourceNotFoundException
	return errors.As(err, &policyNotFound) || errors.As(err, &resourceNotFound)
}
