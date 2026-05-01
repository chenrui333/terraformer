// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchevents"
	"github.com/chenrui333/terraformer/terraformutils"
)

var cloudwatchAllowEmptyValues = []string{"tags."}

const defaultEventBusName = "default"

type cloudwatchOptionalResourceLoader struct {
	name string
	load func() error
}

type CloudWatchGenerator struct {
	AWSService
}

func (g *CloudWatchGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}

	cloudwatchSvc := cloudwatch.NewFromConfig(config)
	err := g.createMetricAlarms(cloudwatchSvc)
	if err != nil {
		return err
	}
	err = g.createDashboards(cloudwatchSvc)
	if err != nil {
		return err
	}

	eventsSvc := cloudwatchevents.NewFromConfig(config)
	eventBusNames, err := g.createEventBuses(eventsSvc)
	if err != nil {
		log.Printf("skipping EventBridge event bus discovery: %v", err)
		eventBusNames = []string{defaultEventBusName}
	}
	err = g.createRules(eventsSvc, eventBusNames)
	if err != nil {
		return err
	}
	g.getOptionalCloudWatchResources(
		cloudwatchOptionalResourceLoader{name: "EventBridge archives", load: func() error { return g.createArchives(eventsSvc) }},
		cloudwatchOptionalResourceLoader{name: "EventBridge connections", load: func() error { return g.createConnections(eventsSvc) }},
		cloudwatchOptionalResourceLoader{name: "EventBridge API destinations", load: func() error { return g.createAPIDestinations(eventsSvc) }},
	)

	return nil
}

func (g *CloudWatchGenerator) getOptionalCloudWatchResources(loaders ...cloudwatchOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			log.Printf("skipping %s discovery: %v", loader.name, err)
		}
	}
}

func (g *CloudWatchGenerator) createMetricAlarms(cloudwatchSvc *cloudwatch.Client) error {
	var nextToken *string
	for {
		output, err := cloudwatchSvc.DescribeAlarms(context.TODO(), &cloudwatch.DescribeAlarmsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return err
		}
		for _, metricAlarm := range output.MetricAlarms {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				*metricAlarm.AlarmName,
				*metricAlarm.AlarmName,
				"aws_cloudwatch_metric_alarm",
				"aws",
				cloudwatchAllowEmptyValues))
		}
		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}
	return nil
}

func (g *CloudWatchGenerator) createDashboards(cloudwatchSvc *cloudwatch.Client) error {
	var nextToken *string
	for {
		output, err := cloudwatchSvc.ListDashboards(context.TODO(), &cloudwatch.ListDashboardsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return err
		}
		for _, dashboardEntry := range output.DashboardEntries {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				*dashboardEntry.DashboardName,
				*dashboardEntry.DashboardName,
				"aws_cloudwatch_dashboard",
				"aws",
				cloudwatchAllowEmptyValues))
		}
		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}
	return nil
}

func (g *CloudWatchGenerator) createEventBuses(eventsSvc *cloudwatchevents.Client) ([]string, error) {
	var eventBusNames []string
	var nextToken *string
	for {
		output, err := eventsSvc.ListEventBuses(context.TODO(), &cloudwatchevents.ListEventBusesInput{
			NextToken: nextToken,
		})
		if err != nil {
			return eventBusNames, err
		}
		for _, eventBus := range output.EventBuses {
			eventBusName := StringValue(eventBus.Name)
			if eventBusName == "" {
				continue
			}
			eventBusNames = append(eventBusNames, eventBusName)
			if eventBusName != defaultEventBusName {
				g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
					eventBusName,
					eventBusName,
					"aws_cloudwatch_event_bus",
					"aws",
					cloudwatchAllowEmptyValues))
			}
			if StringValue(eventBus.Policy) != "" {
				g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
					eventBusName,
					cloudwatchEventResourceName(eventBusName, "policy"),
					"aws_cloudwatch_event_bus_policy",
					"aws",
					cloudwatchAllowEmptyValues))
			}
		}
		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}
	if len(eventBusNames) == 0 {
		eventBusNames = append(eventBusNames, defaultEventBusName)
	}
	return eventBusNames, nil
}

func (g *CloudWatchGenerator) createRules(eventsSvc *cloudwatchevents.Client, eventBusNames []string) error {
	for _, eventBusName := range eventBusNames {
		if err := g.createRulesForEventBus(eventsSvc, eventBusName); err != nil {
			if eventBusName != defaultEventBusName {
				log.Printf("skipping EventBridge rules for event bus %s: %v", eventBusName, err)
				continue
			}
			return err
		}
	}
	return nil
}

