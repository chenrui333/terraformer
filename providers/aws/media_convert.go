// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	return g.loadQueues(svc)
}

func (g *MediaConvertGenerator) loadQueues(svc *mediaconvert.Client) error {
	p := mediaconvert.NewListQueuesPaginator(svc, &mediaconvert.ListQueuesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if mediaConvertQueueDiscoverySkippable(err) {
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

func mediaConvertQueueDiscoverySkippable(err error) bool {
	return mediaConvertQueueNotFound(err) || mediaConvertQueueEndpointUnavailable(err)
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
