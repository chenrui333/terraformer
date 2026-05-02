// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"

	"github.com/aws/aws-sdk-go-v2/service/configservice"
	configtypes "github.com/aws/aws-sdk-go-v2/service/configservice/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var configAllowEmptyValues = []string{"tags."}

const configRemediationBatchSize = 25

type ConfigGenerator struct {
	AWSService
}

type configOptionalResourceLoader struct {
	name string
	load func() error
}

func (g *ConfigGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	client := configservice.NewFromConfig(config)

	configurationRecorderRefs, err := g.addConfigurationRecorders(client)
	if err != nil {
		return err
	}
	configRuleNames, err := g.addConfigRules(client, configurationRecorderRefs)
	if err != nil {
		return err
	}
	deliveryChannelRefs, err := g.addDeliveryChannels(client, configurationRecorderRefs)
	if err != nil {
		return err
	}
	if err := g.addConfigurationRecorderStatuses(client, slices.Concat(configurationRecorderRefs, deliveryChannelRefs)); err != nil {
		return err
	}

	g.loadOptionalResources([]configOptionalResourceLoader{
		{name: "configuration aggregators", load: func() error { return g.addConfigurationAggregators(client) }},
		{name: "aggregate authorizations", load: func() error { return g.addAggregateAuthorizations(client) }},
		{name: "organization config rules", load: func() error { return g.addOrganizationConfigRules(client) }},
		{name: "remediation configurations", load: func() error { return g.addRemediationConfigurations(client, configRuleNames) }},
		{name: "retention configurations", load: func() error { return g.addRetentionConfigurations(client) }},
	})

	return nil
}

func (g *ConfigGenerator) addConfigurationRecorders(svc *configservice.Client) ([]string, error) {
	configurationRecorders, err := svc.DescribeConfigurationRecorders(context.TODO(),
		&configservice.DescribeConfigurationRecordersInput{})

	if err != nil {
		return nil, err
	}
	var configurationRecorderRefs []string
	for _, configurationRecorder := range configurationRecorders.ConfigurationRecorders {
		name := StringValue(configurationRecorder.Name)
		if name == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			name,
			name,
			"aws_config_configuration_recorder",
			"aws",
			configAllowEmptyValues,
		))
		configurationRecorderRefs = append(configurationRecorderRefs,
			configResourceRef("aws_config_configuration_recorder", name))
	}
	return configurationRecorderRefs, nil
}

func (g *ConfigGenerator) addConfigRules(svc *configservice.Client, configurationRecorderRefs []string) ([]string, error) {
	var nextToken *string
	var configRuleNames []string

	for {
		configRules, err := svc.DescribeConfigRules(
			context.TODO(),
			&configservice.DescribeConfigRulesInput{
				NextToken: nextToken,
			})

		if err != nil {
			return nil, err
		}
		for _, configRule := range configRules.ConfigRules {
			name := StringValue(configRule.ConfigRuleName)
			if name == "" {
				continue
			}
			configRuleNames = append(configRuleNames, name)
			g.Resources = append(g.Resources, terraformutils.NewResource(
				name,
				name,
				"aws_config_config_rule",
				"aws",
				map[string]string{},
				configAllowEmptyValues,
				map[string]interface{}{
					"depends_on": configurationRecorderRefs,
				},
			))
		}
		nextToken = configRules.NextToken
		if nextToken == nil {
			break
		}
	}
	return configRuleNames, nil
}

func (g *ConfigGenerator) addDeliveryChannels(svc *configservice.Client, configurationRecorderRefs []string) ([]string, error) {
	deliveryChannels, err := svc.DescribeDeliveryChannels(context.TODO(),
		&configservice.DescribeDeliveryChannelsInput{})

	if err != nil {
		return nil, err
	}
	var deliveryChannelRefs []string
	for _, deliveryChannel := range deliveryChannels.DeliveryChannels {
		name := StringValue(deliveryChannel.Name)
		if name == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			name,
			name,
			"aws_config_delivery_channel",
			"aws",
			map[string]string{},
			configAllowEmptyValues,
			map[string]interface{}{
				"depends_on": configurationRecorderRefs,
			},
		))
		deliveryChannelRefs = append(deliveryChannelRefs, configResourceRef("aws_config_delivery_channel", name))
	}
	return deliveryChannelRefs, nil
}

func (g *ConfigGenerator) addConfigurationRecorderStatuses(svc *configservice.Client, dependsOn []string) error {
	output, err := svc.DescribeConfigurationRecorderStatus(context.TODO(),
		&configservice.DescribeConfigurationRecorderStatusInput{})
	if err != nil {
		return err
	}
	for _, recorderStatus := range output.ConfigurationRecordersStatus {
		name := StringValue(recorderStatus.Name)
		if name == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			name,
			name,
			"aws_config_configuration_recorder_status",
			"aws",
			map[string]string{
				"name": name,
			},
			configAllowEmptyValues,
			map[string]interface{}{
				"depends_on": dependsOn,
			},
		))
	}
	return nil
}

