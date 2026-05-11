// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/mq"
	mqtypes "github.com/aws/aws-sdk-go-v2/service/mq/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	mqBrokerResourceType        = "aws_mq_broker"
	mqConfigurationResourceType = "aws_mq_configuration"
)

var mqAllowEmptyValues = []string{"tags."}

type MQGenerator struct {
	AWSService
}

func (g *MQGenerator) loadBrokers(svc *mq.Client) error {
	p := mq.NewListBrokersPaginator(svc, &mq.ListBrokersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, broker := range page.BrokerSummaries {
			if resource, ok := newMQBrokerResource(broker); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *MQGenerator) loadConfigurations(svc *mq.Client) error {
	input := &mq.ListConfigurationsInput{MaxResults: aws.Int32(100)}
	for {
		output, err := svc.ListConfigurations(context.TODO(), input)
		if err != nil {
			return err
		}
		for _, configuration := range output.Configurations {
			if resource, ok := newMQConfigurationResource(configuration); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
		if output.NextToken == nil {
			break
		}
		input.NextToken = output.NextToken
	}

	return nil
}

func (g *MQGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := mq.NewFromConfig(config)

	err := g.loadBrokers(svc)
	if err != nil {
		return err
	}

	err = g.loadConfigurations(svc)
	if err != nil {
		return err
	}

	return nil
}

func newMQBrokerResource(broker mqtypes.BrokerSummary) (terraformutils.Resource, bool) {
	brokerID := StringValue(broker.BrokerId)
	if brokerID == "" {
		return terraformutils.Resource{}, false
	}
	brokerName := StringValue(broker.BrokerName)
	if brokerName == "" {
		brokerName = brokerID
	}
	return terraformutils.NewSimpleResource(
		mqBrokerImportID(brokerID),
		brokerName,
		mqBrokerResourceType,
		"aws",
		mqAllowEmptyValues), true
}

func newMQConfigurationResource(configuration mqtypes.Configuration) (terraformutils.Resource, bool) {
	configurationID := StringValue(configuration.Id)
	if configurationID == "" {
		return terraformutils.Resource{}, false
	}
	configurationName := StringValue(configuration.Name)
	if configurationName == "" {
		configurationName = configurationID
	}
	return terraformutils.NewSimpleResource(
		mqConfigurationImportID(configurationID),
		mqResourceName("configuration", configurationName, configurationID),
		mqConfigurationResourceType,
		"aws",
		mqAllowEmptyValues), true
}

func mqBrokerImportID(brokerID string) string {
	return brokerID
}

func mqConfigurationImportID(configurationID string) string {
	return configurationID
}

func mqResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d/%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return "mq-resource"
	}
	return strings.Join(cleanParts, "/")
}
