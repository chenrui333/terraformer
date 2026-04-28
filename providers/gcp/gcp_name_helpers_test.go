// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"strings"
	"testing"
)

// These tests validate the GCP resource name extraction conventions used
// across providers. The parsing pattern (split on "/" and take last segment)
// is inlined in generated code, so these tests serve as convention guards
// rather than direct production-code callers.

func TestGCPResourceNameLastSegment(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
		want     string
	}{
		{"cloud function", "projects/p/locations/us-central1/functions/my-func", "my-func"},
		{"pubsub subscription", "projects/p/subscriptions/my-sub", "my-sub"},
		{"pubsub topic", "projects/p/topics/my-topic", "my-topic"},
		{"cloud tasks queue", "projects/p/locations/us-central1/queues/my-queue", "my-queue"},
		{"memorystore instance", "projects/p/locations/us-central1/instances/my-redis", "my-redis"},
		{"scheduler job", "projects/p/locations/us-central1/jobs/my-job", "my-job"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parts := strings.Split(tc.fullName, "/")
			got := parts[len(parts)-1]
			if got != tc.want {
				t.Errorf("last segment of %q = %q, want %q", tc.fullName, got, tc.want)
			}
		})
	}
}

func TestBigQueryDatasetIDExtraction(t *testing.T) {
	tests := []struct {
		name   string
		fullID string
		want   string
	}{
		{"standard dataset", "my-project:my_dataset", "my_dataset"},
		{"dataset with numbers", "project-123:dataset_2024", "dataset_2024"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.Split(tc.fullID, ":")[1]
			if got != tc.want {
				t.Errorf("dataset name from %q = %q, want %q", tc.fullID, got, tc.want)
			}
		})
	}
}

func TestKMSKeyRingIDComposition(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
		wantID   string
	}{
		{"global key ring", "projects/my-project/locations/global/keyRings/my-ring", "my-project/global/my-ring"},
		{"regional key ring", "projects/my-project/locations/us-east1/keyRings/prod-ring", "my-project/us-east1/prod-ring"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parts := strings.Split(tc.fullName, "/")
			got := parts[1] + "/" + parts[3] + "/" + parts[5]
			if got != tc.wantID {
				t.Errorf("KMS key ring ID from %q = %q, want %q", tc.fullName, got, tc.wantID)
			}
		})
	}
}

func TestGCPZoneFromLink(t *testing.T) {
	tests := []struct {
		name     string
		zoneLink string
		want     string
	}{
		{"us zone", "https://www.googleapis.com/compute/v1/projects/p/zones/us-central1-a", "us-central1-a"},
		{"eu zone", "https://www.googleapis.com/compute/v1/projects/p/zones/europe-west1-b", "europe-west1-b"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parts := strings.Split(tc.zoneLink, "/")
			got := parts[len(parts)-1]
			if got != tc.want {
				t.Errorf("zone from %q = %q, want %q", tc.zoneLink, got, tc.want)
			}
		})
	}
}
