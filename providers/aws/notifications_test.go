// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	notificationstypes "github.com/aws/aws-sdk-go-v2/service/notifications/types"
)

func TestNotificationsImportIDs(t *testing.T) {
	configurationARN := "arn:aws:notifications::123456789012:configuration/config-123"
	channelARN := "arn:aws:notificationscontacts::123456789012:emailcontact/contact-123"
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "notification configuration", got: notificationsNotificationConfigurationImportID(configurationARN), want: configurationARN},
		{name: "event rule", got: notificationsEventRuleImportID("arn:aws:notifications::123456789012:eventrule/rule-123"), want: "arn:aws:notifications::123456789012:eventrule/rule-123"},
		{name: "channel association", got: notificationsChannelAssociationImportID(configurationARN, channelARN), want: configurationARN + "," + channelARN},
		{name: "notification hub", got: notificationsNotificationHubImportID("us-east-1"), want: "us-east-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("import ID = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestNewNotificationsResources(t *testing.T) {
	configurationARN := "arn:aws:notifications::123456789012:configuration/config-123"
	configuration, ok := newNotificationsNotificationConfigurationResource(notificationstypes.NotificationConfigurationStructure{
		AggregationDuration: notificationstypes.AggregationDurationShort,
		Arn:                 aws.String(configurationARN),
		Description:         aws.String("primary notifications"),
		Name:                aws.String("primary"),
		Status:              notificationstypes.NotificationConfigurationStatusActive,
		Subtype:             notificationstypes.NotificationConfigurationSubtypeAccount,
	})
	assertMessagingResource(t, configuration, ok, notificationsNotificationConfigurationResourceType, configurationARN, map[string]string{
		"aggregation_duration": "SHORT",
		"arn":                  configurationARN,
		"description":          "primary notifications",
		"name":                 "primary",
	})

	ref := notificationsConfigurationReference{arn: configurationARN, name: "primary"}
	eventRule, ok := newNotificationsEventRuleResource(ref, notificationstypes.EventRuleStructure{
		Arn:          aws.String("arn:aws:notifications::123456789012:eventrule/rule-123"),
		EventPattern: aws.String("{\"source\":[\"aws.ec2\"]}"),
		EventType:    aws.String("EC2 Instance State-change Notification"),
		Regions:      []string{"us-east-1", "us-west-2"},
		Source:       aws.String("aws.ec2"),
	})
	assertMessagingResource(t, eventRule, ok, notificationsEventRuleResourceType, "arn:aws:notifications::123456789012:eventrule/rule-123", map[string]string{
		"event_pattern":                  "{\"source\":[\"aws.ec2\"]}",
		"event_type":                     "EC2 Instance State-change Notification",
		"notification_configuration_arn": configurationARN,
		"source":                         "aws.ec2",
	})
	regions, ok := eventRule.AdditionalFields["regions"].([]interface{})
	if !ok || len(regions) != 2 {
		t.Fatalf("regions additional field = %#v, want two regions", eventRule.AdditionalFields["regions"])
	}

	channelARN := "arn:aws:notificationscontacts::123456789012:emailcontact/contact-123"
	channelAssociation, ok := newNotificationsChannelAssociationResource(ref, channelARN)
	assertMessagingResource(t, channelAssociation, ok, notificationsChannelAssociationResourceType, configurationARN+","+channelARN, map[string]string{
		"arn":                            channelARN,
		"notification_configuration_arn": configurationARN,
	})

	hub, ok := newNotificationsNotificationHubResource(notificationstypes.NotificationHubOverview{
		NotificationHubRegion: aws.String("us-east-1"),
		StatusSummary:         &notificationstypes.NotificationHubStatusSummary{Status: notificationstypes.NotificationHubStatusActive},
	})
	assertMessagingResource(t, hub, ok, notificationsNotificationHubResourceType, "us-east-1", map[string]string{
		"notification_hub_region": "us-east-1",
	})

	hub, ok = newNotificationsNotificationHubResource(notificationstypes.NotificationHubOverview{
		NotificationHubRegion: aws.String("us-west-2"),
		StatusSummary:         &notificationstypes.NotificationHubStatusSummary{Status: notificationstypes.NotificationHubStatusInactive},
	})
	assertMessagingResource(t, hub, ok, notificationsNotificationHubResourceType, "us-west-2", map[string]string{
		"notification_hub_region": "us-west-2",
	})
}

func TestNewNotificationsNotificationConfigurationResourceAllowsEmptyDescription(t *testing.T) {
	configurationARN := "arn:aws:notifications::123456789012:configuration/config-blank-description"
	resource, ok := newNotificationsNotificationConfigurationResource(notificationstypes.NotificationConfigurationStructure{
		Arn:         aws.String(configurationARN),
		Description: aws.String(""),
		Name:        aws.String("blank-description"),
		Status:      notificationstypes.NotificationConfigurationStatusActive,
		Subtype:     notificationstypes.NotificationConfigurationSubtypeAccount,
	})
	assertMessagingResource(t, resource, ok, notificationsNotificationConfigurationResourceType, configurationARN, map[string]string{
		"arn":         configurationARN,
		"description": "",
		"name":        "blank-description",
	})
	if !notificationAllowsEmptyDescription(resource.AllowEmptyValues) {
		t.Fatalf("AllowEmptyValues = %#v, want description allowed", resource.AllowEmptyValues)
	}
}

func TestNotificationsResourceSkips(t *testing.T) {
	if _, ok := newNotificationsNotificationConfigurationResource(notificationstypes.NotificationConfigurationStructure{
		Arn:         aws.String("arn"),
		Description: aws.String("deleting"),
		Name:        aws.String("deleting"),
		Status:      notificationstypes.NotificationConfigurationStatusDeleting,
		Subtype:     notificationstypes.NotificationConfigurationSubtypeAccount,
	}); ok {
		t.Fatal("deleting notification configuration should be skipped")
	}
	if _, ok := newNotificationsNotificationConfigurationResource(notificationstypes.NotificationConfigurationStructure{
		Arn:         aws.String("arn"),
		Description: aws.String("admin-managed"),
		Name:        aws.String("admin-managed"),
		Status:      notificationstypes.NotificationConfigurationStatusActive,
		Subtype:     notificationstypes.NotificationConfigurationSubtypeAdminManaged,
	}); ok {
		t.Fatal("admin-managed notification configuration should be skipped")
	}
	if _, ok := newNotificationsNotificationConfigurationResource(notificationstypes.NotificationConfigurationStructure{
		Arn:     aws.String("arn"),
		Name:    aws.String("nil-description"),
		Status:  notificationstypes.NotificationConfigurationStatusActive,
		Subtype: notificationstypes.NotificationConfigurationSubtypeAccount,
	}); ok {
		t.Fatal("notification configuration with nil description should be skipped")
	}
	if _, ok := newNotificationsEventRuleResource(notificationsConfigurationReference{arn: "arn"}, notificationstypes.EventRuleStructure{
		Arn:       aws.String("rule"),
		EventType: aws.String("event"),
		Source:    aws.String("aws.ec2"),
	}); ok {
		t.Fatal("event rule without regions should be skipped")
	}
	if _, ok := newNotificationsChannelAssociationResource(notificationsConfigurationReference{arn: "arn"}, ""); ok {
		t.Fatal("channel association without channel ARN should be skipped")
	}
	if _, ok := newNotificationsNotificationHubResource(notificationstypes.NotificationHubOverview{
		NotificationHubRegion: aws.String("us-east-1"),
	}); ok {
		t.Fatal("notification hub without status summary should be skipped")
	}
}

func TestNotificationsListNotificationConfigurationsInputFiltersAccountOwned(t *testing.T) {
	input := notificationsListNotificationConfigurationsInput()
	if input.Subtype != notificationstypes.NotificationConfigurationSubtypeAccount {
		t.Fatalf("Subtype = %q, want %q", input.Subtype, notificationstypes.NotificationConfigurationSubtypeAccount)
	}
}

func notificationAllowsEmptyDescription(allowEmptyValues []string) bool {
	for _, value := range allowEmptyValues {
		if value == "description" {
			return true
		}
	}
	return false
}
