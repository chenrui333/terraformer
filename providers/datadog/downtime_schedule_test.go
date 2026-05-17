// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/zclconf/go-cty/cty"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestDowntimeScheduleAllowEmptyValuesPreservesMessage(t *testing.T) {
	allowEmptyValues := []*regexp.Regexp{}
	for _, pattern := range DowntimeScheduleAllowEmptyValues {
		allowEmptyValues = append(allowEmptyValues, regexp.MustCompile(pattern))
	}

	parser := terraformutils.NewFlatmapParser(map[string]string{
		downtimeScheduleMessageKey: "",
	}, nil, allowEmptyValues)
	downtimeScheduleType := cty.Object(map[string]cty.Type{
		downtimeScheduleMessageKey: cty.String,
	})

	result, err := parser.Parse(downtimeScheduleType)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	message, ok := result[downtimeScheduleMessageKey].(string)
	if !ok {
		t.Fatalf("message = %T, want string", result[downtimeScheduleMessageKey])
	}
	if message != "" {
		t.Fatalf("message = %q, want empty string", message)
	}
}

func TestDowntimeScheduleCreateResource(t *testing.T) {
	downtime := datadogV2.NewDowntimeResponseDataWithDefaults()
	downtime.SetId("downtime-123")

	generator := &DowntimeScheduleGenerator{}
	resource, err := generator.createResource(*downtime)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "downtime-123" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "downtime-123")
	}
	if resource.ResourceName != "tfer--downtime_schedule_downtime-123" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--downtime_schedule_downtime-123")
	}
	if resource.InstanceInfo.Type != "datadog_downtime_schedule" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_downtime_schedule")
	}
}

func TestDowntimeScheduleCreateResourceMissingID(t *testing.T) {
	generator := &DowntimeScheduleGenerator{}
	_, err := generator.createResource(datadogV2.DowntimeResponseData{})
	if err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestDowntimeScheduleCreateResources(t *testing.T) {
	firstDowntime := datadogV2.NewDowntimeResponseDataWithDefaults()
	firstDowntime.SetId("downtime-1")
	secondDowntime := datadogV2.NewDowntimeResponseDataWithDefaults()
	secondDowntime.SetId("downtime-2")

	generator := &DowntimeScheduleGenerator{}
	resources, err := generator.createResources([]datadogV2.DowntimeResponseData{*firstDowntime, *secondDowntime})
	if err != nil {
		t.Fatalf("createResources returned error: %v", err)
	}

	if len(resources) != 2 {
		t.Fatalf("resource count = %d, want %d", len(resources), 2)
	}
	if resources[0].ResourceName == resources[1].ResourceName {
		t.Fatalf("resource names should be unique, got %q", resources[0].ResourceName)
	}
}

func TestDowntimeSchedulePostConvertHookRemovesEmptyRecurringScheduleState(t *testing.T) {
	resource := terraformutils.NewSimpleResource(
		"downtime-1",
		"downtime_schedule_downtime-1",
		"datadog_downtime_schedule",
		"datadog",
		DowntimeScheduleAllowEmptyValues,
	)
	resource.Item = map[string]interface{}{
		"id": "downtime-1",
		downtimeScheduleOneTimeKey: map[string]interface{}{
			"start": "2026-05-17T14:00:00Z",
		},
		downtimeScheduleRecurringKey: map[string]interface{}{},
	}
	resource.InstanceState.Attributes = map[string]string{
		"id":                               "downtime-1",
		"one_time_schedule.start":          "2026-05-17T14:00:00Z",
		"recurring_schedule.recurrences.#": "0",
		"recurring_schedule.timezone":      "",
	}
	resource.InstanceState.SetTypedAttributes(json.RawMessage("{\"display_timezone\":\"UTC\",\"id\":\"downtime-1\",\"one_time_schedule\":{\"end\":null,\"start\":\"2026-05-17T14:00:00Z\"},\"recurring_schedule\":{\"recurrences\":null,\"timezone\":null}}"))

	generator := &DowntimeScheduleGenerator{
		DatadogService: DatadogService{
			Service: terraformutils.Service{
				Resources: []terraformutils.Resource{resource},
			},
		},
	}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}

	updatedResource := generator.Resources[0]
	if _, ok := updatedResource.Item[downtimeScheduleRecurringKey]; ok {
		t.Fatal("PostConvertHook left empty recurring_schedule in generated item")
	}
	if _, ok := updatedResource.InstanceState.Attributes["recurring_schedule.recurrences.#"]; ok {
		t.Fatal("PostConvertHook left recurring_schedule flatmap state")
	}
	typedAttributes := decodeDowntimeScheduleTypedAttributes(t, updatedResource.InstanceState.TypedAttributes)
	if _, ok := typedAttributes[downtimeScheduleRecurringKey]; ok {
		t.Fatal("PostConvertHook left empty recurring_schedule in typed state")
	}
	if _, ok := typedAttributes[downtimeScheduleOneTimeKey]; !ok {
		t.Fatal("PostConvertHook removed active one_time_schedule typed state")
	}
}

