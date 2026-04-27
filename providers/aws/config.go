// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/chenrui333/terraformer/terraformutils"
)

var configAllowEmptyValues = []string{"tags."}

type ConfigGenerator struct {
	AWSService
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
	err = g.addConfigRules(client, configurationRecorderRefs)
	if err != nil {
		return err
	}
	err = g.addDeliveryChannels(client, configurationRecorderRefs)
	return err
}

func (g *ConfigGenerator) addConfigurationRecorders(svc *configservice.Client) ([]string, error) {
	configurationRecorders, err := svc.DescribeConfigurationRecorders(context.TODO(),
		&configservice.DescribeConfigurationRecordersInput{})

	if err != nil {
		return nil, err
	}
	var configurationRecorderRefs []string
	for _, configurationRecorder := range configurationRecorders.ConfigurationRecorders {
		name := *configurationRecorder.Name
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			name,
			name,
			"aws_config_configuration_recorder",
			"aws",
			configAllowEmptyValues,
		))
		configurationRecorderRefs = append(configurationRecorderRefs,
			"aws_config_configuration_recorder.tfer--"+name)
	}
	return configurationRecorderRefs, nil
}

func (g *ConfigGenerator) addConfigRules(svc *configservice.Client, configurationRecorderRefs []string) error {
	var nextToken *string

	for {
		configRules, err := svc.DescribeConfigRules(
			context.TODO(),
			&configservice.DescribeConfigRulesInput{
				NextToken: nextToken,
			})

		if err != nil {
			return err
		}
		for _, configRule := range configRules.ConfigRules {
			name := *configRule.ConfigRuleName
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
	return nil
}

func (g *ConfigGenerator) addDeliveryChannels(svc *configservice.Client, configurationRecorderRefs []string) error {
	deliveryChannels, err := svc.DescribeDeliveryChannels(context.TODO(),
		&configservice.DescribeDeliveryChannelsInput{})

	if err != nil {
		return err
	}
	for _, deliveryChannel := range deliveryChannels.DeliveryChannels {
		name := *deliveryChannel.Name
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
	}
	return nil
}
