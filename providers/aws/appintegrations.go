// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/appintegrations"
	appintegrationstypes "github.com/aws/aws-sdk-go-v2/service/appintegrations/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	appIntegrationsDataIntegrationResourceType  = "aws_appintegrations_data_integration"
	appIntegrationsEventIntegrationResourceType = "aws_appintegrations_event_integration"
)

var appIntegrationsAllowEmptyValues = []string{"tags."}

type AppIntegrationsGenerator struct {
	AWSService
}

func (g *AppIntegrationsGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := appintegrations.NewFromConfig(config)
	if err := g.loadDataIntegrations(svc); err != nil {
		return err
	}
	return g.loadEventIntegrations(svc)
}

func (g *AppIntegrationsGenerator) loadDataIntegrations(svc *appintegrations.Client) error {
	p := appintegrations.NewListDataIntegrationsPaginator(svc, &appintegrations.ListDataIntegrationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, integration := range page.DataIntegrations {
			integrationID := arnLastSegment(StringValue(integration.Arn), "/")
			if integrationID == "" {
				continue
			}
			output, err := svc.GetDataIntegration(context.TODO(), &appintegrations.GetDataIntegrationInput{
				Identifier: &integrationID,
			})
			if err != nil {
				if appIntegrationsNotFound(err) {
					continue
				}
				return err
			}
			if resource, ok := newAppIntegrationsDataIntegrationResource(output); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AppIntegrationsGenerator) loadEventIntegrations(svc *appintegrations.Client) error {
	p := appintegrations.NewListEventIntegrationsPaginator(svc, &appintegrations.ListEventIntegrationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, integration := range page.EventIntegrations {
			name := StringValue(integration.Name)
			if name == "" {
				continue
			}
			output, err := svc.GetEventIntegration(context.TODO(), &appintegrations.GetEventIntegrationInput{
				Name: &name,
			})
			if err != nil {
				if appIntegrationsNotFound(err) {
					continue
				}
				return err
			}
			if resource, ok := newAppIntegrationsEventIntegrationResource(output); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newAppIntegrationsDataIntegrationResource(output *appintegrations.GetDataIntegrationOutput) (terraformutils.Resource, bool) {
	if output == nil || StringValue(output.Id) == "" || StringValue(output.Name) == "" || StringValue(output.KmsKey) == "" || StringValue(output.SourceURI) == "" || !appIntegrationsScheduleConfigImportable(output.ScheduleConfiguration) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"arn":        StringValue(output.Arn),
		"kms_key":    StringValue(output.KmsKey),
		"name":       StringValue(output.Name),
		"source_uri": StringValue(output.SourceURI),
	}
	if description := StringValue(output.Description); description != "" {
		attributes["description"] = description
	}
	return terraformutils.NewResource(
		appIntegrationsDataIntegrationImportID(StringValue(output.Id)),
		appIntegrationsResourceName("data_integration", StringValue(output.Name), StringValue(output.Id)),
		appIntegrationsDataIntegrationResourceType,
		"aws",
		attributes,
		appIntegrationsAllowEmptyValues,
		map[string]interface{}{
			"schedule_config": []interface{}{map[string]interface{}{
				"first_execution_from": StringValue(output.ScheduleConfiguration.FirstExecutionFrom),
				"object":               StringValue(output.ScheduleConfiguration.Object),
				"schedule_expression":  StringValue(output.ScheduleConfiguration.ScheduleExpression),
			}},
		},
	), true
}

func newAppIntegrationsEventIntegrationResource(output *appintegrations.GetEventIntegrationOutput) (terraformutils.Resource, bool) {
	if output == nil || StringValue(output.Name) == "" || StringValue(output.EventBridgeBus) == "" || output.EventFilter == nil || StringValue(output.EventFilter.Source) == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"arn":             StringValue(output.EventIntegrationArn),
		"name":            StringValue(output.Name),
		"eventbridge_bus": StringValue(output.EventBridgeBus),
	}
	if description := StringValue(output.Description); description != "" {
		attributes["description"] = description
	}
	return terraformutils.NewResource(
		appIntegrationsEventIntegrationImportID(StringValue(output.Name)),
		appIntegrationsResourceName("event_integration", StringValue(output.Name), StringValue(output.EventIntegrationArn)),
		appIntegrationsEventIntegrationResourceType,
		"aws",
		attributes,
		appIntegrationsAllowEmptyValues,
		map[string]interface{}{
			"event_filter": []interface{}{map[string]interface{}{"source": StringValue(output.EventFilter.Source)}},
		},
	), true
}

func appIntegrationsScheduleConfigImportable(schedule *appintegrationstypes.ScheduleConfiguration) bool {
	return schedule != nil && StringValue(schedule.FirstExecutionFrom) != "" && StringValue(schedule.Object) != "" && StringValue(schedule.ScheduleExpression) != ""
}

func appIntegrationsDataIntegrationImportID(id string) string {
	return id
}

func appIntegrationsEventIntegrationImportID(name string) string {
	return name
}

func appIntegrationsResourceName(parts ...string) string {
	return resourceNameWithLengthPrefixes(parts...)
}

func appIntegrationsNotFound(err error) bool {
	var notFound *appintegrationstypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
