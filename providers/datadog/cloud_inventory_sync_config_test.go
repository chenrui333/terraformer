// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"encoding/json"
	"testing"
)

func TestCloudInventorySyncConfigResponseDataListUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{
			name: "single data object",
			body: `{"data":{"id":"3526615b-4d65-4d9b-947a-d89c18faf0dc","attributes":{"cloud_provider":"aws"}}}`,
			want: 1,
		},
		{
			name: "data list",
			body: `{"data":[{"id":"aws","attributes":{"cloud_provider":"aws"}},{"id":"azure","attributes":{"cloud_provider":"azure"}}]}`,
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response cloudInventorySyncConfigResponse
			if err := json.Unmarshal([]byte(tt.body), &response); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if got := len(response.Data); got != tt.want {
				t.Fatalf("len(response.Data) = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCloudInventorySyncConfigCreateResource(t *testing.T) {
	generator := CloudInventorySyncConfigGenerator{}
	resource := generator.createResource(cloudInventorySyncConfigResponseData{
		ID: "3526615b-4d65-4d9b-947a-d89c18faf0dc",
		Attributes: cloudInventorySyncConfigResponseAttributes{
			CloudProvider: "aws",
		},
	})

	if resource.InstanceState.ID != "3526615b-4d65-4d9b-947a-d89c18faf0dc" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "3526615b-4d65-4d9b-947a-d89c18faf0dc")
	}
	if resource.ResourceName != "tfer--cloud_inventory_sync_config_aws" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--cloud_inventory_sync_config_aws")
	}
	if resource.InstanceInfo.Type != "datadog_cloud_inventory_sync_config" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_cloud_inventory_sync_config")
	}
}
