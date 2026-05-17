// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/google/uuid"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// IncidentTypeAllowEmptyValues ...
	IncidentTypeAllowEmptyValues = []string{}
	// IncidentNotificationTemplateAllowEmptyValues ...
	IncidentNotificationTemplateAllowEmptyValues = []string{"name", "subject", "content", "category", "incident_type"}
	// IncidentNotificationRuleAllowEmptyValues ...
	IncidentNotificationRuleAllowEmptyValues = []string{}
)

// IncidentTypeGenerator ...
type IncidentTypeGenerator struct {
	DatadogService
}

// IncidentNotificationTemplateGenerator ...
type IncidentNotificationTemplateGenerator struct {
	DatadogService
}

// IncidentNotificationRuleGenerator ...
type IncidentNotificationRuleGenerator struct {
	DatadogService
}

func (g *IncidentTypeGenerator) createResource(incidentType datadogV2.IncidentTypeObject) (terraformutils.Resource, error) {
	incidentTypeID := incidentType.GetId()
	if incidentTypeID == "" {
		return terraformutils.Resource{}, fmt.Errorf("incident type missing id")
	}

	return terraformutils.NewSimpleResource(
		incidentTypeID,
		fmt.Sprintf("incident_type_%s", incidentTypeID),
		"datadog_incident_type",
		"datadog",
		IncidentTypeAllowEmptyValues,
	), nil
}

