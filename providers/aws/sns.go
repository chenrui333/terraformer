// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	snsTopicResourceType             = "aws_sns_topic"
	snsTopicDataProtectionPolicyType = "aws_sns_topic_data_protection_policy"
	snsTopicPolicyResourceType       = "aws_sns_topic_policy"
	snsTopicSubscriptionResourceType = "aws_sns_topic_subscription"
	snsTopicAttributeOwner           = "Owner"
	snsTopicAttributePolicy          = "Policy"
	snsTopicAttributeTopicARN        = "TopicArn"
	snsTopicPolicyName               = "topic_policy"
	snsTopicDataProtectionPolicyName = "data_protection_policy"
)

var snsAllowEmptyValues = []string{"tags."}

type SnsGenerator struct {
	AWSService
}

// TF currently doesn't support email subscriptions + subscriptions with pending confirmations
func (g *SnsGenerator) isSupportedSubscription(protocol, subscriptionID string) bool {
	return protocol != "email" && protocol != "email-json" && subscriptionID != "PendingConfirmation"
}

func (g *SnsGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := sns.NewFromConfig(config)
	p := sns.NewListTopicsPaginator(svc, &sns.ListTopicsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, topic := range page.Topics {
			topicARN := StringValue(topic.TopicArn)
			topicName := arnLastSegment(topicARN, ":")

			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				topicARN,
				topicName,
				snsTopicResourceType,
				"aws",
				snsAllowEmptyValues,
			))
			if err := g.loadTopicPolicies(svc, topicARN, topicName); err != nil {
				return err
			}

			topicSubsPage := sns.NewListSubscriptionsByTopicPaginator(svc, &sns.ListSubscriptionsByTopicInput{
				TopicArn: topic.TopicArn,
			})
			for topicSubsPage.HasMorePages() {
				topicSubsNextPage, err := topicSubsPage.NextPage(context.TODO())
				if err != nil {
					return fmt.Errorf("list SNS subscriptions for topic %s: %w", StringValue(topic.TopicArn), err)
				}
				for _, subscription := range topicSubsNextPage.Subscriptions {
					subscriptionID := arnLastSegment(StringValue(subscription.SubscriptionArn), ":")

					if g.isSupportedSubscription(StringValue(subscription.Protocol), subscriptionID) {
						g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
							StringValue(subscription.SubscriptionArn),
							"subscription-"+subscriptionID,
							snsTopicSubscriptionResourceType,
							"aws",
							snsAllowEmptyValues,
						))
					}
				}
			}
		}
	}
	return nil
}

func (g *SnsGenerator) loadTopicPolicies(svc *sns.Client, topicARN, topicName string) error {
	if topicARN == "" {
		return nil
	}
	attributes, err := svc.GetTopicAttributes(context.TODO(), &sns.GetTopicAttributesInput{
		TopicArn: &topicARN,
	})
	if err != nil {
		return fmt.Errorf("get SNS topic attributes for %s: %w", topicARN, err)
	}
	if resource, ok := newSNSTopicPolicyResource(topicARN, topicName, attributes); ok {
		g.Resources = append(g.Resources, resource)
	}

	dataProtectionPolicy, err := svc.GetDataProtectionPolicy(context.TODO(), &sns.GetDataProtectionPolicyInput{
		ResourceArn: &topicARN,
	})
	if err != nil {
		if snsDataProtectionPolicyNotFound(err) {
			return nil
		}
		return fmt.Errorf("get SNS topic data protection policy for %s: %w", topicARN, err)
	}
	if resource, ok := newSNSTopicDataProtectionPolicyResource(topicARN, topicName, dataProtectionPolicy); ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func newSNSTopicPolicyResource(topicARN, topicName string, output *sns.GetTopicAttributesOutput) (terraformutils.Resource, bool) {
	if topicARN == "" || output == nil || output.Attributes == nil {
		return terraformutils.Resource{}, false
	}
	policy := output.Attributes[snsTopicAttributePolicy]
	if policy == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"arn":    output.Attributes[snsTopicAttributeTopicARN],
		"policy": policy,
	}
	if attributes["arn"] == "" {
		attributes["arn"] = topicARN
	}
	if owner := output.Attributes[snsTopicAttributeOwner]; owner != "" {
		attributes["owner"] = owner
	}
	return terraformutils.NewResource(
		snsTopicPolicyImportID(topicARN),
		snsResourceName(snsTopicPolicyName, topicName, topicARN),
		snsTopicPolicyResourceType,
		"aws",
		attributes,
		snsAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newSNSTopicDataProtectionPolicyResource(topicARN, topicName string, output *sns.GetDataProtectionPolicyOutput) (terraformutils.Resource, bool) {
	if topicARN == "" || output == nil || StringValue(output.DataProtectionPolicy) == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		snsTopicDataProtectionPolicyImportID(topicARN),
		snsResourceName(snsTopicDataProtectionPolicyName, topicName, topicARN),
		snsTopicDataProtectionPolicyType,
		"aws",
		map[string]string{
			"arn":    topicARN,
			"policy": StringValue(output.DataProtectionPolicy),
		},
		snsAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func snsTopicPolicyImportID(topicARN string) string {
	return topicARN
}

func snsTopicDataProtectionPolicyImportID(topicARN string) string {
	return topicARN
}

func snsResourceName(parts ...string) string {
	return resourceNameWithLengthPrefixes(parts...)
}

func snsDataProtectionPolicyNotFound(err error) bool {
	var notFound *snstypes.NotFoundException
	return errors.As(err, &notFound)
}

// PostConvertHook for add policy json as heredoc
func (g *SnsGenerator) PostConvertHook() error {
	splitPolicyTopicARNs := snsSplitTopicPolicyARNs(g.Resources)
	for i, resource := range g.Resources {
		if resource.InstanceInfo == nil {
			continue
		}
		switch resource.InstanceInfo.Type {
		case snsTopicResourceType:
			if resource.InstanceState != nil && splitPolicyTopicARNs[resource.InstanceState.ID] {
				delete(g.Resources[i].Item, "policy")
			}
			g.wrapSNSPolicyField(i, "policy")
		case snsTopicPolicyResourceType, snsTopicDataProtectionPolicyType:
			g.wrapSNSPolicyField(i, "policy")
		}
	}
	return nil
}

func (g *SnsGenerator) wrapSNSPolicyField(resourceIndex int, field string) {
	val, ok := g.Resources[resourceIndex].Item[field].(string)
	if !ok || val == "" {
		return
	}
	policy := g.escapeAwsInterpolation(val)
	g.Resources[resourceIndex].Item[field] = fmt.Sprintf(`<<POLICY
%s
POLICY`, policy)
}

func snsSplitTopicPolicyARNs(resources []terraformutils.Resource) map[string]bool {
	arns := map[string]bool{}
	for _, resource := range resources {
		if resource.InstanceInfo == nil || resource.InstanceInfo.Type != snsTopicPolicyResourceType || resource.InstanceState == nil {
			continue
		}
		arns[resource.InstanceState.ID] = true
	}
	return arns
}
