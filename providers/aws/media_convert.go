// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/mediaconvert"
	mediaconverttypes "github.com/aws/aws-sdk-go-v2/service/mediaconvert/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const mediaConvertQueueResourceType = "aws_media_convert_queue"

var mediaConvertAllowEmptyValues = []string{"tags."}

type MediaConvertGenerator struct {
	AWSService
}

func (g *MediaConvertGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := mediaconvert.NewFromConfig(config)
	if err := g.loadQueues(svc); err != nil {
		if !mediaConvertQueueEndpointUnavailable(err) {
			return err
		}
		// Regional endpoints are preferred, but older accounts can require an
		// existing account endpoint. GET_ONLY avoids creating one during discovery.
		endpoint, endpointErr := mediaConvertAccountEndpoint(context.TODO(), svc)
		if endpointErr != nil {
			return endpointErr
		}
		if endpoint == "" {
			return nil
		}
		return g.loadQueues(mediaconvert.NewFromConfig(config, func(o *mediaconvert.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		}))
	}
	return nil
}

func (g *MediaConvertGenerator) loadQueues(svc *mediaconvert.Client) error {
	p := mediaconvert.NewListQueuesPaginator(svc, &mediaconvert.ListQueuesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if mediaConvertQueueNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, queue := range page.Queues {
			if resource, ok := newMediaConvertQueueResource(queue); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}

	return nil
}

func mediaConvertAccountEndpoint(ctx context.Context, svc mediaconvert.DescribeEndpointsAPIClient) (string, error) {
	p := mediaconvert.NewDescribeEndpointsPaginator(svc, &mediaconvert.DescribeEndpointsInput{
		Mode: mediaconverttypes.DescribeEndpointsModeGetOnly,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if mediaConvertQueueNotFound(err) {
			return "", nil
		}
		if err != nil {
			return "", err
		}
		for _, endpoint := range page.Endpoints {
			if endpointURL := StringValue(endpoint.Url); endpointURL != "" {
				return endpointURL, nil
			}
		}
	}
	return "", nil
}

func newMediaConvertQueueResource(queue mediaconverttypes.Queue) (terraformutils.Resource, bool) {
	queueName := StringValue(queue.Name)
	if queueName == "" || !mediaConvertQueueImportable(queue) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		mediaConvertQueueImportID(queueName),
		mediaConvertResourceName("queue", queueName),
		mediaConvertQueueResourceType,
		"aws",
		map[string]string{
			"name": queueName,
		},
		mediaConvertAllowEmptyValues,
		map[string]interface{}{}), true
}

func mediaConvertQueueImportable(queue mediaconverttypes.Queue) bool {
	return queue.Type != mediaconverttypes.TypeSystem
}

func mediaConvertQueueImportID(queueName string) string {
	return queueName
}

func mediaConvertResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d/%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return "media-convert-resource"
	}
	return strings.Join(cleanParts, "/")
}

func mediaConvertQueueNotFound(err error) bool {
	var notFound *mediaconverttypes.NotFoundException
	return errors.As(err, &notFound)
}

func mediaConvertQueueEndpointUnavailable(err error) bool {
	var badRequest *mediaconverttypes.BadRequestException
	if !errors.As(err, &badRequest) {
		return false
	}
	message := strings.ToLower(StringValue(badRequest.Message))
	return strings.Contains(message, "endpoint") &&
		(strings.Contains(message, "customer") || strings.Contains(message, "account"))
}