func (g *CloudWatchGenerator) createRulesForEventBus(eventsSvc *cloudwatchevents.Client, eventBusName string) error {
	var listRulesNextToken *string
	for {
		input := &cloudwatchevents.ListRulesInput{
			EventBusName: aws.String(eventBusName),
			NextToken:    listRulesNextToken,
		}
		output, err := eventsSvc.ListRules(context.TODO(), input)
		if err != nil {
			return err
		}
		for _, rule := range output.Rules {
			ruleName := StringValue(rule.Name)
			if ruleName == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				cloudwatchEventRuleImportID(eventBusName, ruleName),
				cloudwatchEventResourceName(eventBusName, ruleName),
				"aws_cloudwatch_event_rule",
				"aws",
				cloudwatchAllowEmptyValues))

			var listTargetsNextToken *string
			for {
				targetResponse, err := eventsSvc.ListTargetsByRule(context.TODO(), &cloudwatchevents.ListTargetsByRuleInput{
					EventBusName: aws.String(eventBusName),
					Rule:         aws.String(ruleName),
					NextToken:    listTargetsNextToken,
				})
				if err != nil {
					return err
				}
				for _, target := range targetResponse.Targets {
					targetID := StringValue(target.Id)
					if targetID == "" {
						continue
					}
					targetRef := cloudwatchEventTargetImportID(eventBusName, ruleName, targetID)
					attributes := map[string]string{
						"event_bus_name": eventBusName,
						"rule":           ruleName,
						"target_id":      targetID,
					}
					g.Resources = append(g.Resources, terraformutils.NewResource(
						targetRef,
						cloudwatchEventResourceName(eventBusName, ruleName, targetID),
						"aws_cloudwatch_event_target",
						"aws",
						attributes,
						cloudwatchAllowEmptyValues,
						map[string]interface{}{}))
				}
				listTargetsNextToken = targetResponse.NextToken
				if listTargetsNextToken == nil {
					break
				}
			}
		}
		listRulesNextToken = output.NextToken
		if listRulesNextToken == nil {
			break
		}
	}

	return nil
}

func (g *CloudWatchGenerator) createArchives(eventsSvc *cloudwatchevents.Client) error {
	var nextToken *string
	for {
		output, err := eventsSvc.ListArchives(context.TODO(), &cloudwatchevents.ListArchivesInput{
			NextToken: nextToken,
		})
		if err != nil {
			return err
		}
		for _, archive := range output.Archives {
			archiveName := StringValue(archive.ArchiveName)
			if archiveName == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				archiveName,
				archiveName,
				"aws_cloudwatch_event_archive",
				"aws",
				cloudwatchAllowEmptyValues))
		}
		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}
	return nil
}

func (g *CloudWatchGenerator) createConnections(eventsSvc *cloudwatchevents.Client) error {
	var nextToken *string
	for {
		output, err := eventsSvc.ListConnections(context.TODO(), &cloudwatchevents.ListConnectionsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return err
		}
		for _, connection := range output.Connections {
			connectionName := StringValue(connection.Name)
			if connectionName == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				connectionName,
				connectionName,
				"aws_cloudwatch_event_connection",
				"aws",
				cloudwatchAllowEmptyValues))
		}
		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}
	return nil
}

func (g *CloudWatchGenerator) createAPIDestinations(eventsSvc *cloudwatchevents.Client) error {
	var nextToken *string
	for {
		output, err := eventsSvc.ListApiDestinations(context.TODO(), &cloudwatchevents.ListApiDestinationsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return err
		}
		for _, destination := range output.ApiDestinations {
			destinationName := StringValue(destination.Name)
			if destinationName == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				destinationName,
				destinationName,
				"aws_cloudwatch_event_api_destination",
				"aws",
				cloudwatchAllowEmptyValues))
		}
		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}
	return nil
}

func cloudwatchEventRuleImportID(eventBusName, ruleName string) string {
	if eventBusName == "" || eventBusName == defaultEventBusName {
		return ruleName
	}
	return eventBusName + "/" + ruleName
}

func cloudwatchEventTargetImportID(eventBusName, ruleName, targetID string) string {
	if eventBusName == "" || eventBusName == defaultEventBusName {
		return ruleName + "/" + targetID
	}
	return eventBusName + "/" + ruleName + "/" + targetID
}

func cloudwatchEventResourceName(parts ...string) string {
	var name string
	for _, part := range parts {
		if part == "" || part == defaultEventBusName {
			continue
		}
		if name != "" {
			name += "_"
		}
		name += part
	}
	return name
}