func TestDowntimeSchedulePostConvertHookRemovesEmptyOneTimeScheduleState(t *testing.T) {
	resource := terraformutils.NewSimpleResource(
		"downtime-2",
		"downtime_schedule_downtime-2",
		"datadog_downtime_schedule",
		"datadog",
		DowntimeScheduleAllowEmptyValues,
	)
	resource.Item = map[string]interface{}{
		"id":                       "downtime-2",
		downtimeScheduleOneTimeKey: map[string]interface{}{},
		downtimeScheduleRecurringKey: map[string]interface{}{
			"recurrences": []interface{}{
				map[string]interface{}{
					"duration": "1h",
					"rrule":    "FREQ=DAILY",
					"start":    "2026-05-17T14:00:00Z",
				},
			},
			"timezone": "UTC",
		},
	}
	resource.InstanceState.Attributes = map[string]string{
		"id":                                     "downtime-2",
		"one_time_schedule.start":                "",
		"recurring_schedule.recurrences.#":       "1",
		"recurring_schedule.recurrences.0.start": "2026-05-17T14:00:00Z",
		"recurring_schedule.timezone":            "UTC",
	}
	resource.InstanceState.SetTypedAttributes(json.RawMessage("{\"display_timezone\":\"UTC\",\"id\":\"downtime-2\",\"one_time_schedule\":{\"end\":null,\"start\":null},\"recurring_schedule\":{\"recurrences\":[{\"duration\":\"1h\",\"rrule\":\"FREQ=DAILY\",\"start\":\"2026-05-17T14:00:00Z\"}],\"timezone\":\"UTC\"}}"))

	generator := &DowntimeScheduleGenerator{
		DatadogService: DatadogService{
			Service: terraformutils.Service{
				Resources: []terraformutils.Resource{resource},
			},
		},
	}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}

	updatedResource := generator.Resources[0]
	if _, ok := updatedResource.Item[downtimeScheduleOneTimeKey]; ok {
		t.Fatal("PostConvertHook left empty one_time_schedule in generated item")
	}
	if _, ok := updatedResource.InstanceState.Attributes["one_time_schedule.start"]; ok {
		t.Fatal("PostConvertHook left one_time_schedule flatmap state")
	}
	typedAttributes := decodeDowntimeScheduleTypedAttributes(t, updatedResource.InstanceState.TypedAttributes)
	if _, ok := typedAttributes[downtimeScheduleOneTimeKey]; ok {
		t.Fatal("PostConvertHook left empty one_time_schedule in typed state")
	}
	if _, ok := typedAttributes[downtimeScheduleRecurringKey]; !ok {
		t.Fatal("PostConvertHook removed active recurring_schedule typed state")
	}
}

