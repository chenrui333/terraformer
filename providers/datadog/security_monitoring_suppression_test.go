// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"regexp"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/zclconf/go-cty/cty"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestSecurityMonitoringSuppressionAllowEmptyValuesPreservesQueries(t *testing.T) {
	allowEmptyValues := []*regexp.Regexp{}
	for _, pattern := range SecurityMonitoringSuppressionAllowEmptyValues {
		allowEmptyValues = append(allowEmptyValues, regexp.MustCompile(pattern))
	}

	parser := terraformutils.NewFlatmapParser(map[string]string{
		"rule_query":        "",
		"suppression_query": "",
	}, nil, allowEmptyValues)
	suppressionType := cty.Object(map[string]cty.Type{
		"rule_query":        cty.String,
		"suppression_query": cty.String,
	})

	result, err := parser.Parse(suppressionType)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if result["rule_query"] != "" {
		t.Fatalf("rule_query = %v, want empty string", result["rule_query"])
	}
	if result["suppression_query"] != "" {
		t.Fatalf("suppression_query = %v, want empty string", result["suppression_query"])
	}
}

func TestSecurityMonitoringSuppressionCreateResource(t *testing.T) {
	suppression := datadogV2.NewSecurityMonitoringSuppressionWithDefaults()
	suppression.SetId("suppression-id")

	generator := &SecurityMonitoringSuppressionGenerator{}
	resource, err := generator.createResource(*suppression)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "suppression-id" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "suppression-id")
	}
	if resource.ResourceName != "tfer--security_monitoring_suppression_suppression-id" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--security_monitoring_suppression_suppression-id")
	}
	if resource.InstanceInfo.Type != "datadog_security_monitoring_suppression" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_security_monitoring_suppression")
	}
}

func TestSecurityMonitoringSuppressionCreateResourceMissingID(t *testing.T) {
	generator := &SecurityMonitoringSuppressionGenerator{}
	_, err := generator.createResource(datadogV2.SecurityMonitoringSuppression{})
	if err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestSecurityMonitoringSuppressionCreateResources(t *testing.T) {
	firstSuppression := datadogV2.NewSecurityMonitoringSuppressionWithDefaults()
	firstSuppression.SetId("suppression-1")
	secondSuppression := datadogV2.NewSecurityMonitoringSuppressionWithDefaults()
	secondSuppression.SetId("suppression-2")

	generator := &SecurityMonitoringSuppressionGenerator{}
	resources, err := generator.createResources([]datadogV2.SecurityMonitoringSuppression{
		*firstSuppression,
		*secondSuppression,
	})
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

func TestSecurityMonitoringSuppressionsFromRawData(t *testing.T) {
	suppressions := securityMonitoringSuppressionsFromRawData([]interface{}{
		map[string]interface{}{
			"id":   "suppression-1",
			"type": "suppressions",
		},
		map[string]interface{}{
			"id": "suppression-2",
		},
		map[string]interface{}{
			"id":   "ignored-type",
			"type": "unknown",
		},
		map[string]interface{}{
			"type": "suppressions",
		},
	})

	if len(suppressions) != 2 {
		t.Fatalf("suppression count = %d, want %d", len(suppressions), 2)
	}
	if suppressions[0].GetId() != "suppression-1" {
		t.Fatalf("first suppression ID = %q, want %q", suppressions[0].GetId(), "suppression-1")
	}
	if suppressions[1].GetId() != "suppression-2" {
		t.Fatalf("second suppression ID = %q, want %q", suppressions[1].GetId(), "suppression-2")
	}
}

func TestSecurityMonitoringSuppressionFromRawDataRejectsNonObjects(t *testing.T) {
	if _, ok := securityMonitoringSuppressionFromRawData("suppression-id"); ok {
		t.Fatal("securityMonitoringSuppressionFromRawData accepted non-object raw data")
	}
}

func TestSecurityMonitoringSuppressionsHasNextPage(t *testing.T) {
	response := datadogV2.NewSecurityMonitoringPaginatedSuppressionsResponseWithDefaults()
	meta := datadogV2.NewSecurityMonitoringSuppressionsMetaWithDefaults()
	page := datadogV2.NewSecurityMonitoringSuppressionsPageMetaWithDefaults()
	page.SetTotalCount(101)
	meta.SetPage(*page)
	response.SetMeta(*meta)

	if !securityMonitoringSuppressionsHasNextPage(*response, 0, 100, 100) {
		t.Fatal("hasNextPage = false, want true")
	}
	if securityMonitoringSuppressionsHasNextPage(*response, 1, 100, 1) {
		t.Fatal("hasNextPage = true, want false")
	}
}
