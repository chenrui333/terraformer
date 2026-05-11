// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
	"github.com/iancoleman/strcase"
)

var logsAllowEmptyValues = []string{"tags."}

const (
	logsAnomalyDetectorResourceType           = "aws_cloudwatch_log_anomaly_detector"
	logsDeliveryResourceType                  = "aws_cloudwatch_log_delivery"
	logsDeliveryDestinationResourceType       = "aws_cloudwatch_log_delivery_destination"
	logsDeliveryDestinationPolicyResourceType = "aws_cloudwatch_log_delivery_destination_policy"
	logsDeliverySourceResourceType            = "aws_cloudwatch_log_delivery_source"
	logsDestinationPolicyResourceType         = "aws_cloudwatch_log_destination_policy"
	logsIndexPolicyResourceType               = "aws_cloudwatch_log_index_policy"
	logsTransformerResourceType               = "aws_cloudwatch_log_transformer"
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
	var logGroups []types.LogGroup
	p := cloudwatchlogs.NewDescribeLogGroupsPaginator(svc, &cloudwatchlogs.DescribeLogGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, g.createResources(page)...)
		logGroups = append(logGroups, page.LogGroups...)
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
		logsOptionalResourceLoader{name: "anomaly detectors", load: func() error { return g.addAnomalyDetectors(svc) }},
		logsOptionalResourceLoader{name: "transformers", load: func() error { return g.addTransformers(svc, logGroups) }},
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

func (g *LogsGenerator) addAnomalyDetectors(svc *cloudwatchlogs.Client) error {
	p := cloudwatchlogs.NewListLogAnomalyDetectorsPaginator(svc, &cloudwatchlogs.ListLogAnomalyDetectorsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, detector := range page.AnomalyDetectors {
			if resource, ok := newLogsAnomalyDetectorResource(detector); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *LogsGenerator) addTransformers(svc *cloudwatchlogs.Client, logGroups []types.LogGroup) error {
	for _, logGroup := range logGroups {
		logGroupARN := logsTransformerLogGroupARN(logGroup)
		if logGroupARN == "" {
			continue
		}
		transformer, err := svc.GetTransformer(context.TODO(), &cloudwatchlogs.GetTransformerInput{
			LogGroupIdentifier: &logGroupARN,
		})
		if err != nil {
			if logsResourceNotFound(err) {
				continue
			}
			return err
		}
		if resource, ok := newLogsTransformerResource(StringValue(logGroup.LogGroupName), logGroupARN, transformer); ok {
			g.Resources = append(g.Resources, resource)
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

func newLogsAnomalyDetectorResource(detector types.AnomalyDetector) (terraformutils.Resource, bool) {
	detectorARN := StringValue(detector.AnomalyDetectorArn)
	enabled, ok := logsAnomalyDetectorEnabledValue(detector.AnomalyDetectorStatus)
	if !ok || detectorARN == "" || !logsStringListValuesComplete(detector.LogGroupArnList) {
		return terraformutils.Resource{}, false
	}

	attributes := map[string]string{
		"arn":     detectorARN,
		"enabled": strconv.FormatBool(enabled),
	}
	for key, value := range logsStringListAttributes("log_group_arn_list", detector.LogGroupArnList) {
		attributes[key] = value
	}

	resource := terraformutils.NewResource(
		detectorARN,
		logsAnomalyDetectorResourceName(StringValue(detector.DetectorName), detectorARN),
		logsAnomalyDetectorResourceType,
		"aws",
		attributes,
		logsAllowEmptyValues,
		map[string]interface{}{})
	if resource.InstanceState.Meta == nil {
		resource.InstanceState.Meta = map[string]interface{}{}
	}
	resource.InstanceState.Meta[tfcompat.MetaKeyPreserveIDAfterRefresh] = true
	return resource, true
}

func logsAnomalyDetectorEnabledValue(status types.AnomalyDetectorStatus) (bool, bool) {
	switch status {
	case "":
		return false, false
	case types.AnomalyDetectorStatusDeleted:
		return false, false
	case types.AnomalyDetectorStatusPaused:
		return false, true
	default:
		return true, true
	}
}

func logsAnomalyDetectorResourceName(detectorName, detectorARN string) string {
	return logsResourceNameWithLengths("anomaly_detector", detectorName, detectorARN)
}

func newLogsTransformerResource(logGroupName, logGroupARN string, transformer *cloudwatchlogs.GetTransformerOutput) (terraformutils.Resource, bool) {
	if logGroupARN == "" || transformer == nil || len(transformer.TransformerConfig) == 0 {
		return terraformutils.Resource{}, false
	}

	attributes := map[string]string{
		"log_group_arn": logGroupARN,
	}
	for key, value := range logsTransformerConfigAttributes(transformer.TransformerConfig) {
		attributes[key] = value
	}

	resource := terraformutils.NewResource(
		logGroupARN,
		logsTransformerResourceName(logGroupName, logGroupARN),
		logsTransformerResourceType,
		"aws",
		attributes,
		logsAllowEmptyValues,
		map[string]interface{}{})
	if resource.InstanceState.Meta == nil {
		resource.InstanceState.Meta = map[string]interface{}{}
	}
	resource.InstanceState.Meta[tfcompat.MetaKeyPreserveIDAfterRefresh] = true
	return resource, true
}

func logsTransformerLogGroupARN(logGroup types.LogGroup) string {
	if logGroupARN := StringValue(logGroup.LogGroupArn); logGroupARN != "" {
		return logGroupARN
	}
	return strings.TrimSuffix(StringValue(logGroup.Arn), ":*")
}

func logsTransformerResourceName(logGroupName, logGroupARN string) string {
	return logsResourceNameWithLengths("transformer", logGroupName, logGroupARN)
}

func logsTransformerConfigAttributes(processors []types.Processor) map[string]string {
	attributes := map[string]string{
		"transformer_config.#": strconv.Itoa(len(processors)),
	}
	for i, processor := range processors {
		logsFlattenTransformerValue(reflect.ValueOf(processor), fmt.Sprintf("transformer_config.%d", i), attributes)
	}
	return attributes
}

func logsFlattenTransformerValue(value reflect.Value, path string, attributes map[string]string) {
	value = logsIndirectTransformerValue(value)
	if !value.IsValid() {
		return
	}
	switch value.Kind() {
	case reflect.Struct:
		valueType := value.Type()
		for i := 0; i < value.NumField(); i++ {
			field := valueType.Field(i)
			if field.PkgPath != "" {
				continue
			}
			fieldValue := value.Field(i)
			if logsTransformerValueEmpty(fieldValue) {
				continue
			}
			logsFlattenTransformerField(fieldValue, path+"."+logsTransformerFieldName(field.Name), attributes)
		}
	case reflect.String:
		if value.String() != "" {
			attributes[path] = value.String()
		}
	case reflect.Bool:
		attributes[path] = strconv.FormatBool(value.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value.Int() != 0 {
			attributes[path] = strconv.FormatInt(value.Int(), 10)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if value.Uint() != 0 {
			attributes[path] = strconv.FormatUint(value.Uint(), 10)
		}
	case reflect.Float32, reflect.Float64:
		if value.Float() != 0 {
			attributes[path] = strconv.FormatFloat(value.Float(), 'f', -1, value.Type().Bits())
		}
	}
}

func logsFlattenTransformerField(value reflect.Value, path string, attributes map[string]string) {
	value = logsIndirectTransformerValue(value)
	if !value.IsValid() {
		return
	}
	switch value.Kind() {
	case reflect.Struct:
		attributes[path+".#"] = "1"
		logsFlattenTransformerValue(value, path+".0", attributes)
	case reflect.Slice:
		attributes[path+".#"] = strconv.Itoa(value.Len())
		for i := 0; i < value.Len(); i++ {
			logsFlattenTransformerValue(value.Index(i), fmt.Sprintf("%s.%d", path, i), attributes)
		}
	default:
		logsFlattenTransformerValue(value, path, attributes)
	}
}

func logsIndirectTransformerValue(value reflect.Value) reflect.Value {
	for value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func logsTransformerValueEmpty(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface:
		return value.IsNil()
	case reflect.Slice, reflect.Map:
		return value.Len() == 0
	case reflect.String:
		return value.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return value.Float() == 0
	default:
		return false
	}
}

func logsTransformerFieldName(name string) string {
	if name == "Entries" {
		return "entry"
	}
	return strcase.ToSnake(name)
}

func logsStringListAttributes(name string, values []string) map[string]string {
	attributes := map[string]string{
		name + ".#": strconv.Itoa(len(values)),
	}
	for i, value := range values {
		attributes[fmt.Sprintf("%s.%d", name, i)] = value
	}
	return attributes
}

func logsStringListValuesComplete(values []string) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if value == "" {
			return false
		}
	}
	return true
}
