// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/notifications"
	notificationstypes "github.com/aws/aws-sdk-go-v2/service/notifications/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	notificationsChannelAssociationResourceType        = "aws_notifications_channel_association"
	notificationsEventRuleResourceType                 = "aws_notifications_event_rule"
	notificationsNotificationConfigurationResourceType = "aws_notifications_notification_configuration"
	notificationsNotificationHubResourceType           = "aws_notifications_notification_hub"
	notificationsResourceIDSeparator                   = ","
)

var notificationsAllowEmptyValues = []string{"tags."}

type NotificationsGenerator struct {
	AWSService
}

type notificationsConfigurationReference struct {
	arn  string
	name string
}

type notificationsOptionalResourceLoader struct {
	name string
	load func() error
}

func (g *NotificationsGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := notifications.NewFromConfig(config)
	if err := g.loadNotificationHubs(svc); err != nil {
		return err
	}
	configurations, err := g.loadNotificationConfigurations(svc)
	if err != nil {
		return err
	}
	for _, configuration := range configurations {
		g.getOptionalNotificationsResources(
			notificationsOptionalResourceLoader{name: "event rules", load: func() error {
				return g.loadEventRules(svc, configuration)
			}},
			notificationsOptionalResourceLoader{name: "channel associations", load: func() error {
				return g.loadChannelAssociations(svc, configuration)
			}},
		)
	}
	return nil
}

func (g *NotificationsGenerator) loadNotificationConfigurations(svc *notifications.Client) ([]notificationsConfigurationReference, error) {
	configurations := []notificationsConfigurationReference{}
	p := notifications.NewListNotificationConfigurationsPaginator(svc, &notifications.ListNotificationConfigurationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		for _, configuration := range page.NotificationConfigurations {
			resource, ok := newNotificationsNotificationConfigurationResource(configuration)
			if !ok {
				continue
			}
			g.Resources = append(g.Resources, resource)
			configurations = append(configurations, notificationsConfigurationReference{
				arn:  StringValue(configuration.Arn),
				name: StringValue(configuration.Name),
			})
		}
	}
	return configurations, nil
}

func (g *NotificationsGenerator) loadEventRules(svc *notifications.Client, configuration notificationsConfigurationReference) error {
	if configuration.arn == "" {
		return nil
	}
	p := notifications.NewListEventRulesPaginator(svc, &notifications.ListEventRulesInput{
		NotificationConfigurationArn: &configuration.arn,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			if notificationsNotFound(err) {
				return nil
			}
			return err
		}
		for _, eventRule := range page.EventRules {
			if resource, ok := newNotificationsEventRuleResource(configuration, eventRule); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *NotificationsGenerator) loadChannelAssociations(svc *notifications.Client, configuration notificationsConfigurationReference) error {
	if configuration.arn == "" {
		return nil
	}
	p := notifications.NewListChannelsPaginator(svc, &notifications.ListChannelsInput{
		NotificationConfigurationArn: &configuration.arn,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			if notificationsNotFound(err) {
				return nil
			}
			return err
		}
		for _, channelARN := range page.Channels {
			if resource, ok := newNotificationsChannelAssociationResource(configuration, channelARN); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *NotificationsGenerator) loadNotificationHubs(svc *notifications.Client) error {
	p := notifications.NewListNotificationHubsPaginator(svc, &notifications.ListNotificationHubsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, hub := range page.NotificationHubs {
			if resource, ok := newNotificationsNotificationHubResource(hub); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *NotificationsGenerator) getOptionalNotificationsResources(loaders ...notificationsOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			log.Printf("skipping User Notifications %s discovery: %v", loader.name, err)
		}
	}
}

func newNotificationsNotificationConfigurationResource(configuration notificationstypes.NotificationConfigurationStructure) (terraformutils.Resource, bool) {
	arn := StringValue(configuration.Arn)
	name := StringValue(configuration.Name)
	description := StringValue(configuration.Description)
	if arn == "" || name == "" || description == "" || !notificationsConfigurationImportable(configuration) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"arn":         arn,
		"name":        name,
		"description": description,
	}
	if configuration.AggregationDuration != "" {
		attributes["aggregation_duration"] = string(configuration.AggregationDuration)
	}
	return terraformutils.NewResource(
		notificationsNotificationConfigurationImportID(arn),
		notificationsResourceName("notification_configuration", name, arn),
		notificationsNotificationConfigurationResourceType,
		"aws",
		attributes,
		notificationsAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newNotificationsEventRuleResource(configuration notificationsConfigurationReference, rule notificationstypes.EventRuleStructure) (terraformutils.Resource, bool) {
	arn := StringValue(rule.Arn)
	source := StringValue(rule.Source)
	eventType := StringValue(rule.EventType)
	if configuration.arn == "" || arn == "" || source == "" || eventType == "" || len(rule.Regions) == 0 {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"arn":                            arn,
		"notification_configuration_arn": configuration.arn,
		"source":                         source,
		"event_type":                     eventType,
	}
	if eventPattern := StringValue(rule.EventPattern); eventPattern != "" {
		attributes["event_pattern"] = eventPattern
	}
	return terraformutils.NewResource(
		notificationsEventRuleImportID(arn),
		notificationsResourceName("event_rule", configuration.name, source, eventType, arn),
		notificationsEventRuleResourceType,
		"aws",
		attributes,
		notificationsAllowEmptyValues,
		map[string]interface{}{
			"regions": stringSliceToInterfaceSlice(rule.Regions),
		},
	), true
}

func newNotificationsChannelAssociationResource(configuration notificationsConfigurationReference, channelARN string) (terraformutils.Resource, bool) {
	if configuration.arn == "" || channelARN == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		notificationsChannelAssociationImportID(configuration.arn, channelARN),
		notificationsResourceName("channel_association", configuration.name, channelARN),
		notificationsChannelAssociationResourceType,
		"aws",
		map[string]string{
			"arn":                            channelARN,
			"notification_configuration_arn": configuration.arn,
		},
		notificationsAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newNotificationsNotificationHubResource(hub notificationstypes.NotificationHubOverview) (terraformutils.Resource, bool) {
	region := StringValue(hub.NotificationHubRegion)
	if region == "" || !notificationsHubImportable(hub) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		notificationsNotificationHubImportID(region),
		notificationsResourceName("notification_hub", region),
		notificationsNotificationHubResourceType,
		"aws",
		map[string]string{"notification_hub_region": region},
		notificationsAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func notificationsConfigurationImportable(configuration notificationstypes.NotificationConfigurationStructure) bool {
	return configuration.Status != notificationstypes.NotificationConfigurationStatusDeleting
}

func notificationsHubImportable(hub notificationstypes.NotificationHubOverview) bool {
	return hub.StatusSummary != nil && hub.StatusSummary.Status == notificationstypes.NotificationHubStatusActive
}

func notificationsNotificationConfigurationImportID(arn string) string {
	return arn
}

func notificationsEventRuleImportID(arn string) string {
	return arn
}

func notificationsChannelAssociationImportID(notificationConfigurationARN, channelARN string) string {
	return strings.Join([]string{notificationConfigurationARN, channelARN}, notificationsResourceIDSeparator)
}

func notificationsNotificationHubImportID(region string) string {
	return region
}

func notificationsResourceName(parts ...string) string {
	return resourceNameWithLengthPrefixes(parts...)
}

func notificationsNotFound(err error) bool {
	var notFound *notificationstypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
