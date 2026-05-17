// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/google/uuid"
	"github.com/zclconf/go-cty/cty"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestIncidentResourceConstruction(t *testing.T) {
	incidentType := datadogV2.NewIncidentTypeObject("incident-type-1", datadogV2.INCIDENTTYPETYPE_INCIDENT_TYPES)
	template := datadogV2.NewIncidentNotificationTemplateResponseData(uuid.MustParse("11111111-2222-3333-4444-555555555555"), datadogV2.INCIDENTNOTIFICATIONTEMPLATETYPE_NOTIFICATION_TEMPLATES)
	rule := datadogV2.NewIncidentNotificationRuleResponseData(uuid.MustParse("22222222-3333-4444-5555-666666666666"), datadogV2.INCIDENTNOTIFICATIONRULETYPE_INCIDENT_NOTIFICATION_RULES)

	tests := []struct {
		name     string
		create   func() (terraformutils.Resource, error)
		wantID   string
		wantName string
		wantType string
	}{
		{
			name: "incident_type",
			create: func() (terraformutils.Resource, error) {
				return (&IncidentTypeGenerator{}).createResource(*incidentType)
			},
			wantID:   "incident-type-1",
			wantName: "tfer--incident_type_incident-type-1",
			wantType: "datadog_incident_type",
		},
		{
			name: "incident_notification_template",
			create: func() (terraformutils.Resource, error) {
				return (&IncidentNotificationTemplateGenerator{}).createResource(*template)
			},
			wantID:   "11111111-2222-3333-4444-555555555555",
			wantName: "tfer--incident_notification_template_11111111-2222-3333-4444-555555555555",
			wantType: "datadog_incident_notification_template",
		},
		{
			name: "incident_notification_rule",
			create: func() (terraformutils.Resource, error) {
				return (&IncidentNotificationRuleGenerator{}).createResource(*rule)
			},
			wantID:   "22222222-3333-4444-5555-666666666666",
			wantName: "tfer--incident_notification_rule_22222222-3333-4444-5555-666666666666",
			wantType: "datadog_incident_notification_rule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, err := tt.create()
			if err != nil {
				t.Fatalf("createResource returned error: %v", err)
			}
			if resource.InstanceState.ID != tt.wantID {
				t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, tt.wantID)
			}
			if resource.ResourceName != tt.wantName {
				t.Fatalf("resource name = %q, want %q", resource.ResourceName, tt.wantName)
			}
			if resource.InstanceInfo.Type != tt.wantType {
				t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, tt.wantType)
			}
		})
	}
}

func TestIncidentResourceConstructionMissingID(t *testing.T) {
	if _, err := (&IncidentTypeGenerator{}).createResource(datadogV2.IncidentTypeObject{}); err == nil {
		t.Fatal("incident type createResource returned nil error, want missing id error")
	}
	if _, err := (&IncidentNotificationTemplateGenerator{}).createResource(datadogV2.IncidentNotificationTemplateResponseData{}); err == nil {
		t.Fatal("incident notification template createResource returned nil error, want missing id error")
	}
	if _, err := (&IncidentNotificationRuleGenerator{}).createResource(datadogV2.IncidentNotificationRuleResponseData{}); err == nil {
		t.Fatal("incident notification rule createResource returned nil error, want missing id error")
	}
}

func TestIncidentTypeAllowEmptyValuesPreservesZeroValues(t *testing.T) {
	allowEmptyValues := allowEmptyValueRegexps(IncidentTypeAllowEmptyValues)
	parser := terraformutils.NewFlatmapParser(map[string]string{
		"description": "",
		"is_default":  "",
	}, nil, allowEmptyValues)
	incidentType := cty.Object(map[string]cty.Type{
		"description": cty.String,
		"is_default":  cty.Bool,
	})

	result, err := parser.Parse(incidentType)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	for _, field := range []string{"description", "is_default"} {
		if _, ok := result[field]; !ok {
			t.Fatalf("%s was not preserved", field)
		}
	}
}

