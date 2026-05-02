// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var sqsAllowEmptyValues = []string{"tags."}

type SqsGenerator struct {
	AWSService
}

type sqsQueueAttributeResource struct {
	attributeName sqstypes.QueueAttributeName
	attributeKey  string
	resourceType  string
	serviceName   string
}

var sqsQueueAttributeResources = []sqsQueueAttributeResource{
	{
		attributeName: sqstypes.QueueAttributeNamePolicy,
		attributeKey:  "policy",
		resourceType:  "aws_sqs_queue_policy",
		serviceName:   "sqs_queue_policy",
	},
	{
		attributeName: sqstypes.QueueAttributeNameRedrivePolicy,
		attributeKey:  "redrive_policy",
		resourceType:  "aws_sqs_queue_redrive_policy",
		serviceName:   "sqs_queue_redrive_policy",
	},
	{
		attributeName: sqstypes.QueueAttributeNameRedriveAllowPolicy,
		attributeKey:  "redrive_allow_policy",
		resourceType:  "aws_sqs_queue_redrive_allow_policy",
		serviceName:   "sqs_queue_redrive_allow_policy",
	},
}

func (g *SqsGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := sqs.NewFromConfig(config)

	listQueuesInput := sqs.ListQueuesInput{}

	sqsPrefix, hasPrefix := os.LookupEnv("SQS_PREFIX")
	if hasPrefix {
		listQueuesInput.QueueNamePrefix = aws.String(sqsPrefix)
	}

	queuesList, err := svc.ListQueues(context.TODO(), &listQueuesInput)

	if err != nil {
		return err
	}

	for _, queueURL := range queuesList.QueueUrls {
		queueName := arnLastSegment(queueURL, "/")
		queueResource := newSqsQueueResource(queueURL, queueName)

		if g.shouldAppendQueueResource(queueResource) {
			g.Resources = append(g.Resources, queueResource)
		}

		if !g.shouldLoadQueueAttributeResources(queueResource) {
			continue
		}
		if err := g.addQueueAttributeResources(svc, queueURL, queueName); err != nil {
			if sqsQueueMissing(err) {
				continue
			}
			log.Printf("Skipping SQS queue attribute resources for %s: %v", queueURL, err)
		}
	}

	return nil
}

func newSqsQueueResource(queueURL, queueName string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		queueURL,
		queueName,
		"aws_sqs_queue",
		"aws",
		sqsAllowEmptyValues,
	)
}

func newSqsQueueAttributeResource(queueURL, queueName, value string, resource sqsQueueAttributeResource) terraformutils.Resource {
	return terraformutils.NewResource(
		queueURL,
		queueName,
		resource.resourceType,
		"aws",
		map[string]string{
			"queue_url":           queueURL,
			resource.attributeKey: value,
		},
		sqsAllowEmptyValues,
		map[string]interface{}{},
	)
}

func (g *SqsGenerator) addQueueAttributeResources(svc *sqs.Client, queueURL, queueName string) error {
	output, err := svc.GetQueueAttributes(context.TODO(), &sqs.GetQueueAttributesInput{
		AttributeNames: []sqstypes.QueueAttributeName{
			sqstypes.QueueAttributeNamePolicy,
			sqstypes.QueueAttributeNameRedrivePolicy,
			sqstypes.QueueAttributeNameRedriveAllowPolicy,
		},
		QueueUrl: aws.String(queueURL),
	})
	if err != nil {
		return err
	}
	if output == nil || len(output.Attributes) == 0 {
		return nil
	}

	for _, resource := range sqsQueueAttributeResources {
		value := output.Attributes[string(resource.attributeName)]
		if !sqsQueueAttributeConfigured(value) {
			continue
		}
		queueAttributeResource := newSqsQueueAttributeResource(queueURL, queueName, value, resource)
		if !g.shouldAppendQueueAttributeResource(resource, queueAttributeResource) {
			continue
		}
		g.Resources = append(g.Resources, queueAttributeResource)
	}

	return nil
}

func (g *SqsGenerator) shouldAppendQueueResource(queueResource terraformutils.Resource) bool {
	if !g.queueMatchesInitialIDFilters(queueResource) {
		return false
	}
	if g.hasTypedSqsChildFilter() && !g.hasTypedFilterFor("sqs_queue") && !g.hasUntypedIDFilter() {
		return false
	}
	return true
}

func (g *SqsGenerator) shouldLoadQueueAttributeResources(queueResource terraformutils.Resource) bool {
	if !g.queueMatchesInitialIDFilters(queueResource) {
		return false
	}
	if !g.hasTypedSqsChildFilter() {
		return !g.hasTypedNonIDQueueFilter()
	}
	return g.queueMatchesAnySqsChildInitialFilter(queueResource)
}

