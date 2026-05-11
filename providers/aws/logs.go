// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var logsAllowEmptyValues = []string{"tags."}

const (
	logsDeliveryResourceType                  = "aws_cloudwatch_log_delivery"
	logsDeliveryDestinationResourceType       = "aws_cloudwatch_log_delivery_destination"
	logsDeliveryDestinationPolicyResourceType = "aws_cloudwatch_log_delivery_destination_policy"
	logsDeliverySourceResourceType            = "aws_cloudwatch_log_delivery_source"
	logsDestinationPolicyResourceType         = "aws_cloudwatch_log_destination_policy"
	logsIndexPolicyResourceType               = "aws_cloudwatch_log_index_policy"
)

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
		logsOptionalResourceLoader{name: "index policies", load: func() error { return g.addIndexPolicies(svc, logGroupNames) }},
		logsOptionalResourceLoader{name: "delivery sources", load: func() error { return g.addDeliverySources(svc) }},
		logsOptionalResourceLoader{name: "delivery destinations", load: func() error { return g.addDeliveryDestinations(svc) }},
		logsOptionalResourceLoader{name: "deliveries", load: func() error { return g.addDeliveries(svc) }},
		logsOptionalResourceLoader{name: "resource policies", load: func() error { return g.addResourcePolicies(svc) }},
		logsOptionalResourceLoader{name: "account policies", load: func() error { return g.addAccountPolicies(svc) }},
		logsOptionalResourceLoader{name: "query definitions", load: func() error { return g.addQueryDefinitions(svc) }},
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
				g.Resources = append(g.Resources, terraformutils.NewResource(
					filterName,
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

// AWS does not provide a list API for log group data protection policies, so
// this optional loader probes each imported log group.
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
			if resource, ok := newLogsDestinationPolicyResource(destination); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *LogsGenerator) addIndexPolicies(svc *cloudwatchlogs.Client, logGroupNames []string) error {
	for _, logGroupName := range logGroupNames {
		var nextToken *string
		for {
			output, err := svc.DescribeIndexPolicies(context.TODO(), &cloudwatchlogs.DescribeIndexPoliciesInput{
				LogGroupIdentifiers: []string{logGroupName},
				NextToken:           nextToken,
			})
			if err != nil {
				if logsResourceNotFound(err) {
					break
				}
				return err
			}
			for _, policy := range output.IndexPolicies {
				if resource, ok := newLogsIndexPolicyResource(logGroupName, policy); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
			nextToken = output.NextToken
			if !awsHasMorePages(nextToken) {
				break
			}
		}
	}
	return nil
}

func (g *LogsGenerator) addDeliverySources(svc *cloudwatchlogs.Client) error {
	p := cloudwatchlogs.NewDescribeDeliverySourcesPaginator(svc, &cloudwatchlogs.DescribeDeliverySourcesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, source := range page.DeliverySources {
			if resource, ok := newLogsDeliverySourceResource(source); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *LogsGenerator) addDeliveryDestinations(svc *cloudwatchlogs.Client) error {
	p := cloudwatchlogs.NewDescribeDeliveryDestinationsPaginator(svc, &cloudwatchlogs.DescribeDeliveryDestinationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, destination := range page.DeliveryDestinations {
			destinationName := StringValue(destination.Name)
			if resource, ok := newLogsDeliveryDestinationResource(destination); ok {
				g.Resources = append(g.Resources, resource)
			}
			if destinationName == "" {
				continue
			}
			policy, err := svc.GetDeliveryDestinationPolicy(context.TODO(), &cloudwatchlogs.GetDeliveryDestinationPolicyInput{
				DeliveryDestinationName: &destinationName,
			})
			if err != nil {
				if logsResourceNotFound(err) {
					continue
				}
				log.Printf("skipping CloudWatch Logs delivery destination policy discovery for %s: %v", destinationName, err)
				continue
			}
			if policy == nil || policy.Policy == nil {
				continue
			}
			if resource, ok := newLogsDeliveryDestinationPolicyResource(destinationName, *policy.Policy); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *LogsGenerator) addDeliveries(svc *cloudwatchlogs.Client) error {
	p := cloudwatchlogs.NewDescribeDeliveriesPaginator(svc, &cloudwatchlogs.DescribeDeliveriesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, delivery := range page.Deliveries {
			if resource, ok := newLogsDeliveryResource(delivery); ok {
				g.Resources = append(g.Resources, resource)
			}
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
			policyID, resourceName, attributes := logsResourcePolicyResource(policy)
			if policyID == "" || StringValue(policy.PolicyDocument) == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				policyID,
				resourceName,
				"aws_cloudwatch_log_resource_policy",
				"aws",
				attributes,
				logsAllowEmptyValues,
				map[string]interface{}{}))
		}
		nextToken = output.NextToken
		if !awsHasMorePages(nextToken) {
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
				g.Resources = append(g.Resources, terraformutils.NewResource(
					policyName,
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
			if !awsHasMorePages(nextToken) {
				break
			}
		}
	}
	return nil
}

func (g *LogsGenerator) addQueryDefinitions(svc *cloudwatchlogs.Client) error {
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
			g.Resources = append(g.Resources, terraformutils.NewResource(
				queryDefinitionID,
				logsResourceName(StringValue(queryDefinition.Name), queryDefinitionID),
				"aws_cloudwatch_query_definition",
				"aws",
				map[string]string{
					"name": StringValue(queryDefinition.Name),
				},
				logsAllowEmptyValues,
				map[string]interface{}{}))
		}
		nextToken = output.NextToken
		if !awsHasMorePages(nextToken) {
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

func logsResourcePolicyResource(policy types.ResourcePolicy) (string, string, map[string]string) {
	if policy.PolicyScope == types.PolicyScopeResource {
		resourceArn := StringValue(policy.ResourceArn)
		if resourceArn == "" {
			return "", "", nil
		}
		return resourceArn, logsResourceName(StringValue(policy.PolicyName), resourceArn), map[string]string{
			"policy_scope": string(types.PolicyScopeResource),
			"resource_arn": resourceArn,
		}
	}

	policyName := StringValue(policy.PolicyName)
	if policyName == "" {
		return "", "", nil
	}
	return policyName, policyName, map[string]string{
		"policy_name": policyName,
	}
}

func newLogsDestinationPolicyResource(destination types.Destination) (terraformutils.Resource, bool) {
	destinationName := StringValue(destination.DestinationName)
	if destinationName == "" || StringValue(destination.AccessPolicy) == "" {
		return terraformutils.Resource{}, false
	}

	return terraformutils.NewResource(
		destinationName,
		destinationName,
		logsDestinationPolicyResourceType,
		"aws",
		map[string]string{
			"destination_name": destinationName,
		},
		logsAllowEmptyValues,
		map[string]interface{}{}), true
}

func newLogsIndexPolicyResource(logGroupName string, policy types.IndexPolicy) (terraformutils.Resource, bool) {
	if logGroupName == "" || StringValue(policy.PolicyDocument) == "" {
		return terraformutils.Resource{}, false
	}
	if policy.Source != "" && policy.Source != types.IndexSourceLogGroup {
		return terraformutils.Resource{}, false
	}

	return terraformutils.NewResource(
		logGroupName,
		logGroupName,
		logsIndexPolicyResourceType,
		"aws",
		map[string]string{
			"log_group_name": logGroupName,
		},
		logsAllowEmptyValues,
		map[string]interface{}{}), true
}

func newLogsDeliverySourceResource(source types.DeliverySource) (terraformutils.Resource, bool) {
	if !logsDeliverySourceImportable(source) {
		return terraformutils.Resource{}, false
	}

	sourceName := StringValue(source.Name)
	resourceArn := source.ResourceArns[0]
	return terraformutils.NewResource(
		sourceName,
		logsDeliveryResourceName("delivery_source", sourceName),
		logsDeliverySourceResourceType,
		"aws",
		map[string]string{
			"log_type":     StringValue(source.LogType),
			"name":         sourceName,
			"resource_arn": resourceArn,
		},
		logsAllowEmptyValues,
		map[string]interface{}{}), true
}

func logsDeliverySourceImportable(source types.DeliverySource) bool {
	if StringValue(source.Name) == "" || StringValue(source.LogType) == "" || len(source.ResourceArns) == 0 || source.ResourceArns[0] == "" {
		return false
	}
	return source.StatusReason != types.DeliverySourceStatusReasonResourceDeleted
}

func newLogsDeliveryDestinationResource(destination types.DeliveryDestination) (terraformutils.Resource, bool) {
	destinationName := StringValue(destination.Name)
	if destinationName == "" {
		return terraformutils.Resource{}, false
	}

	return terraformutils.NewResource(
		destinationName,
		logsDeliveryResourceName("delivery_destination", destinationName),
		logsDeliveryDestinationResourceType,
		"aws",
		map[string]string{
			"name": destinationName,
		},
		logsAllowEmptyValues,
		map[string]interface{}{}), true
}

func newLogsDeliveryDestinationPolicyResource(destinationName string, policy types.Policy) (terraformutils.Resource, bool) {
	if destinationName == "" || StringValue(policy.DeliveryDestinationPolicy) == "" {
		return terraformutils.Resource{}, false
	}

	return terraformutils.NewResource(
		destinationName,
		logsDeliveryResourceName("delivery_destination_policy", destinationName),
		logsDeliveryDestinationPolicyResourceType,
		"aws",
		map[string]string{
			"delivery_destination_name":   destinationName,
			"delivery_destination_policy": StringValue(policy.DeliveryDestinationPolicy),
		},
		logsAllowEmptyValues,
		map[string]interface{}{}), true
}

func newLogsDeliveryResource(delivery types.Delivery) (terraformutils.Resource, bool) {
	deliveryID := StringValue(delivery.Id)
	deliverySourceName := StringValue(delivery.DeliverySourceName)
	deliveryDestinationArn := StringValue(delivery.DeliveryDestinationArn)
	if deliveryID == "" || deliverySourceName == "" || deliveryDestinationArn == "" {
		return terraformutils.Resource{}, false
	}

	return terraformutils.NewResource(
		deliveryID,
		logsDeliveryResourceName("delivery", deliveryID),
		logsDeliveryResourceType,
		"aws",
		map[string]string{
			"delivery_destination_arn": deliveryDestinationArn,
			"delivery_source_name":     deliverySourceName,
		},
		logsAllowEmptyValues,
		map[string]interface{}{}), true
}

func logsDeliveryResourceName(parts ...string) string {
	return logsResourceNameWithLengths(parts...)
}

func logsResourceNameWithLengths(parts ...string) string {
	var name string
	for _, part := range parts {
		if part == "" {
			continue
		}
		if name != "" {
			name += "_"
		}
		name += fmt.Sprintf("%d_%s", len(part), part)
	}
	return name
}
