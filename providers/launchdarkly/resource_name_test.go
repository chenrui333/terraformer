// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import "testing"

func TestGeneratedResourceNamesIncludeStableIdentifiers(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "feature flag",
			got:  featureFlagResourceName("proj", "Duplicate", "flag-key"),
			want: "proj-Duplicate-flag-key",
		},
		{
			name: "environment",
			got:  environmentResourceName("proj", "Production", "prod"),
			want: "proj-Production-prod",
		},
		{
			name: "segment",
			got:  segmentResourceName("proj", "prod", "Duplicate", "segment-key"),
			want: "proj-prod-Duplicate-segment-key",
		},
		{
			name: "metric",
			got:  metricResourceName("proj", "Checkout", "checkout-count"),
			want: "proj-Checkout-checkout-count",
		},
		{
			name: "relay proxy configuration",
			got:  relayProxyConfigurationResourceName("Production", "rpc-123"),
			want: "Production-rpc-123",
		},
		{
			name: "team",
			got:  teamResourceName("Platform", "platform"),
			want: "Platform-platform",
		},
		{
			name: "team member",
			got:  teamMemberResourceName("user@example.com", "member-123"),
			want: "user@example.com-member-123",
		},
		{
			name: "webhook",
			got:  webhookResourceName("Deploy", "webhook-123"),
			want: "Deploy-webhook-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("resource name = %q, want %q", tt.got, tt.want)
			}
		})
	}
}
