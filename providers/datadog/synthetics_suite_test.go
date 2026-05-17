// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestSyntheticsSuiteCreateResource(t *testing.T) {
	generator := &SyntheticsSuiteGenerator{}
	resource, err := generator.createResource("suite-123")
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "suite-123" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "suite-123")
	}
	if resource.ResourceName != "tfer--synthetics_suite_suite-123" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--synthetics_suite_suite-123")
	}
	if resource.InstanceInfo.Type != "datadog_synthetics_suite" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_synthetics_suite")
	}
}

func TestSyntheticsSuiteCreateResourceMissingID(t *testing.T) {
	generator := &SyntheticsSuiteGenerator{}
	_, err := generator.createResource("")
	if err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestSyntheticsSuiteCreateResources(t *testing.T) {
	firstSuite := datadogV2.NewSyntheticsSuiteWithDefaults()
	firstSuite.SetPublicId("suite-1")
	secondSuite := datadogV2.NewSyntheticsSuiteWithDefaults()
	secondSuite.SetPublicId("suite-2")

	generator := &SyntheticsSuiteGenerator{}
	resources, err := generator.createResources([]datadogV2.SyntheticsSuite{*firstSuite, *secondSuite})
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

func TestSyntheticsSuitePostConvertHookPreservesEmptyTags(t *testing.T) {
	resource := terraformutils.NewSimpleResource(
		"suite-1",
		"synthetics_suite_suite-1",
		"datadog_synthetics_suite",
		"datadog",
		SyntheticsSuiteAllowEmptyValues,
	)
	resource.Item = map[string]interface{}{
		"id":   "suite-1",
		"name": "suite one",
	}
	resource.InstanceState.Attributes = map[string]string{
		"id":     "suite-1",
		"name":   "suite one",
		"tags.#": "0",
	}
	resource.InstanceState.SetTypedAttributes(json.RawMessage("{\"id\":\"suite-1\",\"name\":\"suite one\",\"tags\":[]}"))

	generator := &SyntheticsSuiteGenerator{
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
	tags, ok := updatedResource.Item[syntheticsSuiteTagsKey].([]interface{})
	if !ok {
		t.Fatalf("tags item type = %T, want []interface{}", updatedResource.Item[syntheticsSuiteTagsKey])
	}
	if len(tags) != 0 {
		t.Fatalf("tags length = %d, want 0", len(tags))
	}
	if got := updatedResource.InstanceState.Attributes["tags.#"]; got != "0" {
		t.Fatalf("tags.# = %q, want 0", got)
	}
	typedAttributes := decodeSyntheticsSuiteTypedAttributes(t, updatedResource.InstanceState.TypedAttributes)
	if got := string(typedAttributes[syntheticsSuiteTagsKey]); got != "[]" {
		t.Fatalf("typed tags = %s, want []", got)
	}
}

func TestSyntheticsSuiteInitResourcesList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/synthetics/suites/search" {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("start"); got != "0" {
			t.Errorf("start query = %q, want %q", got, "0")
		}
		if got := r.URL.Query().Get("count"); got != fmt.Sprint(datadogSyntheticsSuitePageSize) {
			t.Errorf("count query = %q, want %q", got, fmt.Sprint(datadogSyntheticsSuitePageSize))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"data\":{\"attributes\":{\"suites\":[{\"public_id\":\"suite-1\",\"name\":\"suite one\",\"options\":{\"alerting_threshold\":0.5},\"tests\":[],\"type\":\"suite\"}],\"total\":1},\"type\":\"suites_search\"}}"))
	}))
	t.Cleanup(server.Close)

	config := datadog.NewConfiguration()
	config.Servers = datadog.ServerConfigurations{{URL: server.URL}}
	config.HTTPClient = server.Client()

	generator := &SyntheticsSuiteGenerator{
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
	if generator.Resources[0].InstanceState.ID != "suite-1" {
		t.Fatalf("resource ID = %q, want %q", generator.Resources[0].InstanceState.ID, "suite-1")
	}
}

func decodeSyntheticsSuiteTypedAttributes(t *testing.T, rawAttributes json.RawMessage) map[string]json.RawMessage {
	t.Helper()

	attributes := map[string]json.RawMessage{}
	if err := json.Unmarshal(rawAttributes, &attributes); err != nil {
		t.Fatalf("typed attributes unmarshal error: %v", err)
	}
	return attributes
}
