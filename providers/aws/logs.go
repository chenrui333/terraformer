// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var logsAllowEmptyValues = []string{"tags."}

var logsAccountPolicyTypes = []types.PolicyType{
	types.PolicyTypeDataProtectionPolicy,
	types.PolicyTypeSubscriptionFilterPolicy,
	types.PolicyTypeFieldIndexPolicy,
	types.PolicyTypeTransformerPolicy,
}

type logsOptionalResourceLoader struct {
	name string
	load func() error
}

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

	var logGroupNames []string
	p := cloudwatchlogs.NewDescribeLogGroupsPaginator(svc, &cloudwatchlogs.DescribeLogGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, g.createResources(page)...)
		for _, logGroup := range page.LogGroups {
			logGroupName := StringValue(logGroup.LogGroupName)
			if logGroupName != "" {
				logGroupNames = append(logGroupNames, logGroupName)
			}
		}
	}
	g.getOptionalLogResources(
		logsOptionalResourceLoader{name: "metric filters", load: func() error { return g.addMetricFilters(svc, logGroupNames) }},
		logsOptionalResourceLoader{name: "subscription filters", load: func() error { return g.addSubscriptionFilters(svc, logGroupNames) }},
		logsOptionalResourceLoader{name: "data protection policies", load: func() error { return g.addDataProtectionPolicies(svc, logGroupNames) }},
		logsOptionalResourceLoader{name: "destinations", load: func() error { return g.addDestinations(svc) }},
		logsOptionalResourceLoader{name: "resource policies", load: func() error { return g.addResourcePolicies(svc) }},
		logsOptionalResourceLoader{name: "account policies", load: func() error { return g.addAccountPolicies(svc) }},
		logsOptionalResourceLoader{name: "query definitions", load: func() error {
			account, err := g.getAccountNumber(config)
			if err != nil {
				return err
			}
			return g.addQueryDefinitions(svc, config.Region, StringValue(account))
		}},
	)
	return nil
}

func (g *LogsGenerator) getOptionalLogResources(loaders ...logsOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			log.Printf("skipping CloudWatch Logs %s discovery: %v", loader.name, err)
		}
	}
}

func (g *LogsGenerator) addMetricFilters(svc *cloudwatchlogs.Client, logGroupNames []string) error {
	for _, logGroupName := range logGroupNames {
		p := cloudwatchlogs.NewDescribeMetricFiltersPaginator(svc, &cloudwatchlogs.DescribeMetricFiltersInput{
			LogGroupName: &logGroupName,
		})
		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if err != nil {
				return err
			}
			for _, filter := range page.MetricFilters {
				filterName := StringValue(filter.FilterName)
				if filterName == "" {
					continue
				}
				id := fmt.Sprintf("%s:%s", logGroupName, filterName)
				g.Resources = append(g.Resources, terraformutils.NewResource(
					id,
					logsResourceName(logGroupName, filterName),
					"aws_cloudwatch_log_metric_filter",
					"aws",
					map[string]string{
						"log_group_name": logGroupName,
						"name":           filterName,
					},
					logsAllowEmptyValues,
					map[string]interface{}{}))
			}
		}
	}
	return nil
}

func (g *LogsGenerator) addSubscriptionFilters(svc *cloudwatchlogs.Client, logGroupNames []string) error {
	for _, logGroupName := range logGroupNames {
		p := cloudwatchlogs.NewDescribeSubscriptionFiltersPaginator(svc, &cloudwatchlogs.DescribeSubscriptionFiltersInput{
			LogGroupName: &logGroupName,
		})
		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if err != nil {
				return err
			}
			for _, filter := range page.SubscriptionFilters {
				filterName := StringValue(filter.FilterName)
				if filterName == "" {
					continue
				}
				id := fmt.Sprintf("%s|%s", logGroupName, filterName)
				g.Resources = append(g.Resources, terraformutils.NewResource(
					id,
					logsResourceName(logGroupName, filterName),
					"aws_cloudwatch_log_subscription_filter",
					"aws",
					map[string]string{
						"log_group_name": logGroupName,
						"name":           filterName,
					},
					logsAllowEmptyValues,
					map[string]interface{}{}))
			}
		}
	}
	return nil
}

