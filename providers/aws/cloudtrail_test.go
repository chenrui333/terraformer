// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"
)

func TestCloudTrailEventDataStoreSkipsPendingDeletion(t *testing.T) {
	edsList := []types.EventDataStore{
		{
			EventDataStoreArn: strPtr("arn:aws:cloudtrail:us-east-1:123456789012:eventdatastore/abc-123"),
			Name:              strPtr("my-eds"),
			Status:            types.EventDataStoreStatusEnabled,
		},
		{
			EventDataStoreArn: strPtr("arn:aws:cloudtrail:us-east-1:123456789012:eventdatastore/del-456"),
			Name:              strPtr("deleting-eds"),
			Status:            types.EventDataStoreStatusPendingDeletion,
		},
	}

	var count int
	for _, eds := range edsList {
		if resource, ok := eventDataStoreToResource(eds); ok {
			count++
			if got := resource.InstanceState.ID; got != "arn:aws:cloudtrail:us-east-1:123456789012:eventdatastore/abc-123" {
				t.Fatalf("resource ID = %q, want ARN of enabled EDS", got)
			}
			if got := resource.ResourceName; got != "tfer--my-eds" {
				t.Fatalf("resource name = %q, want %q", got, "tfer--my-eds")
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 resource, got %d", count)
	}
}

func TestCloudTrailEventDataStoreNilARNSkipped(t *testing.T) {
	eds := types.EventDataStore{
		EventDataStoreArn: nil,
		Name:              strPtr("no-arn"),
		Status:            types.EventDataStoreStatusEnabled,
	}
	if _, ok := eventDataStoreToResource(eds); ok {
		t.Fatal("expected nil ARN to be skipped")
	}
}

func TestCloudTrailEventDataStoreFallbackName(t *testing.T) {
	arn := "arn:aws:cloudtrail:us-east-1:123456789012:eventdatastore/abc-123"
	resource, ok := eventDataStoreToResource(types.EventDataStore{
		EventDataStoreArn: &arn,
		Name:              strPtr(""),
		Status:            types.EventDataStoreStatusEnabled,
	})
	if !ok {
		t.Fatal("expected resource to be emitted")
	}
	if got := resource.ResourceName; got != "tfer--abc-123" {
		t.Fatalf("resource name = %q, want %q", got, "tfer--abc-123")
	}
}

func strPtr(s string) *string { return &s }
