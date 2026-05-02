// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	kinesistypes "github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

var kinesisAllowEmptyValues = []string{"tags."}

type KinesisGenerator struct {
	AWSService
}

type kinesisOptionalResourceLoader struct {
	name string
	load func() error
}

func (g *KinesisGenerator) loadOptionalResources(loaders []kinesisOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			if kinesisResourceMissing(err) {
				continue
			}
			log.Printf("Skipping Kinesis %s: %v", loader.name, err)
		}
	}
}

func (g *KinesisGenerator) createResources(streamNames []string) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, resourceName := range streamNames {
		resources = append(resources, newKinesisStreamResource(resourceName))
	}
	return resources
}

func newKinesisStreamResource(resourceName string) terraformutils.Resource {
	return terraformutils.NewResource(
		resourceName,
		resourceName,
		"aws_kinesis_stream",
		"aws",
		map[string]string{"name": resourceName},
		kinesisAllowEmptyValues,
		map[string]interface{}{})
}

func (g *KinesisGenerator) shouldLoadStreamChildren(streamName string) bool {
	streamResource := newKinesisStreamResource(streamName)
	for _, filter := range g.Filter {
		if filter.ServiceName != "" && filter.FieldPath == "id" && filter.IsApplicable("kinesis_stream") && !filter.Filter(streamResource) {
			return false
		}
	}
	return true
}

func (g *KinesisGenerator) loadStreamChildren(svc *kinesis.Client, streamName string) {
	output, err := svc.DescribeStreamSummary(context.TODO(), &kinesis.DescribeStreamSummaryInput{
		StreamName: &streamName,
	})
	if err != nil {
		if !kinesisResourceMissing(err) {
			log.Printf("Skipping Kinesis stream child resources for %s: %v", streamName, err)
		}
		return
	}
	if output.StreamDescriptionSummary == nil {
		return
	}
	streamARN := StringValue(output.StreamDescriptionSummary.StreamARN)
	if streamARN == "" {
		return
	}
	g.loadOptionalResources([]kinesisOptionalResourceLoader{
		{name: fmt.Sprintf("resource policy for stream %s", streamName), load: func() error {
			return g.loadResourcePolicy(svc, streamARN, kinesisResourceName(streamName, "policy"))
		}},
		{name: fmt.Sprintf("stream consumers for %s", streamName), load: func() error {
			return g.loadStreamConsumers(svc, streamName, streamARN)
		}},
	})
}

func (g *KinesisGenerator) loadStreamConsumers(svc *kinesis.Client, streamName string, streamARN string) error {
	p := kinesis.NewListStreamConsumersPaginator(svc, &kinesis.ListStreamConsumersInput{StreamARN: &streamARN})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, consumer := range page.Consumers {
			consumerARN := StringValue(consumer.ConsumerARN)
			consumerName := StringValue(consumer.ConsumerName)
			if consumerARN == "" || consumerName == "" || !kinesisConsumerImportable(consumer) {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				consumerARN,
				kinesisResourceName(streamName, consumerName),
				"aws_kinesis_stream_consumer",
				"aws",
				map[string]string{
					"name":       consumerName,
					"stream_arn": streamARN,
				},
				kinesisAllowEmptyValues,
				map[string]interface{}{},
			))
			g.loadOptionalResources([]kinesisOptionalResourceLoader{
				{name: fmt.Sprintf("resource policy for consumer %s", consumerName), load: func() error {
					return g.loadResourcePolicy(svc, consumerARN, kinesisResourceName(streamName, consumerName, "policy"))
				}},
			})
		}
	}
	return nil
}

func (g *KinesisGenerator) loadResourcePolicy(svc *kinesis.Client, resourceARN string, resourceName string) error {
	output, err := svc.GetResourcePolicy(context.TODO(), &kinesis.GetResourcePolicyInput{ResourceARN: &resourceARN})
	if err != nil {
		return err
	}
	policy := StringValue(output.Policy)
	if policy == "" {
		return nil
	}
	g.Resources = append(g.Resources, terraformutils.NewResource(
		resourceARN,
		resourceName,
		"aws_kinesis_resource_policy",
		"aws",
		map[string]string{
			"policy":       policy,
			"resource_arn": resourceARN,
		},
		kinesisAllowEmptyValues,
		map[string]interface{}{},
	))
	return nil
}

func kinesisConsumerImportable(consumer kinesistypes.Consumer) bool {
	return consumer.ConsumerStatus == kinesistypes.ConsumerStatusActive
}

func kinesisResourceName(parts ...string) string {
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			segments = append(segments, part)
		}
	}
	if len(segments) == 0 {
		return "kinesis_resource"
	}
	return strings.Join(segments, "/")
}

func kinesisResourceMissing(err error) bool {
	var resourceNotFound *kinesistypes.ResourceNotFoundException
	if errors.As(err, &resourceNotFound) {
		return true
	}
	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && strings.Contains(strings.ToLower(apiErr.ErrorCode()), "resourcenotfound")
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
		for _, streamName := range results.StreamNames {
			if streamName == "" || !g.shouldLoadStreamChildren(streamName) {
				continue
			}
			g.loadStreamChildren(svc, streamName)
		}

		if len(results.StreamNames) > 0 {
			request = kinesis.ListStreamsInput{
				ExclusiveStartStreamName: &results.StreamNames[len(results.StreamNames)-1],
			}
		}
	}
	return nil
}

func (g *KinesisGenerator) PostConvertHook() error {
	for i, resource := range g.Resources {
		if resource.InstanceInfo.Type != "aws_kinesis_resource_policy" {
			continue
		}
		policy, ok := resource.Item["policy"].(string)
		if !ok || policy == "" {
			continue
		}
		g.Resources[i].Item["policy"] = fmt.Sprintf("<<POLICY\n%s\nPOLICY", g.escapeAwsInterpolation(policy))
	}
	return nil
}
