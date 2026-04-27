// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/mq"
	"github.com/chenrui333/terraformer/terraformutils"
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
			resourceName := StringValue(broker.BrokerName)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_mq_broker",
				"aws",
				mqAllowEmptyValues))
		}
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

	return nil
}
