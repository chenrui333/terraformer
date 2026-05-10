// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestDynamoDBContributorInsightsImportID(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		indexName string
		accountID string
		want      string
	}{
		{name: "table", tableName: "events", accountID: "123456789012", want: "name:events/index:/123456789012"},
		{name: "index", tableName: "events", indexName: "by-user", accountID: "123456789012", want: "name:events/index:by-user/123456789012"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dynamodbContributorInsightsImportID(tt.tableName, tt.indexName, tt.accountID)
			if got != tt.want {
				t.Fatalf("dynamodbContributorInsightsImportID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDynamoDBKinesisStreamingDestinationImportID(t *testing.T) {
	got := dynamodbKinesisStreamingDestinationImportID("events", "arn:aws:kinesis:us-east-1:123456789012:stream/events")
	want := "events,arn:aws:kinesis:us-east-1:123456789012:stream/events"
	if got != want {
		t.Fatalf("dynamodbKinesisStreamingDestinationImportID() = %q, want %q", got, want)
	}
}

func TestDynamoDBResourceName(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "filters empty parts", parts: []string{"", "events", "", "policy"}, want: "events/policy"},
		{name: "preserves segment boundaries", parts: []string{"orders", "stream", "policy"}, want: "orders/stream/policy"},
		{name: "fallback", parts: nil, want: "dynamodb_resource"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dynamodbResourceName(tt.parts...)
			if got != tt.want {
				t.Fatalf("dynamodbResourceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDynamoDBContributorInsightsImportable(t *testing.T) {
	if !dynamodbContributorInsightsImportable(dynamodbtypes.ContributorInsightsSummary{
		ContributorInsightsStatus: dynamodbtypes.ContributorInsightsStatusEnabled,
	}) {
		t.Fatal("enabled contributor insights should be importable")
	}
	if dynamodbContributorInsightsImportable(dynamodbtypes.ContributorInsightsSummary{
		ContributorInsightsStatus: dynamodbtypes.ContributorInsightsStatusDisabled,
	}) {
		t.Fatal("disabled contributor insights should not be importable")
	}
	if dynamodbContributorInsightsImportable(dynamodbtypes.ContributorInsightsSummary{
		ContributorInsightsStatus: dynamodbtypes.ContributorInsightsStatusFailed,
	}) {
		t.Fatal("failed contributor insights should not be importable")
	}
	if dynamodbContributorInsightsImportable(dynamodbtypes.ContributorInsightsSummary{
		ContributorInsightsStatus: dynamodbtypes.ContributorInsightsStatusDisabling,
	}) {
		t.Fatal("disabling contributor insights should not be importable")
	}
	if dynamodbContributorInsightsImportable(dynamodbtypes.ContributorInsightsSummary{}) {
		t.Fatal("empty contributor insights status should not be importable")
	}
}

func TestDynamoDBKinesisStreamingDestinationImportable(t *testing.T) {
	if !dynamodbKinesisStreamingDestinationImportable(dynamodbtypes.KinesisDataStreamDestination{
		DestinationStatus: dynamodbtypes.DestinationStatusActive,
	}) {
		t.Fatal("active Kinesis streaming destination should be importable")
	}
	if dynamodbKinesisStreamingDestinationImportable(dynamodbtypes.KinesisDataStreamDestination{
		DestinationStatus: dynamodbtypes.DestinationStatusDisabled,
	}) {
		t.Fatal("disabled Kinesis streaming destination should not be importable")
	}
	if dynamodbKinesisStreamingDestinationImportable(dynamodbtypes.KinesisDataStreamDestination{
		DestinationStatus: dynamodbtypes.DestinationStatusEnableFailed,
	}) {
		t.Fatal("enable-failed Kinesis streaming destination should not be importable")
	}
	if dynamodbKinesisStreamingDestinationImportable(dynamodbtypes.KinesisDataStreamDestination{
		DestinationStatus: dynamodbtypes.DestinationStatusDisabling,
	}) {
		t.Fatal("disabling Kinesis streaming destination should not be importable")
	}
	if dynamodbKinesisStreamingDestinationImportable(dynamodbtypes.KinesisDataStreamDestination{}) {
		t.Fatal("empty Kinesis streaming destination status should not be importable")
	}
}

func TestDynamoDBTableExportImportable(t *testing.T) {
	if !dynamodbTableExportImportable(dynamodbtypes.ExportSummary{ExportStatus: dynamodbtypes.ExportStatusCompleted}) {
		t.Fatal("completed table export should be importable")
	}
	if !dynamodbTableExportImportable(dynamodbtypes.ExportSummary{ExportStatus: dynamodbtypes.ExportStatusFailed}) {
		t.Fatal("failed table export should be importable")
	}
	if dynamodbTableExportImportable(dynamodbtypes.ExportSummary{}) {
		t.Fatal("empty table export status should not be importable")
	}
}

func TestDynamoDBGlobalTableStatusImportable(t *testing.T) {
	if !dynamodbGlobalTableStatusImportable(dynamodbtypes.GlobalTableStatusActive) {
		t.Fatal("active global table should be importable")
	}
	for _, status := range []dynamodbtypes.GlobalTableStatus{
		dynamodbtypes.GlobalTableStatusCreating,
		dynamodbtypes.GlobalTableStatusDeleting,
		dynamodbtypes.GlobalTableStatusUpdating,
		"",
	} {
		if dynamodbGlobalTableStatusImportable(status) {
			t.Fatalf("global table status %q should not be importable", status)
		}
	}
}

func TestDynamoDBResourcePolicyTargets(t *testing.T) {
	table := dynamodbTableReference{
		name:      "events",
		tableARN:  "arn:aws:dynamodb:us-east-1:123456789012:table/events",
		streamARN: "arn:aws:dynamodb:us-east-1:123456789012:table/events/stream/2026-05-02T00:00:00.000",
	}
	targets := table.resourcePolicyTargets()
	if len(targets) != 2 {
		t.Fatalf("len(targets) = %d, want 2", len(targets))
	}
	if got, want := targets[0].name, "events"; got != want {
		t.Fatalf("table target name = %q, want %q", got, want)
	}
	if got, want := targets[1].name, "events/stream"; got != want {
		t.Fatalf("stream target name = %q, want %q", got, want)
	}
}

func TestDynamoDBResourceNamesAvoidSegmentCollisions(t *testing.T) {
	tablePolicy := dynamodbResourceName("orders_stream", "policy")
	streamPolicy := dynamodbResourceName("orders", "stream", "policy")
	if tablePolicy == streamPolicy {
		t.Fatalf("resource names collide: %q", tablePolicy)
	}
}

func TestDynamoDBResourcePolicyMissing(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "policy not found", err: &dynamodbtypes.PolicyNotFoundException{}, want: true},
		{name: "resource not found", err: &dynamodbtypes.ResourceNotFoundException{}, want: true},
		{name: "wrapped", err: &wrappedError{err: &dynamodbtypes.PolicyNotFoundException{}}, want: true},
		{name: "other", err: errors.New("boom"), want: false},
		{name: "nil", err: nil, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dynamodbResourcePolicyMissing(tt.err)
			if got != tt.want {
				t.Fatalf("dynamodbResourcePolicyMissing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDynamoDBPostConvertHookWrapsResourcePolicy(t *testing.T) {
	g := &DynamoDbGenerator{}
	g.Resources = append(g.Resources, terraformutilsNewResourceForTest(
		"arn:aws:dynamodb:us-east-1:123456789012:table/events",
		"events_policy",
		"aws_dynamodb_resource_policy",
		map[string]interface{}{"policy": "{\"Version\":\"2012-10-17\"}"},
	))
	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}
	got := g.Resources[0].Item["policy"]
	want := "<<POLICY\n{\"Version\":\"2012-10-17\"}\nPOLICY"
	if got != want {
		t.Fatalf("policy = %q, want %q", got, want)
	}
}

func TestDynamoDBPostConvertHookRemovesDisabledTTL(t *testing.T) {
	resource := terraformutilsNewResourceForTest("events", "events", "aws_dynamodb_table", map[string]interface{}{"ttl": []interface{}{map[string]interface{}{"enabled": false}}})
	resource.InstanceState.Attributes["ttl.0.enabled"] = "false"
	g := &DynamoDbGenerator{}
	g.Resources = append(g.Resources, resource)
	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}
	if _, ok := g.Resources[0].Item["ttl"]; ok {
		t.Fatal("ttl should be removed when disabled")
	}
}

type wrappedError struct {
	err error
}

func (e *wrappedError) Error() string {
	return e.err.Error()
}

func (e *wrappedError) Unwrap() error {
	return e.err
}

func terraformutilsNewResourceForTest(id, name, resourceType string, item map[string]interface{}) terraformutils.Resource {
	resource := terraformutils.NewSimpleResource(id, name, resourceType, "aws", dynamodbAllowEmptyValues)
	resource.Item = item
	return resource
}
