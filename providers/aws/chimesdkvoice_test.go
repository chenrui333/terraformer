// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/chimesdkvoice"
	chimesdkvoicetypes "github.com/aws/aws-sdk-go-v2/service/chimesdkvoice/types"
)

func TestChimeSDKVoiceImportIDs(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "global settings", got: chimeSDKVoiceGlobalSettingsImportID("123456789012"), want: "123456789012"},
		{name: "SIP media application", got: chimeSDKVoiceSIPMediaApplicationImportID("sma-123"), want: "sma-123"},
		{name: "SIP rule", got: chimeSDKVoiceSIPRuleImportID("rule-123"), want: "rule-123"},
		{name: "voice profile domain", got: chimeSDKVoiceVoiceProfileDomainImportID("domain-123"), want: "domain-123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("import ID = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestNewChimeSDKVoiceGlobalSettingsResource(t *testing.T) {
	resource, ok := newChimeSDKVoiceGlobalSettingsResource("123456789012", &chimesdkvoice.GetGlobalSettingsOutput{
		VoiceConnector: &chimesdkvoicetypes.VoiceConnectorSettings{CdrBucket: aws.String("cdr-bucket")},
	})
	assertMessagingResource(t, resource, ok, chimeSDKVoiceGlobalSettingsResourceType, "123456789012", map[string]string{})
	voiceConnector, ok := resource.AdditionalFields["voice_connector"].([]interface{})
	if !ok || len(voiceConnector) != 1 {
		t.Fatalf("voice_connector additional field = %#v, want one block", resource.AdditionalFields["voice_connector"])
	}

	if _, ok := newChimeSDKVoiceGlobalSettingsResource("", &chimesdkvoice.GetGlobalSettingsOutput{}); ok {
		t.Fatal("global settings without account ID should be skipped")
	}
	if _, ok := newChimeSDKVoiceGlobalSettingsResource("123456789012", &chimesdkvoice.GetGlobalSettingsOutput{}); ok {
		t.Fatal("global settings without CDR bucket should be skipped")
	}
}

func TestChimeSDKVoiceShouldLoadGlobalSettings(t *testing.T) {
	tests := []struct {
		name   string
		region string
		want   bool
	}{
		{name: "default import", region: NoRegion, want: true},
		{name: "canonical public partition region", region: MainRegionPublicPartition, want: true},
		{name: "regional import", region: "us-west-2", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chimeSDKVoiceShouldLoadGlobalSettings(tt.region); got != tt.want {
				t.Fatalf("should load global settings = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestNewChimeSDKVoiceSIPMediaApplicationResource(t *testing.T) {
	resource, ok := newChimeSDKVoiceSIPMediaApplicationResource(&chimesdkvoice.GetSipMediaApplicationOutput{
		SipMediaApplication: &chimesdkvoicetypes.SipMediaApplication{
			AwsRegion:              aws.String("us-east-1"),
			Endpoints:              []chimesdkvoicetypes.SipMediaApplicationEndpoint{{LambdaArn: aws.String("arn:aws:lambda:us-east-1:123456789012:function:handler")}},
			Name:                   aws.String("media-app"),
			SipMediaApplicationArn: aws.String("arn:aws:chime:us-east-1:123456789012:sma/sma-123"),
			SipMediaApplicationId:  aws.String("sma-123"),
		},
	})
	assertMessagingResource(t, resource, ok, chimeSDKVoiceSIPMediaApplicationResourceType, "sma-123", map[string]string{
		"arn":                           "arn:aws:chime:us-east-1:123456789012:sma/sma-123",
		chimeSDKVoiceAWSRegionAttribute: "us-east-1",
		"name":                          "media-app",
	})
	endpoints, ok := resource.AdditionalFields["endpoints"].([]interface{})
	if !ok || len(endpoints) != 1 {
		t.Fatalf("endpoints additional field = %#v, want one endpoint", resource.AdditionalFields["endpoints"])
	}

	if _, ok := newChimeSDKVoiceSIPMediaApplicationResource(&chimesdkvoice.GetSipMediaApplicationOutput{
		SipMediaApplication: &chimesdkvoicetypes.SipMediaApplication{
			AwsRegion:             aws.String("us-east-1"),
			Name:                  aws.String("media-app"),
			SipMediaApplicationId: aws.String("sma-123"),
		},
	}); ok {
		t.Fatal("SIP media application without endpoint should be skipped")
	}
}

func TestNewChimeSDKVoiceSIPRuleResource(t *testing.T) {
	resource, ok := newChimeSDKVoiceSIPRuleResource(&chimesdkvoice.GetSipRuleOutput{
		SipRule: &chimesdkvoicetypes.SipRule{
			Disabled:     aws.Bool(false),
			Name:         aws.String("phone-rule"),
			SipRuleId:    aws.String("rule-123"),
			TriggerType:  chimesdkvoicetypes.SipRuleTriggerTypeToPhoneNumber,
			TriggerValue: aws.String("+12065550100"),
			TargetApplications: []chimesdkvoicetypes.SipRuleTargetApplication{
				{AwsRegion: aws.String("us-east-1"), Priority: aws.Int32(1), SipMediaApplicationId: aws.String("sma-123")},
			},
		},
	})
	assertMessagingResource(t, resource, ok, chimeSDKVoiceSIPRuleResourceType, "rule-123", map[string]string{
		"disabled":      "false",
		"name":          "phone-rule",
		"trigger_type":  "ToPhoneNumber",
		"trigger_value": "+12065550100",
	})
	targetApplications, ok := resource.AdditionalFields["target_applications"].([]interface{})
	if !ok || len(targetApplications) != 1 {
		t.Fatalf("target_applications additional field = %#v, want one target", resource.AdditionalFields["target_applications"])
	}

	if _, ok := newChimeSDKVoiceSIPRuleResource(&chimesdkvoice.GetSipRuleOutput{
		SipRule: &chimesdkvoicetypes.SipRule{
			Name:         aws.String("phone-rule"),
			SipRuleId:    aws.String("rule-123"),
			TriggerType:  chimesdkvoicetypes.SipRuleTriggerTypeToPhoneNumber,
			TriggerValue: aws.String("+12065550100"),
		},
	}); ok {
		t.Fatal("SIP rule without target application should be skipped")
	}
}

func TestNewChimeSDKVoiceVoiceProfileDomainResource(t *testing.T) {
	resource, ok := newChimeSDKVoiceVoiceProfileDomainResource(&chimesdkvoice.GetVoiceProfileDomainOutput{
		VoiceProfileDomain: &chimesdkvoicetypes.VoiceProfileDomain{
			Description:                       aws.String("voice profiles"),
			Name:                              aws.String("profiles"),
			ServerSideEncryptionConfiguration: &chimesdkvoicetypes.ServerSideEncryptionConfiguration{KmsKeyArn: aws.String("arn:aws:kms:us-east-1:123456789012:key/key-123")},
			VoiceProfileDomainArn:             aws.String("arn:aws:chime:us-east-1:123456789012:voice-profile-domain/domain-123"),
			VoiceProfileDomainId:              aws.String("domain-123"),
		},
	})
	assertMessagingResource(t, resource, ok, chimeSDKVoiceVoiceProfileDomainResourceType, "domain-123", map[string]string{
		"arn":         "arn:aws:chime:us-east-1:123456789012:voice-profile-domain/domain-123",
		"description": "voice profiles",
		"id":          "domain-123",
		"name":        "profiles",
	})
	encryptionConfig, ok := resource.AdditionalFields["server_side_encryption_configuration"].([]interface{})
	if !ok || len(encryptionConfig) != 1 {
		t.Fatalf("server_side_encryption_configuration additional field = %#v, want one block", resource.AdditionalFields["server_side_encryption_configuration"])
	}

	if _, ok := newChimeSDKVoiceVoiceProfileDomainResource(&chimesdkvoice.GetVoiceProfileDomainOutput{
		VoiceProfileDomain: &chimesdkvoicetypes.VoiceProfileDomain{Name: aws.String("profiles"), VoiceProfileDomainId: aws.String("domain-123")},
	}); ok {
		t.Fatal("voice profile domain without encryption config should be skipped")
	}
}