func TestIncidentNotificationTemplateAllowEmptyValuesPreservesRequiredStrings(t *testing.T) {
	allowEmptyValues := allowEmptyValueRegexps(IncidentNotificationTemplateAllowEmptyValues)
	parser := terraformutils.NewFlatmapParser(map[string]string{
		"name":          "",
		"subject":       "",
		"content":       "",
		"category":      "",
		"incident_type": "",
	}, nil, allowEmptyValues)
	templateType := cty.Object(map[string]cty.Type{
		"name":          cty.String,
		"subject":       cty.String,
		"content":       cty.String,
		"category":      cty.String,
		"incident_type": cty.String,
	})

	result, err := parser.Parse(templateType)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	for _, field := range []string{"name", "subject", "content", "category", "incident_type"} {
		if result[field] != "" {
			t.Fatalf("%s = %v, want empty string", field, result[field])
		}
	}
}

func TestIncidentTypeInitResourcesListsTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/incidents/config/types" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, incidentTypeListResponseJSON("incident-type-1", "incident-type-2"))
	}))
	defer server.Close()

	generator := newIncidentTypeTestGenerator(server, nil)
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertResourceIDs(t, generator.Resources, []string{"incident-type-1", "incident-type-2"})
}

func TestIncidentNotificationTemplateInitResourcesListsTemplates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/incidents/config/notification-templates" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, incidentNotificationTemplateListResponseJSON("11111111-2222-3333-4444-555555555555", "22222222-3333-4444-5555-666666666666"))
	}))
	defer server.Close()

	generator := newIncidentNotificationTemplateTestGenerator(server, nil)
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertResourceIDs(t, generator.Resources, []string{"11111111-2222-3333-4444-555555555555", "22222222-3333-4444-5555-666666666666"})
}

func TestIncidentNotificationRuleInitResourcesListsRules(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/incidents/config/notification-rules" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, incidentNotificationRuleListResponseJSON("33333333-4444-5555-6666-777777777777", "44444444-5555-6666-7777-888888888888"))
	}))
	defer server.Close()

	generator := newIncidentNotificationRuleTestGenerator(server, nil)
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertResourceIDs(t, generator.Resources, []string{"33333333-4444-5555-6666-777777777777", "44444444-5555-6666-7777-888888888888"})
}

func TestIncidentInitResourcesFiltersByID(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		id          string
		path        string
		body        string
		init        func(*httptest.Server, []terraformutils.ResourceFilter) ([]terraformutils.Resource, error)
	}{
		{
			name:        "incident_type",
			serviceName: "incident_type",
			id:          "incident-type-1",
			path:        "/api/v2/incidents/config/types/incident-type-1",
			body:        incidentTypeResponseJSON("incident-type-1"),
			init: func(server *httptest.Server, filters []terraformutils.ResourceFilter) ([]terraformutils.Resource, error) {
				generator := newIncidentTypeTestGenerator(server, filters)
				err := generator.InitResources()
				return generator.Resources, err
			},
		},
		{
			name:        "incident_notification_template",
			serviceName: "incident_notification_template",
			id:          "11111111-2222-3333-4444-555555555555",
			path:        "/api/v2/incidents/config/notification-templates/11111111-2222-3333-4444-555555555555",
			body:        incidentNotificationTemplateResponseJSON("11111111-2222-3333-4444-555555555555"),
			init: func(server *httptest.Server, filters []terraformutils.ResourceFilter) ([]terraformutils.Resource, error) {
				generator := newIncidentNotificationTemplateTestGenerator(server, filters)
				err := generator.InitResources()
				return generator.Resources, err
			},
		},
		{
			name:        "incident_notification_rule",
			serviceName: "incident_notification_rule",
			id:          "33333333-4444-5555-6666-777777777777",
			path:        "/api/v2/incidents/config/notification-rules/33333333-4444-5555-6666-777777777777",
			body:        incidentNotificationRuleResponseJSON("33333333-4444-5555-6666-777777777777"),
			init: func(server *httptest.Server, filters []terraformutils.ResourceFilter) ([]terraformutils.Resource, error) {
				generator := newIncidentNotificationRuleTestGenerator(server, filters)
				err := generator.InitResources()
				return generator.Resources, err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path != tt.path {
					http.NotFound(w, r)
					return
				}
				_, _ = fmt.Fprint(w, tt.body)
			}))
			defer server.Close()

			resources, err := tt.init(server, []terraformutils.ResourceFilter{{
				ServiceName:      tt.serviceName,
				FieldPath:        "id",
				AcceptableValues: []string{tt.id},
			}})
			if err != nil {
				t.Fatalf("InitResources returned error: %v", err)
			}
			assertResourceIDs(t, resources, []string{tt.id})
		})
	}
}