func (g *SqsGenerator) shouldAppendQueueAttributeResource(resource sqsQueueAttributeResource, queueAttributeResource terraformutils.Resource) bool {
	if !g.hasTypedSqsChildFilter() {
		return true
	}

	hasTypedResourceFilter := false
	for _, filter := range g.Filter {
		if filter.ServiceName == "" || !filter.IsApplicable(resource.serviceName) {
			continue
		}
		hasTypedResourceFilter = true
		if !filter.Filter(queueAttributeResource) {
			return false
		}
	}
	return hasTypedResourceFilter
}

func (g *SqsGenerator) queueMatchesInitialIDFilters(queueResource terraformutils.Resource) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("sqs_queue") {
			continue
		}
		if !filter.Filter(queueResource) {
			return false
		}
	}
	return true
}

func (g *SqsGenerator) queueMatchesAnySqsChildInitialFilter(queueResource terraformutils.Resource) bool {
	for _, resource := range sqsQueueAttributeResources {
		childResource := newSqsQueueAttributeResource(queueResource.InstanceState.ID, queueResource.ResourceName, "", resource)
		childHasFilter := false
		childMatches := true
		for _, filter := range g.Filter {
			if filter.ServiceName == "" || !filter.IsApplicable(resource.serviceName) {
				continue
			}
			childHasFilter = true
			if filter.FieldPath != "id" {
				return true
			}
			if !filter.Filter(childResource) {
				childMatches = false
				break
			}
		}
		if childHasFilter && childMatches {
			return true
		}
	}
	return false
}

func (g *SqsGenerator) hasTypedSqsChildFilter() bool {
	for _, resource := range sqsQueueAttributeResources {
		if g.hasTypedFilterFor(resource.serviceName) {
			return true
		}
	}
	return false
}

func (g *SqsGenerator) hasTypedFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName {
			return true
		}
	}
	return false
}

func (g *SqsGenerator) hasTypedNonIDQueueFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "sqs_queue" && filter.FieldPath != "id" {
			return true
		}
	}
	return false
}

func (g *SqsGenerator) hasUntypedIDFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "" && filter.FieldPath == "id" {
			return true
		}
	}
	return false
}

// PostConvertHook for add policy json as heredoc
func (g *SqsGenerator) PostConvertHook() error {
	splitAttributesByQueueURL := g.sqsSplitAttributesByQueueURL()

	for i, resource := range g.Resources {
		switch resource.InstanceInfo.Type {
		case "aws_sqs_queue":
			for attributeKey := range splitAttributesByQueueURL[resource.InstanceState.ID] {
				delete(g.Resources[i].Item, attributeKey)
			}
			g.wrapSqsJSONField(i, "policy")
		case "aws_sqs_queue_policy":
			g.wrapSqsJSONField(i, "policy")
		case "aws_sqs_queue_redrive_policy":
			g.wrapSqsJSONField(i, "redrive_policy")
		case "aws_sqs_queue_redrive_allow_policy":
			g.wrapSqsJSONField(i, "redrive_allow_policy")
		}
	}
	return nil
}

func (g *SqsGenerator) sqsSplitAttributesByQueueURL() map[string]map[string]struct{} {
	attributesByQueueURL := map[string]map[string]struct{}{}
	for _, resource := range g.Resources {
		var attributeKey string
		switch resource.InstanceInfo.Type {
		case "aws_sqs_queue_policy":
			attributeKey = "policy"
		case "aws_sqs_queue_redrive_policy":
			attributeKey = "redrive_policy"
		case "aws_sqs_queue_redrive_allow_policy":
			attributeKey = "redrive_allow_policy"
		default:
			continue
		}

		queueURL := resource.InstanceState.ID
		if queueURL == "" {
			continue
		}
		if _, ok := attributesByQueueURL[queueURL]; !ok {
			attributesByQueueURL[queueURL] = map[string]struct{}{}
		}
		attributesByQueueURL[queueURL][attributeKey] = struct{}{}
	}
	return attributesByQueueURL
}

func (g *SqsGenerator) wrapSqsJSONField(resourceIndex int, field string) {
	value, ok := g.Resources[resourceIndex].Item[field].(string)
	if !ok || value == "" {
		return
	}
	g.Resources[resourceIndex].Item[field] = fmt.Sprintf("<<POLICY\n%s\nPOLICY", g.escapeAwsInterpolation(value))
}

func sqsQueueAttributeConfigured(value string) bool {
	return value != ""
}

func sqsQueueMissing(err error) bool {
	var notFound *sqstypes.QueueDoesNotExist
	return errors.As(err, &notFound)
}
