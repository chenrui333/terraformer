// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	chatbottypes "github.com/aws/aws-sdk-go-v2/service/chatbot/types"
)

func TestChatbotImportIDs(t *testing.T) {
	arn := "arn:aws:chatbot::123456789012:chat-configuration/slack-channel/support"
	if got := chatbotSlackChannelConfigurationImportID(arn); got != arn {
		t.Fatalf("import ID = %q, want %q", got, arn)
	}
}

func TestNewChatbotSlackChannelConfigurationResource(t *testing.T) {
	arn := "arn:aws:chatbot::123456789012:chat-configuration/slack-channel/support"
	resource, ok := newChatbotSlackChannelConfigurationResource(chatbottypes.SlackChannelConfiguration{
		ChatConfigurationArn: aws.String(arn),
		ConfigurationName:    aws.String("support"),
		IamRoleArn:           aws.String("arn:aws:iam::123456789012:role/chatbot"),
		LoggingLevel:         aws.String("ERROR"),
		SlackChannelId:       aws.String("C123"),
		SlackTeamId:          aws.String("T123"),
	})
	assertMessagingResource(t, resource, ok, chatbotSlackChannelConfigurationResourceType, arn, map[string]string{
		"chat_configuration_arn": arn,
		"configuration_name":     "support",
		"iam_role_arn":           "arn:aws:iam::123456789012:role/chatbot",
		"logging_level":          "ERROR",
		"slack_channel_id":       "C123",
		"slack_team_id":          "T123",
	})

	if _, ok := newChatbotSlackChannelConfigurationResource(chatbottypes.SlackChannelConfiguration{ConfigurationName: aws.String("support")}); ok {
		t.Fatal("incomplete Slack channel configuration should be skipped")
	}
}
