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

func TestChatbotAPIRegion(t *testing.T) {
	tests := []struct {
		name   string
		region string
		want   string
	}{
		{name: "supported us-east-2", region: "us-east-2", want: "us-east-2"},
		{name: "supported us-west-2", region: "us-west-2", want: "us-west-2"},
		{name: "supported eu-west-1", region: "eu-west-1", want: "eu-west-1"},
		{name: "supported ap-southeast-1", region: "ap-southeast-1", want: "ap-southeast-1"},
		{name: "unsupported us-east-1", region: "us-east-1", want: chatbotDefaultRegion},
		{name: "unsupported eu-central-1", region: "eu-central-1", want: chatbotDefaultRegion},
		{name: "empty", region: "", want: chatbotDefaultRegion},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chatbotAPIRegion(tt.region); got != tt.want {
				t.Fatalf("chatbotAPIRegion(%q) = %q, want %q", tt.region, got, tt.want)
			}
		})
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