func TestIncidentInitResourcesListsWithUnrelatedIDFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/incidents/config/types" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, incidentTypeListResponseJSON("incident-type-1"))
	}))
	defer server.Close()

	generator := newIncidentTypeTestGenerator(server, []terraformutils.ResourceFilter{{
		ServiceName:      "team",
		FieldPath:        "id",
		AcceptableValues: []string{"team-1"},
	}})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertResourceIDs(t, generator.Resources, []string{"incident-type-1"})
}

func TestWebhookCreateResource(t *testing.T) {
	webhook := datadogV1.NewWebhooksIntegration("example-webhook", "https://example.com/webhook")
	resource, err := (&WebhookGenerator{}).createResource(*webhook)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}
	if resource.InstanceState.ID != "example-webhook" {
		t.Fatalf("resource ID = %q, want example-webhook", resource.InstanceState.ID)
	}
	if resource.ResourceName != "tfer--webhook_example-webhook" {
		t.Fatalf("resource name = %q, want tfer--webhook_example-webhook", resource.ResourceName)
	}
	if resource.InstanceInfo.Type != "datadog_webhook" {
		t.Fatalf("resource type = %q, want datadog_webhook", resource.InstanceInfo.Type)
	}
}

func TestWebhookInitResourcesRequiresApplicableIDFilter(t *testing.T) {
	tests := []struct {
		name   string
		filter []terraformutils.ResourceFilter
	}{
		{
			name: "no filter",
		},
		{
			name: "unrelated ID filter",
			filter: []terraformutils.ResourceFilter{{
				ServiceName:      "incident_type",
				FieldPath:        "id",
				AcceptableValues: []string{"incident-type-1"},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "unexpected API request", http.StatusInternalServerError)
			}))
			defer server.Close()

			generator := newWebhookTestGenerator(server, tt.filter)
			if err := generator.InitResources(); err != nil {
				t.Fatalf("InitResources returned error: %v", err)
			}
			if len(generator.Resources) != 0 {
				t.Fatalf("expected no resources without applicable ID filter, got %d", len(generator.Resources))
			}
		})
	}
}

func TestWebhookInitResourcesFiltersByName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v1/integration/webhooks/configuration/webhooks/example-webhook" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, webhookResponseJSON("example-webhook"))
	}))
	defer server.Close()

	generator := newWebhookTestGenerator(server, []terraformutils.ResourceFilter{{
		ServiceName:      "webhook",
		FieldPath:        "id",
		AcceptableValues: []string{"example-webhook"},
	}})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertResourceIDs(t, generator.Resources, []string{"example-webhook"})
}

func TestWorkflowAutomationCreateResource(t *testing.T) {
	workflow := datadogV2.NewWorkflowDataWithDefaults()
	workflow.SetId("workflow-1")
	resource, err := (&WorkflowAutomationGenerator{}).createResource(*workflow)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}
	if resource.InstanceState.ID != "workflow-1" {
		t.Fatalf("resource ID = %q, want workflow-1", resource.InstanceState.ID)
	}
	if resource.ResourceName != "tfer--workflow_automation_workflow-1" {
		t.Fatalf("resource name = %q, want tfer--workflow_automation_workflow-1", resource.ResourceName)
	}
	if resource.InstanceInfo.Type != "datadog_workflow_automation" {
		t.Fatalf("resource type = %q, want datadog_workflow_automation", resource.InstanceInfo.Type)
	}
}