func (g *IncidentTypeGenerator) createResources(incidentTypes []datadogV2.IncidentTypeObject) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, incidentType := range incidentTypes {
		resource, err := g.createResource(incidentType)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each incident_type create 1 TerraformResource.
// Need incident type ID as ID for terraform resource.
func (g *IncidentTypeGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	enableIncidentTypeOperations(datadogClient)
	api := datadogV2.NewIncidentsApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	incidentTypes, err := listIncidentTypes(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(incidentTypes)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}

func (g *IncidentTypeGenerator) filteredResources(auth context.Context, api *datadogV2.IncidentsApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	matchedIDFilter := false

	for _, filter := range g.Filter {
		if filter.FieldPath != "id" {
			continue
		}
		if !filter.IsApplicable("incident_type") {
			continue
		}
		matchedIDFilter = true

		for _, value := range filter.AcceptableValues {
			incidentType, err := getIncidentType(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(incidentType)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, matchedIDFilter, nil
}

func getIncidentType(auth context.Context, api *datadogV2.IncidentsApi, incidentTypeID string) (datadogV2.IncidentTypeObject, error) {
	response, httpResponse, err := api.GetIncidentType(auth, incidentTypeID)
	closeDatadogResponseBody(httpResponse)
	if err != nil {
		return datadogV2.IncidentTypeObject{}, err
	}
	return response.GetData(), nil
}

func listIncidentTypes(auth context.Context, api *datadogV2.IncidentsApi) ([]datadogV2.IncidentTypeObject, error) {
	response, httpResponse, err := api.ListIncidentTypes(auth)
	closeDatadogResponseBody(httpResponse)
	if err != nil {
		return nil, err
	}
	return response.GetData(), nil
}

func (g *IncidentNotificationTemplateGenerator) createResource(template datadogV2.IncidentNotificationTemplateResponseData) (terraformutils.Resource, error) {
	templateID := template.GetId()
	if templateID == uuid.Nil {
		return terraformutils.Resource{}, fmt.Errorf("incident notification template missing id")
	}
	id := templateID.String()
	return terraformutils.NewSimpleResource(
		id,
		fmt.Sprintf("incident_notification_template_%s", id),
		"datadog_incident_notification_template",
		"datadog",
		IncidentNotificationTemplateAllowEmptyValues,
	), nil
}

func (g *IncidentNotificationTemplateGenerator) createResources(templates []datadogV2.IncidentNotificationTemplateResponseData) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, template := range templates {
		resource, err := g.createResource(template)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each incident_notification_template create 1 TerraformResource.
// Need incident notification template ID as ID for terraform resource.
func (g *IncidentNotificationTemplateGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	enableIncidentNotificationTemplateOperations(datadogClient)
	api := datadogV2.NewIncidentsApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	templates, err := listIncidentNotificationTemplates(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(templates)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}

func (g *IncidentNotificationTemplateGenerator) filteredResources(auth context.Context, api *datadogV2.IncidentsApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	matchedIDFilter := false

	for _, filter := range g.Filter {
		if filter.FieldPath != "id" {
			continue
		}
		if !filter.IsApplicable("incident_notification_template") {
			continue
		}
		matchedIDFilter = true

		for _, value := range filter.AcceptableValues {
			template, err := getIncidentNotificationTemplate(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(template)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, matchedIDFilter, nil
}

func getIncidentNotificationTemplate(auth context.Context, api *datadogV2.IncidentsApi, templateID string) (datadogV2.IncidentNotificationTemplateResponseData, error) {
	id, err := uuid.Parse(templateID)
	if err != nil {
		return datadogV2.IncidentNotificationTemplateResponseData{}, err
	}

	response, httpResponse, err := api.GetIncidentNotificationTemplate(auth, id)
	closeDatadogResponseBody(httpResponse)
	if err != nil {
		return datadogV2.IncidentNotificationTemplateResponseData{}, err
	}
	return response.GetData(), nil
}

func listIncidentNotificationTemplates(auth context.Context, api *datadogV2.IncidentsApi) ([]datadogV2.IncidentNotificationTemplateResponseData, error) {
	response, httpResponse, err := api.ListIncidentNotificationTemplates(auth)
	closeDatadogResponseBody(httpResponse)
	if err != nil {
		return nil, err
	}
	return response.GetData(), nil
}

func (g *IncidentNotificationRuleGenerator) createResource(rule datadogV2.IncidentNotificationRuleResponseData) (terraformutils.Resource, error) {
	ruleID := rule.GetId()
	if ruleID == uuid.Nil {
		return terraformutils.Resource{}, fmt.Errorf("incident notification rule missing id")
	}
	id := ruleID.String()
	return terraformutils.NewSimpleResource(
		id,
		fmt.Sprintf("incident_notification_rule_%s", id),
		"datadog_incident_notification_rule",
		"datadog",
		IncidentNotificationRuleAllowEmptyValues,
	), nil
}

func (g *IncidentNotificationRuleGenerator) createResources(rules []datadogV2.IncidentNotificationRuleResponseData) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, rule := range rules {
		resource, err := g.createResource(rule)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each incident_notification_rule create 1 TerraformResource.
// Need incident notification rule ID as ID for terraform resource.
func (g *IncidentNotificationRuleGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	enableIncidentNotificationRuleOperations(datadogClient)
	api := datadogV2.NewIncidentsApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	rules, err := listIncidentNotificationRules(auth, api)
	if err != nil {
		return err
	}
	resources, err = g.createResources(rules)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}

func (g *IncidentNotificationRuleGenerator) filteredResources(auth context.Context, api *datadogV2.IncidentsApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	matchedIDFilter := false

	for _, filter := range g.Filter {
		if filter.FieldPath != "id" {
			continue
		}
		if !filter.IsApplicable("incident_notification_rule") {
			continue
		}
		matchedIDFilter = true

		for _, value := range filter.AcceptableValues {
			rule, err := getIncidentNotificationRule(auth, api, value)
			if err != nil {
				return nil, true, err
			}
			resource, err := g.createResource(rule)
			if err != nil {
				return nil, true, err
			}
			resources = append(resources, resource)
		}
	}

	return resources, matchedIDFilter, nil
}

func getIncidentNotificationRule(auth context.Context, api *datadogV2.IncidentsApi, ruleID string) (datadogV2.IncidentNotificationRuleResponseData, error) {
	id, err := uuid.Parse(ruleID)
	if err != nil {
		return datadogV2.IncidentNotificationRuleResponseData{}, err
	}

	response, httpResponse, err := api.GetIncidentNotificationRule(auth, id)
	closeDatadogResponseBody(httpResponse)
	if err != nil {
		return datadogV2.IncidentNotificationRuleResponseData{}, err
	}
	return response.GetData(), nil
}

func listIncidentNotificationRules(auth context.Context, api *datadogV2.IncidentsApi) ([]datadogV2.IncidentNotificationRuleResponseData, error) {
	response, httpResponse, err := api.ListIncidentNotificationRules(auth)
	closeDatadogResponseBody(httpResponse)
	if err != nil {
		return nil, err
	}
	return response.GetData(), nil
}

func enableIncidentTypeOperations(client *datadog.APIClient) {
	client.GetConfig().SetUnstableOperationEnabled("v2.GetIncidentType", true)
	client.GetConfig().SetUnstableOperationEnabled("v2.ListIncidentTypes", true)
}

func enableIncidentNotificationTemplateOperations(client *datadog.APIClient) {
	client.GetConfig().SetUnstableOperationEnabled("v2.GetIncidentNotificationTemplate", true)
	client.GetConfig().SetUnstableOperationEnabled("v2.ListIncidentNotificationTemplates", true)
}

func enableIncidentNotificationRuleOperations(client *datadog.APIClient) {
	client.GetConfig().SetUnstableOperationEnabled("v2.GetIncidentNotificationRule", true)
	client.GetConfig().SetUnstableOperationEnabled("v2.ListIncidentNotificationRules", true)
}