func (g *ConfigGenerator) addConfigurationAggregators(svc *configservice.Client) error {
	p := configservice.NewDescribeConfigurationAggregatorsPaginator(svc,
		&configservice.DescribeConfigurationAggregatorsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, aggregator := range page.ConfigurationAggregators {
			name := StringValue(aggregator.ConfigurationAggregatorName)
			if name == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				name,
				name,
				"aws_config_configuration_aggregator",
				"aws",
				map[string]string{
					"name": name,
				},
				configAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *ConfigGenerator) addAggregateAuthorizations(svc *configservice.Client) error {
	p := configservice.NewDescribeAggregationAuthorizationsPaginator(svc,
		&configservice.DescribeAggregationAuthorizationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, authorization := range page.AggregationAuthorizations {
			accountID := StringValue(authorization.AuthorizedAccountId)
			region := StringValue(authorization.AuthorizedAwsRegion)
			if accountID == "" || region == "" {
				continue
			}
			id := configAggregateAuthorizationID(accountID, region)
			g.Resources = append(g.Resources, terraformutils.NewResource(
				id,
				id,
				"aws_config_aggregate_authorization",
				"aws",
				map[string]string{
					"account_id": accountID,
					"region":     region,
				},
				configAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *ConfigGenerator) addOrganizationConfigRules(svc *configservice.Client) error {
	p := configservice.NewDescribeOrganizationConfigRulesPaginator(svc,
		&configservice.DescribeOrganizationConfigRulesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, rule := range page.OrganizationConfigRules {
			name := StringValue(rule.OrganizationConfigRuleName)
			resourceType := configOrganizationRuleResourceType(rule)
			if name == "" || resourceType == "" {
				continue
			}
			attributes, ok := g.organizationConfigRuleAttributes(svc, name, rule)
			if !ok {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				name,
				name,
				resourceType,
				"aws",
				attributes,
				configAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *ConfigGenerator) organizationConfigRuleAttributes(
	svc *configservice.Client,
	name string,
	rule configtypes.OrganizationConfigRule,
) (map[string]string, bool) {
	attributes := map[string]string{
		"name": name,
	}
	if rule.OrganizationCustomPolicyRuleMetadata == nil {
		return attributes, true
	}

	output, err := svc.GetOrganizationCustomRulePolicy(context.TODO(),
		&configservice.GetOrganizationCustomRulePolicyInput{
			OrganizationConfigRuleName: &name,
		})
	if err != nil {
		log.Printf("Skipping AWS Config organization custom policy rule %s: %v", name, err)
		return nil, false
	}
	policyText := StringValue(output.PolicyText)
	if policyText == "" {
		return nil, false
	}
	attributes["policy_text"] = policyText
	return attributes, true
}

func (g *ConfigGenerator) addRemediationConfigurations(svc *configservice.Client, configRuleNames []string) error {
	for _, configRuleNamesBatch := range chunkStrings(configRuleNames, configRemediationBatchSize) {
		if err := g.addRemediationConfigurationsBatch(svc, configRuleNamesBatch); err != nil {
			return err
		}
	}
	return nil
}

func (g *ConfigGenerator) addRemediationConfigurationsBatch(svc *configservice.Client, configRuleNames []string) error {
	if len(configRuleNames) == 0 {
		return nil
	}
	output, err := svc.DescribeRemediationConfigurations(context.TODO(),
		&configservice.DescribeRemediationConfigurationsInput{
			ConfigRuleNames: configRuleNames,
		})
	if err != nil {
		if configRemediationConfigurationMissing(err) && len(configRuleNames) > 1 {
			for _, configRuleName := range configRuleNames {
				if err := g.addRemediationConfigurationsBatch(svc, []string{configRuleName}); err != nil {
					return err
				}
			}
			return nil
		}
		if configRemediationConfigurationMissing(err) {
			return nil
		}
		return err
	}
	for _, remediationConfiguration := range output.RemediationConfigurations {
		name := StringValue(remediationConfiguration.ConfigRuleName)
		if name == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			name,
			name,
			"aws_config_remediation_configuration",
			"aws",
			map[string]string{
				"config_rule_name": name,
			},
			configAllowEmptyValues,
			map[string]interface{}{
				"depends_on": []string{configRuleResourceRef(name)},
			},
		))
	}
	return nil
}

func (g *ConfigGenerator) addRetentionConfigurations(svc *configservice.Client) error {
	p := configservice.NewDescribeRetentionConfigurationsPaginator(svc,
		&configservice.DescribeRetentionConfigurationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, retentionConfiguration := range page.RetentionConfigurations {
			name := StringValue(retentionConfiguration.Name)
			if name == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				name,
				name,
				"aws_config_retention_configuration",
				"aws",
				configAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *ConfigGenerator) loadOptionalResources(loaders []configOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			log.Printf("Skipping AWS Config %s: %v", loader.name, err)
		}
	}
}

func configAggregateAuthorizationID(accountID, region string) string {
	return fmt.Sprintf("%s:%s", accountID, region)
}

func configResourceRef(resourceType, name string) string {
	return resourceType + "." + terraformutils.TfSanitize(name)
}

func configRuleResourceRef(name string) string {
	return configResourceRef("aws_config_config_rule", name)
}

func configOrganizationRuleResourceType(rule configtypes.OrganizationConfigRule) string {
	// AWS returns exactly one metadata shape, but prefer the managed shape if a
	// malformed response sets multiple fields so classification stays stable.
	switch {
	case rule.OrganizationManagedRuleMetadata != nil:
		return "aws_config_organization_managed_rule"
	case rule.OrganizationCustomRuleMetadata != nil:
		return "aws_config_organization_custom_rule"
	case rule.OrganizationCustomPolicyRuleMetadata != nil:
		return "aws_config_organization_custom_policy_rule"
	default:
		return ""
	}
}

func configRemediationConfigurationMissing(err error) bool {
	var notFound *configtypes.NoSuchConfigRuleException
	if errors.As(err, &notFound) {
		return true
	}
	var noRemediation *configtypes.NoSuchRemediationConfigurationException
	return errors.As(err, &noRemediation)
}

func chunkStrings(values []string, size int) [][]string {
	if size <= 0 {
		return nil
	}
	var chunks [][]string
	for start := 0; start < len(values); start += size {
		end := start + size
		if end > len(values) {
			end = len(values)
		}
		chunks = append(chunks, values[start:end])
	}
	return chunks
}
