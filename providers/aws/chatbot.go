// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/chatbot"
	chatbottypes "github.com/aws/aws-sdk-go-v2/service/chatbot/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	chatbotSlackChannelConfigurationResourceType = "aws_chatbot_slack_channel_configuration"
	chatbotDefaultRegion                         = "us-west-2"
)

var chatbotAllowEmptyValues = []string{"tags."}

type ChatbotGenerator struct {
	AWSService
}

func (g *ChatbotGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	config.Region = chatbotAPIRegion(config.Region)
	svc := chatbot.NewFromConfig(config)
	return g.loadSlackChannelConfigurations(svc)
}

func (g *ChatbotGenerator) loadSlackChannelConfigurations(svc *chatbot.Client) error {
	p := chatbot.NewDescribeSlackChannelConfigurationsPaginator(svc, &chatbot.DescribeSlackChannelConfigurationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			if chatbotNotFound(err) {
				return nil
			}
			return err
		}
		for _, configuration := range page.SlackChannelConfigurations {
			if resource, ok := newChatbotSlackChannelConfigurationResource(configuration); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newChatbotSlackChannelConfigurationResource(configuration chatbottypes.SlackChannelConfiguration) (terraformutils.Resource, bool) {
	arn := StringValue(configuration.ChatConfigurationArn)
	name := StringValue(configuration.ConfigurationName)
	iamRoleARN := StringValue(configuration.IamRoleArn)
	slackChannelID := StringValue(configuration.SlackChannelId)
	slackTeamID := StringValue(configuration.SlackTeamId)
	if arn == "" || name == "" || iamRoleARN == "" || slackChannelID == "" || slackTeamID == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"chat_configuration_arn": arn,
		"configuration_name":     name,
		"iam_role_arn":           iamRoleARN,
		"slack_channel_id":       slackChannelID,
		"slack_team_id":          slackTeamID,
	}
	if loggingLevel := StringValue(configuration.LoggingLevel); loggingLevel != "" {
		attributes["logging_level"] = loggingLevel
	}
	return terraformutils.NewResource(
		chatbotSlackChannelConfigurationImportID(arn),
		chatbotResourceName("slack_channel_configuration", name, arn),
		chatbotSlackChannelConfigurationResourceType,
		"aws",
		attributes,
		chatbotAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func chatbotSlackChannelConfigurationImportID(arn string) string {
	return arn
}

func chatbotResourceName(parts ...string) string {
	return resourceNameWithLengthPrefixes(parts...)
}

func chatbotAPIRegion(region string) string {
	switch region {
	case "us-east-2", "us-west-2", "eu-west-1", "ap-southeast-1":
		return region
	default:
		return chatbotDefaultRegion
	}
}

func chatbotNotFound(err error) bool {
	var notFound *chatbottypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
