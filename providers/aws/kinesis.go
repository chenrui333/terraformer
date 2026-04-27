// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/chenrui333/terraformer/terraformutils"
)

var kinesisAllowEmptyValues = []string{"tags."}

type KinesisGenerator struct {
	AWSService
}

func (g *KinesisGenerator) createResources(streamNames []string) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, resourceName := range streamNames {
		resources = append(resources, terraformutils.NewResource(
			resourceName,
			resourceName,
			"aws_kinesis_stream",
			"aws",
			map[string]string{"name": resourceName},
			kinesisAllowEmptyValues,
			map[string]interface{}{}))
	}
	return resources
}

func (g *KinesisGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := kinesis.NewFromConfig(config)

	var results *kinesis.ListStreamsOutput
	var request = kinesis.ListStreamsInput{}
	var err error

	for results == nil || *results.HasMoreStreams {
		results, err = svc.ListStreams(context.TODO(), &request)
		if err != nil {
			return err
		}

		g.Resources = append(g.Resources, g.createResources(results.StreamNames)...)

		if len(results.StreamNames) > 0 {
			request = kinesis.ListStreamsInput{
				ExclusiveStartStreamName: &results.StreamNames[len(results.StreamNames)-1],
			}
		}
	}
	return nil
}