func TestWorkflowAutomationAllowEmptyValuesPreservesRequiredPublished(t *testing.T) {
	allowEmptyValues := allowEmptyValueRegexps(WorkflowAutomationAllowEmptyValues)
	parser := terraformutils.NewFlatmapParser(map[string]string{
		"published": "",
	}, nil, allowEmptyValues)
	workflowType := cty.Object(map[string]cty.Type{
		"published": cty.Bool,
	})

	result, err := parser.Parse(workflowType)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if _, ok := result["published"]; !ok {
		t.Fatal("published was not preserved")
	}
}

func TestWorkflowAutomationInitResourcesRequiresApplicableIDFilter(t *testing.T) {
	tests := []struct {
		name   string
		filter []terraformutils.ResourceFilter
	}{
		{
			name: "no filter",
		},
		{
			name: "unrelated ID filter",
			filter: []terraformutils.ResourceFilter{{
				ServiceName:      "incident_type",
				FieldPath:        "id",
				AcceptableValues: []string{"incident-type-1"},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "unexpected API request", http.StatusInternalServerError)
			}))
			defer server.Close()

			generator := newWorkflowAutomationTestGenerator(server, tt.filter)
			if err := generator.InitResources(); err != nil {
				t.Fatalf("InitResources returned error: %v", err)
			}
			if len(generator.Resources) != 0 {
				t.Fatalf("expected no resources without applicable ID filter, got %d", len(generator.Resources))
			}
		})
	}
}

func TestWorkflowAutomationInitResourcesFiltersByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/workflows/workflow-1" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, workflowAutomationResponseJSON("workflow-1"))
	}))
	defer server.Close()

	generator := newWorkflowAutomationTestGenerator(server, []terraformutils.ResourceFilter{{
		ServiceName:      "workflow_automation",
		FieldPath:        "id",
		AcceptableValues: []string{"workflow-1"},
	}})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertResourceIDs(t, generator.Resources, []string{"workflow-1"})
}

func TestWorkflowAutomationPostConvertHookPreservesEmptyTags(t *testing.T) {
	resource := terraformutils.NewSimpleResource(
		"workflow-1",
		"workflow_automation_workflow-1",
		"datadog_workflow_automation",
		"datadog",
		WorkflowAutomationAllowEmptyValues,
	)
	resource.InstanceState.Attributes = map[string]string{"tags.#": "0"}

	generator := &WorkflowAutomationGenerator{}
	generator.Resources = []terraformutils.Resource{resource}
	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}
	tags, ok := generator.Resources[0].Item["tags"].([]interface{})
	if !ok {
		t.Fatalf("tags = %T, want []interface{}", generator.Resources[0].Item["tags"])
	}
	if len(tags) != 0 {
		t.Fatalf("tags length = %d, want 0", len(tags))
	}
}

func newIncidentTypeTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *IncidentTypeGenerator {
	return &IncidentTypeGenerator{DatadogService: newIncidentWorkflowTestService(server, filter)}
}

func newIncidentNotificationTemplateTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *IncidentNotificationTemplateGenerator {
	return &IncidentNotificationTemplateGenerator{DatadogService: newIncidentWorkflowTestService(server, filter)}
}

func newIncidentNotificationRuleTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *IncidentNotificationRuleGenerator {
	return &IncidentNotificationRuleGenerator{DatadogService: newIncidentWorkflowTestService(server, filter)}
}

func newWebhookTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *WebhookGenerator {
	return &WebhookGenerator{DatadogService: newIncidentWorkflowTestService(server, filter)}
}

func newWorkflowAutomationTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *WorkflowAutomationGenerator {
	return &WorkflowAutomationGenerator{DatadogService: newIncidentWorkflowTestService(server, filter)}
}

