// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/chimesdkvoice"
	chimetypes "github.com/aws/aws-sdk-go-v2/service/chimesdkvoice/types"
)

func TestChimeImportIDs(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "voice connector", got: chimeVoiceConnectorImportID("vc-123"), want: "vc-123"},
		{name: "voice connector group", got: chimeVoiceConnectorGroupImportID("vcg-123"), want: "vcg-123"},
		{name: "logging", got: chimeVoiceConnectorLoggingImportID("vc-123"), want: "vc-123"},
		{name: "origination", got: chimeVoiceConnectorOriginationImportID("vc-123"), want: "vc-123"},
		{name: "streaming", got: chimeVoiceConnectorStreamingImportID("vc-123"), want: "vc-123"},
		{name: "termination", got: chimeVoiceConnectorTerminationImportID("vc-123"), want: "vc-123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("import ID = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestNewChimeVoiceConnectorResource(t *testing.T) {
	resource, ok := newChimeVoiceConnectorResource(chimetypes.VoiceConnector{
		Name:              aws.String("support"),
		RequireEncryption: aws.Bool(true),
		VoiceConnectorId:  aws.String("vc-123"),
	})
	assertMessagingResource(t, resource, ok, chimeVoiceConnectorResourceType, "vc-123", map[string]string{
		"name":               "support",
		"require_encryption": "true",
	})

	if _, ok := newChimeVoiceConnectorResource(chimetypes.VoiceConnector{Name: aws.String("support")}); ok {
		t.Fatal("voice connector without ID/encryption should be skipped")
	}
}

func TestNewChimeVoiceConnectorGroupResource(t *testing.T) {
	resource, ok := newChimeVoiceConnectorGroupResource(chimetypes.VoiceConnectorGroup{
		Name:                  aws.String("group-a"),
		VoiceConnectorGroupId: aws.String("vcg-123"),
		VoiceConnectorItems: []chimetypes.VoiceConnectorItem{
			{Priority: aws.Int32(1), VoiceConnectorId: aws.String("vc-123")},
		},
	})
	assertMessagingResource(t, resource, ok, chimeVoiceConnectorGroupResourceType, "vcg-123", map[string]string{
		"name": "group-a",
	})
	connectors, ok := resource.AdditionalFields["connector"].([]interface{})
	if !ok || len(connectors) != 1 {
		t.Fatalf("connector additional field = %#v, want one connector", resource.AdditionalFields["connector"])
	}
	connector, ok := connectors[0].(map[string]interface{})
	if !ok {
		t.Fatalf("connector field type = %T, want map[string]interface{}", connectors[0])
	}
	if connector["voice_connector_id"] != "vc-123" || connector["priority"] != 1 {
		t.Fatalf("connector field = %#v", connector)
	}

	if _, ok := newChimeVoiceConnectorGroupResource(chimetypes.VoiceConnectorGroup{Name: aws.String("group-a")}); ok {
		t.Fatal("voice connector group without ID should be skipped")
	}
}

func TestNewChimeVoiceConnectorOptionalResources(t *testing.T) {
	logging, ok := newChimeVoiceConnectorLoggingResource("vc-123", &chimesdkvoice.GetVoiceConnectorLoggingConfigurationOutput{
		LoggingConfiguration: &chimetypes.LoggingConfiguration{EnableSIPLogs: aws.Bool(true)},
	})
	assertMessagingResource(t, logging, ok, chimeVoiceConnectorLoggingResourceType, "vc-123", map[string]string{
		"enable_sip_logs":    "true",
		"voice_connector_id": "vc-123",
	})

	origination, ok := newChimeVoiceConnectorOriginationResource("vc-123", &chimesdkvoice.GetVoiceConnectorOriginationOutput{
		Origination: &chimetypes.Origination{
			Disabled: aws.Bool(false),
			Routes: []chimetypes.OriginationRoute{
				{
					Host:     aws.String("sip.example.com"),
					Port:     aws.Int32(5060),
					Priority: aws.Int32(1),
					Protocol: chimetypes.OriginationRouteProtocolTcp,
					Weight:   aws.Int32(10),
				},
			},
		},
	})
	assertMessagingResource(t, origination, ok, chimeVoiceConnectorOriginationResourceType, "vc-123", map[string]string{
		"disabled":           "false",
		"voice_connector_id": "vc-123",
	})
	routes, ok := origination.AdditionalFields["route"].([]interface{})
	if !ok || len(routes) != 1 {
		t.Fatalf("route additional field = %#v, want one route", origination.AdditionalFields["route"])
	}

	streaming, ok := newChimeVoiceConnectorStreamingResource("vc-123", &chimesdkvoice.GetVoiceConnectorStreamingConfigurationOutput{
		StreamingConfiguration: &chimetypes.StreamingConfiguration{DataRetentionInHours: aws.Int32(24), Disabled: aws.Bool(false)},
	})
	assertMessagingResource(t, streaming, ok, chimeVoiceConnectorStreamingResourceType, "vc-123", map[string]string{
		"data_retention":     "24",
		"disabled":           "false",
		"voice_connector_id": "vc-123",
	})

	termination, ok := newChimeVoiceConnectorTerminationResource("vc-123", &chimesdkvoice.GetVoiceConnectorTerminationOutput{
		Termination: &chimetypes.Termination{
			CallingRegions:     []string{"US"},
			CidrAllowedList:    []string{"203.0.113.0/24"},
			CpsLimit:           aws.Int32(2),
			DefaultPhoneNumber: aws.String("+12065550100"),
			Disabled:           aws.Bool(false),
		},
	})
	assertMessagingResource(t, termination, ok, chimeVoiceConnectorTerminationResourceType, "vc-123", map[string]string{
		"cps_limit":            "2",
		"default_phone_number": "+12065550100",
		"disabled":             "false",
		"voice_connector_id":   "vc-123",
	})
	if len(termination.AdditionalFields["calling_regions"].([]interface{})) != 1 {
		t.Fatalf("calling_regions additional field = %#v", termination.AdditionalFields["calling_regions"])
	}
	if len(termination.AdditionalFields["cidr_allow_list"].([]interface{})) != 1 {
		t.Fatalf("cidr_allow_list additional field = %#v", termination.AdditionalFields["cidr_allow_list"])
	}
}

func TestChimeOptionalResourceSkips(t *testing.T) {
	if _, ok := newChimeVoiceConnectorLoggingResource("", &chimesdkvoice.GetVoiceConnectorLoggingConfigurationOutput{}); ok {
		t.Fatal("logging without connector ID should be skipped")
	}
	if _, ok := newChimeVoiceConnectorOriginationResource("vc-123", &chimesdkvoice.GetVoiceConnectorOriginationOutput{
		Origination: &chimetypes.Origination{Routes: []chimetypes.OriginationRoute{{Host: aws.String("sip.example.com")}}},
	}); ok {
		t.Fatal("origination without complete route fields should be skipped")
	}
	if _, ok := newChimeVoiceConnectorTerminationResource("vc-123", &chimesdkvoice.GetVoiceConnectorTerminationOutput{
		Termination: &chimetypes.Termination{CallingRegions: []string{"US"}},
	}); ok {
		t.Fatal("termination without CIDR allow list should be skipped")
	}
}
