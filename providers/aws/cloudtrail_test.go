// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"
	"github.com/chenrui333/terraformer/terraformutils"
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

	var resources []terraformutils.Resource
	for _, eds := range edsList {
		if eds.EventDataStoreArn == nil {
			continue
		}
		if eds.Status == types.EventDataStoreStatusPendingDeletion {
			continue
		}
		edsARN := *eds.EventDataStoreArn
		edsName := StringValue(eds.Name)
		if edsName == "" {
			edsName = arnLastSegment(edsARN, "/")
		}
		resources = append(resources, terraformutils.NewSimpleResource(
			edsARN, edsName, "aws_cloudtrail_event_data_store", "aws", cloudtrailAllowEmptyValues))
	}

	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if got := resources[0].InstanceState.ID; got != "arn:aws:cloudtrail:us-east-1:123456789012:eventdatastore/abc-123" {
		t.Fatalf("resource ID = %q, want ARN of enabled EDS", got)
	}
	if got := resources[0].ResourceName; got != "tfer--my-eds" {
		t.Fatalf("resource name = %q, want %q", got, "tfer--my-eds")
	}
}

func TestCloudTrailEventDataStoreFallbackName(t *testing.T) {
	arn := "arn:aws:cloudtrail:us-east-1:123456789012:eventdatastore/abc-123"
	got := arnLastSegment(arn, "/")
	want := "abc-123"
	if got != want {
		t.Fatalf("arnLastSegment() = %q, want %q", got, want)
	}
}

func strPtr(s string) *string { return &s }