func newIncidentWorkflowTestService(server *httptest.Server, filter []terraformutils.ResourceFilter) DatadogService {
	return DatadogService{
		Service: terraformutils.Service{
			Args: map[string]interface{}{
				"auth":          context.Background(),
				"datadogClient": newTeamRelationshipTestClient(server),
			},
			Filter: filter,
		},
	}
}

func assertResourceIDs(t *testing.T, resources []terraformutils.Resource, want []string) {
	t.Helper()
	if len(resources) != len(want) {
		t.Fatalf("resource count = %d, want %d", len(resources), len(want))
	}
	for i, id := range want {
		if resources[i].InstanceState.ID != id {
			t.Fatalf("resource ID[%d] = %q, want %q", i, resources[i].InstanceState.ID, id)
		}
	}
}

func incidentTypeResponseJSON(id string) string {
	return fmt.Sprintf("{\"data\":%s}", incidentTypeJSON(id))
}

func incidentTypeListResponseJSON(ids ...string) string {
	items := []string{}
	for _, id := range ids {
		items = append(items, incidentTypeJSON(id))
	}
	return fmt.Sprintf("{\"data\":[%s]}", strings.Join(items, ","))
}

func incidentTypeJSON(id string) string {
	return fmt.Sprintf("{\"id\":%q,\"type\":\"incident_types\",\"attributes\":{\"name\":%q,\"description\":\"\",\"is_default\":false}}", id, id)
}

func incidentNotificationTemplateResponseJSON(id string) string {
	return fmt.Sprintf("{\"data\":%s}", incidentNotificationTemplateJSON(id))
}

func incidentNotificationTemplateListResponseJSON(ids ...string) string {
	items := []string{}
	for _, id := range ids {
		items = append(items, incidentNotificationTemplateJSON(id))
	}
	return fmt.Sprintf("{\"data\":[%s]}", strings.Join(items, ","))
}

func incidentNotificationTemplateJSON(id string) string {
	return fmt.Sprintf("{\"id\":%q,\"type\":\"notification_templates\",\"attributes\":{\"name\":\"template-%s\",\"subject\":\"\",\"content\":\"\",\"category\":\"incident\",\"created\":\"2024-01-02T03:04:05Z\",\"modified\":\"2024-01-02T03:04:05Z\"},\"relationships\":{\"incident_type\":{\"data\":{\"id\":\"incident-type-1\",\"type\":\"incident_types\"}}}}", id, id)
}

func incidentNotificationRuleResponseJSON(id string) string {
	return fmt.Sprintf("{\"data\":%s}", incidentNotificationRuleJSON(id))
}

func incidentNotificationRuleListResponseJSON(ids ...string) string {
	items := []string{}
	for _, id := range ids {
		items = append(items, incidentNotificationRuleJSON(id))
	}
	return fmt.Sprintf("{\"data\":[%s]}", strings.Join(items, ","))
}

func incidentNotificationRuleJSON(id string) string {
	return fmt.Sprintf("{\"id\":%q,\"type\":\"incident_notification_rules\",\"attributes\":{\"conditions\":[{\"field\":\"state\",\"values\":[\"active\"]}],\"created\":\"2024-01-02T03:04:05Z\",\"enabled\":true,\"handles\":[\"@team@example.com\"],\"modified\":\"2024-01-02T03:04:05Z\",\"trigger\":\"incident_created_trigger\",\"visibility\":\"organization\"},\"relationships\":{\"incident_type\":{\"data\":{\"id\":\"incident-type-1\",\"type\":\"incident_types\"}}}}", id)
}

func webhookResponseJSON(name string) string {
	return fmt.Sprintf("{\"name\":%q,\"url\":\"https://example.com/webhook\",\"payload\":\"\",\"encode_as\":\"json\"}", name)
}

func workflowAutomationResponseJSON(id string) string {
	return fmt.Sprintf("{\"data\":{\"id\":%q,\"type\":\"workflows\",\"attributes\":{\"name\":\"workflow\",\"description\":\"\",\"tags\":[\"team:ops\"],\"published\":true,\"spec\":{\"triggers\":[],\"steps\":[]}}}}", id)
}