func (g *LogsGenerator) addDataProtectionPolicies(svc *cloudwatchlogs.Client, logGroupNames []string) error {
	for _, logGroupName := range logGroupNames {
		output, err := svc.GetDataProtectionPolicy(context.TODO(), &cloudwatchlogs.GetDataProtectionPolicyInput{
			LogGroupIdentifier: &logGroupName,
		})
		if err != nil {
			if logsResourceNotFound(err) {
				continue
			}
			return err
		}
		if StringValue(output.PolicyDocument) == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			logGroupName,
			logGroupName,
			"aws_cloudwatch_log_data_protection_policy",
			"aws",
			map[string]string{
				"log_group_name": logGroupName,
			},
			logsAllowEmptyValues,
			map[string]interface{}{}))
	}
	return nil
}

func (g *LogsGenerator) addDestinations(svc *cloudwatchlogs.Client) error {
	p := cloudwatchlogs.NewDescribeDestinationsPaginator(svc, &cloudwatchlogs.DescribeDestinationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, destination := range page.Destinations {
			destinationName := StringValue(destination.DestinationName)
			if destinationName == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				destinationName,
				destinationName,
				"aws_cloudwatch_log_destination",
				"aws",
				logsAllowEmptyValues))
		}
	}
	return nil
}

func (g *LogsGenerator) addResourcePolicies(svc *cloudwatchlogs.Client) error {
	var nextToken *string
	for {
		output, err := svc.DescribeResourcePolicies(context.TODO(), &cloudwatchlogs.DescribeResourcePoliciesInput{
			NextToken: nextToken,
		})
		if err != nil {
			return err
		}
		for _, policy := range output.ResourcePolicies {
			policyName := StringValue(policy.PolicyName)
			if policyName == "" || StringValue(policy.PolicyDocument) == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				policyName,
				policyName,
				"aws_cloudwatch_log_resource_policy",
				"aws",
				logsAllowEmptyValues))
		}
		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}
	return nil
}

func (g *LogsGenerator) addAccountPolicies(svc *cloudwatchlogs.Client) error {
	for _, policyType := range logsAccountPolicyTypes {
		var nextToken *string
		for {
			output, err := svc.DescribeAccountPolicies(context.TODO(), &cloudwatchlogs.DescribeAccountPoliciesInput{
				PolicyType: policyType,
				NextToken:  nextToken,
			})
			if err != nil {
				return err
			}
			for _, policy := range output.AccountPolicies {
				policyName := StringValue(policy.PolicyName)
				if policyName == "" || StringValue(policy.PolicyDocument) == "" {
					continue
				}
				policyTypeName := string(policy.PolicyType)
				id := fmt.Sprintf("%s:%s", policyName, policyTypeName)
				g.Resources = append(g.Resources, terraformutils.NewResource(
					id,
					logsResourceName(policyName, policyTypeName),
					"aws_cloudwatch_log_account_policy",
					"aws",
					map[string]string{
						"policy_name": policyName,
						"policy_type": policyTypeName,
					},
					logsAllowEmptyValues,
					map[string]interface{}{}))
			}
			nextToken = output.NextToken
			if nextToken == nil {
				break
			}
		}
	}
	return nil
}

func (g *LogsGenerator) addQueryDefinitions(svc *cloudwatchlogs.Client, region, account string) error {
	var nextToken *string
	for {
		output, err := svc.DescribeQueryDefinitions(context.TODO(), &cloudwatchlogs.DescribeQueryDefinitionsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return err
		}
		for _, queryDefinition := range output.QueryDefinitions {
			queryDefinitionID := StringValue(queryDefinition.QueryDefinitionId)
			if queryDefinitionID == "" {
				continue
			}
			queryDefinitionARN := logsQueryDefinitionARN(region, account, queryDefinitionID)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				queryDefinitionARN,
				logsResourceName(StringValue(queryDefinition.Name), queryDefinitionID),
				"aws_cloudwatch_query_definition",
				"aws",
				logsAllowEmptyValues))
		}
		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
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

func logsResourceName(parts ...string) string {
	var name string
	for _, part := range parts {
		if part == "" {
			continue
		}
		if name != "" {
			name += "_"
		}
		name += part
	}
	return name
}

func logsResourceNotFound(err error) bool {
	var notFound *types.ResourceNotFoundException
	return errors.As(err, &notFound)
}

func logsQueryDefinitionARN(region, account, queryDefinitionID string) string {
	return fmt.Sprintf("arn:%s:logs:%s:%s:query-definition:%s", awsPartitionFromRegion(region), region, account, queryDefinitionID)
}

func awsPartitionFromRegion(region string) string {
	switch {
	case strings.HasPrefix(region, "cn-"):
		return "aws-cn"
	case strings.HasPrefix(region, "us-gov-"):
		return "aws-us-gov"
	default:
		return "aws"
	}
}