func TestDowntimeSchedulePostConvertHookPreservesEmptyNotificationDefaults(t *testing.T) {
	resource := terraformutils.NewSimpleResource(
		"downtime-3",
		"downtime_schedule_downtime-3",
		"datadog_downtime_schedule",
		"datadog",
		DowntimeScheduleAllowEmptyValues,
	)
	resource.Item = map[string]interface{}{
		"id": "downtime-3",
		downtimeScheduleOneTimeKey: map[string]interface{}{
			"start": "2026-05-17T14:00:00Z",
		},
	}
	resource.InstanceState.Attributes = map[string]string{
		"id":                      "downtime-3",
		"one_time_schedule.start": "2026-05-17T14:00:00Z",
	}
	resource.InstanceState.SetTypedAttributes(json.RawMessage("{\"display_timezone\":\"UTC\",\"id\":\"downtime-3\",\"notify_end_states\":null,\"one_time_schedule\":{\"end\":null,\"start\":\"2026-05-17T14:00:00Z\"}}"))

	generator := &DowntimeScheduleGenerator{
		DatadogService: DatadogService{
			Service: terraformutils.Service{
				Resources: []terraformutils.Resource{resource},
			},
		},
	}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}

	updatedResource := generator.Resources[0]
	assertEmptyDowntimeScheduleItemList(t, updatedResource, downtimeScheduleNotifyEndStatesKey)
	assertEmptyDowntimeScheduleItemList(t, updatedResource, downtimeScheduleNotifyEndTypesKey)
	if got := updatedResource.InstanceState.Attributes["notify_end_states.#"]; got != "0" {
		t.Fatalf("notify_end_states.# = %q, want 0", got)
	}
	if got := updatedResource.InstanceState.Attributes["notify_end_types.#"]; got != "0" {
		t.Fatalf("notify_end_types.# = %q, want 0", got)
	}
	typedAttributes := decodeDowntimeScheduleTypedAttributes(t, updatedResource.InstanceState.TypedAttributes)
	if got := string(typedAttributes[downtimeScheduleNotifyEndStatesKey]); got != "[]" {
		t.Fatalf("typed notify_end_states = %s, want []", got)
	}
	if got := string(typedAttributes[downtimeScheduleNotifyEndTypesKey]); got != "[]" {
		t.Fatalf("typed notify_end_types = %s, want []", got)
	}
	if _, ok := typedAttributes[downtimeScheduleOneTimeKey]; !ok {
		t.Fatal("PostConvertHook removed active one_time_schedule typed state")
	}
}

func TestDowntimeScheduleInitResourcesList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/downtime" {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("page[offset]"); got != "0" {
			t.Errorf("page[offset] query = %q, want %q", got, "0")
		}
		if got := r.URL.Query().Get("page[limit]"); got != fmt.Sprint(datadogDowntimeSchedulePageLimit) {
			t.Errorf("page[limit] query = %q, want %q", got, fmt.Sprint(datadogDowntimeSchedulePageLimit))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"data\":[{\"id\":\"downtime-1\",\"type\":\"downtime\"}],\"meta\":{\"page\":{\"total_filtered_count\":1}}}"))
	}))
	t.Cleanup(server.Close)

	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	generator := &DowntimeScheduleGenerator{
		DatadogService: DatadogService{
			Service: terraformutils.Service{
				Args: map[string]interface{}{
					"auth":          context.Background(),
					"datadogClient": datadog.NewAPIClient(config),
				},
			},
		},
	}

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("resource count = %d, want %d", len(generator.Resources), 1)
	}
	if generator.Resources[0].InstanceState.ID != "downtime-1" {
		t.Fatalf("resource ID = %q, want %q", generator.Resources[0].InstanceState.ID, "downtime-1")
	}
}

func decodeDowntimeScheduleTypedAttributes(t *testing.T, rawAttributes json.RawMessage) map[string]json.RawMessage {
	t.Helper()

	attributes := map[string]json.RawMessage{}
	if err := json.Unmarshal(rawAttributes, &attributes); err != nil {
		t.Fatalf("typed attributes unmarshal error: %v", err)
	}
	return attributes
}

func assertEmptyDowntimeScheduleItemList(t *testing.T, resource terraformutils.Resource, key string) {
	t.Helper()

	items, ok := resource.Item[key].([]interface{})
	if !ok {
		t.Fatalf("%s item type = %T, want []interface{}", key, resource.Item[key])
	}
	if len(items) != 0 {
		t.Fatalf("%s length = %d, want 0", key, len(items))
	}
}
