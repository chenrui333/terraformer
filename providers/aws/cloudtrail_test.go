// SPDX-License-Identifier: Apache-2.0

package aws

import "testing"

func TestCloudTrailEventDataStoreResource(t *testing.T) {
	arn := "arn:aws:cloudtrail:us-east-1:123456789012:eventdatastore/abc-123"
	resource := eventDataStoreToResource(arn, "my-eds")

	if got := resource.InstanceState.ID; got != arn {
		t.Fatalf("resource ID = %q, want %q", got, arn)
	}
	if got := resource.ResourceName; got != "tfer--my-eds" {
		t.Fatalf("resource name = %q, want %q", got, "tfer--my-eds")
	}
	if got := resource.InstanceInfo.Type; got != "aws_cloudtrail_event_data_store" {
		t.Fatalf("resource type = %q, want %q", got, "aws_cloudtrail_event_data_store")
	}
}

func TestCloudTrailEventDataStoreFallbackName(t *testing.T) {
	arn := "arn:aws:cloudtrail:us-east-1:123456789012:eventdatastore/abc-123"
	resource := eventDataStoreToResource(arn, "")

	if got := resource.ResourceName; got != "tfer--abc-123" {
		t.Fatalf("resource name = %q, want %q", got, "tfer--abc-123")
	}
}
