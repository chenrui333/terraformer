// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appintegrations"
	appintegrationstypes "github.com/aws/aws-sdk-go-v2/service/appintegrations/types"
)

func TestAppIntegrationsImportIDs(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "data integration", got: appIntegrationsDataIntegrationImportID("data-123"), want: "data-123"},
		{name: "event integration", got: appIntegrationsEventIntegrationImportID("events"), want: "events"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("import ID = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestNewAppIntegrationsDataIntegrationResource(t *testing.T) {
	resource, ok := newAppIntegrationsDataIntegrationResource(&appintegrations.GetDataIntegrationOutput{
		Arn:         aws.String("arn:aws:app-integrations:us-east-1:123456789012:data-integration/data-123"),
		Description: aws.String("daily customer sync"),
		Id:          aws.String("data-123"),
		KmsKey:      aws.String("arn:aws:kms:us-east-1:123456789012:key/key-123"),
		Name:        aws.String("customer-sync"),
		ScheduleConfiguration: &appintegrationstypes.ScheduleConfiguration{
			FirstExecutionFrom: aws.String("2026-01-01T00:00:00Z"),
			Object:             aws.String("customers"),
			ScheduleExpression: aws.String("rate(1 day)"),
		},
		SourceURI: aws.String("s3://bucket/prefix"),
	})
	assertMessagingResource(t, resource, ok, appIntegrationsDataIntegrationResourceType, "data-123", map[string]string{
		"arn":         "arn:aws:app-integrations:us-east-1:123456789012:data-integration/data-123",
		"description": "daily customer sync",
		"kms_key":     "arn:aws:kms:us-east-1:123456789012:key/key-123",
		"name":        "customer-sync",
		"source_uri":  "s3://bucket/prefix",
	})
	scheduleConfig, ok := resource.AdditionalFields["schedule_config"].([]interface{})
	if !ok || len(scheduleConfig) != 1 {
		t.Fatalf("schedule_config additional field = %#v, want one block", resource.AdditionalFields["schedule_config"])
	}

	if _, ok := newAppIntegrationsDataIntegrationResource(&appintegrations.GetDataIntegrationOutput{
		Id:        aws.String("data-123"),
		KmsKey:    aws.String("arn:aws:kms:us-east-1:123456789012:key/key-123"),
		Name:      aws.String("customer-sync"),
		SourceURI: aws.String("s3://bucket/prefix"),
	}); ok {
		t.Fatal("data integration without schedule config should be skipped")
	}
}

func TestNewAppIntegrationsEventIntegrationResource(t *testing.T) {
	resource, ok := newAppIntegrationsEventIntegrationResource(&appintegrations.GetEventIntegrationOutput{
		Description:         aws.String("events"),
		EventBridgeBus:      aws.String("default"),
		EventFilter:         &appintegrationstypes.EventFilter{Source: aws.String("aws.partner/example")},
		EventIntegrationArn: aws.String("arn:aws:app-integrations:us-east-1:123456789012:event-integration/events"),
		Name:                aws.String("events"),
	})
	assertMessagingResource(t, resource, ok, appIntegrationsEventIntegrationResourceType, "events", map[string]string{
		"arn":             "arn:aws:app-integrations:us-east-1:123456789012:event-integration/events",
		"description":     "events",
		"eventbridge_bus": "default",
		"name":            "events",
	})
	eventFilter, ok := resource.AdditionalFields["event_filter"].([]interface{})
	if !ok || len(eventFilter) != 1 {
		t.Fatalf("event_filter additional field = %#v, want one block", resource.AdditionalFields["event_filter"])
	}

	if _, ok := newAppIntegrationsEventIntegrationResource(&appintegrations.GetEventIntegrationOutput{Name: aws.String("events")}); ok {
		t.Fatal("event integration without event filter/bus should be skipped")
	}
}
