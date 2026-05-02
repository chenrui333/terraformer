// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	ldapi "github.com/launchdarkly/api-client-go/v22"
)

func TestViewClientUsesOnlyBetaAPIVersion(t *testing.T) {
	tests := []struct {
		name string
		run  func(context.Context, *ldapi.APIClient) error
	}{
		{
			name: "views",
			run: func(ctx context.Context, client *ldapi.APIClient) error {
				_, err := getViews(ctx, client, "project")
				return err
			},
		},
		{
			name: "linked resources",
			run: func(ctx context.Context, client *ldapi.APIClient) error {
				_, err := getViewLinkedResources(ctx, client, "project", "view", "flags")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var apiVersions []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				apiVersions = r.Header.Values("LD-API-Version")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"items":[],"totalCount":0}`))
			}))
			defer server.Close()

			client := newLaunchDarklyViewAPIClient()
			client.GetConfig().Servers = ldapi.ServerConfigurations{{URL: server.URL}}

			if err := tt.run(context.Background(), client); err != nil {
				t.Fatalf("view beta request returned error: %v", err)
			}

			if len(apiVersions) != 1 || apiVersions[0] != launchDarklyViewAPIVersion {
				t.Fatalf("expected only LD-API-Version %q, got %q", launchDarklyViewAPIVersion, apiVersions)
			}
		})
	}
}
