// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"net/http"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestCloudflareConnectivityDirectoryServiceResource(t *testing.T) {
	resource, ok := cloudflareConnectivityDirectoryServiceResource("account-123", cloudflareConnectivityDirectoryService{
		ServiceID: "service-456",
		Name:      "private-api",
		Type:      "http",
	})
	if !ok {
		t.Fatal("expected connectivity directory service resource")
	}
	if got := resource.InstanceInfo.Type; got != "cloudflare_connectivity_directory_service" {
		t.Fatalf("resource type = %q, want cloudflare_connectivity_directory_service", got)
	}
	if got := resource.InstanceState.ID; got != "service-456" {
		t.Fatalf("resource ID = %q, want service-456", got)
	}
	if got := resource.InstanceState.Attributes["account_id"]; got != "account-123" {
		t.Fatalf("account_id = %q, want account-123", got)
	}
	if got := resource.InstanceState.Attributes["service_id"]; got != "service-456" {
		t.Fatalf("service_id = %q, want service-456", got)
	}
	if got := resource.InstanceState.Attributes["name"]; got != "private-api" {
		t.Fatalf("name = %q, want private-api", got)
	}
	if got := resource.InstanceState.Attributes["type"]; got != "http" {
		t.Fatalf("type = %q, want http", got)
	}
	if got, want := resource.ResourceName, terraformutils.TfSanitize(cloudflareResourceName("account-123", "connectivity_directory_service", "private-api", "service-456")); got != want {
		t.Fatalf("resource name = %q, want %q", got, want)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "account-123/service-456" {
		t.Fatalf("import_id = %q, want account-123/service-456", got)
	}
}

func TestCloudflareConnectivityDirectoryServiceResourceSkipsMalformedServices(t *testing.T) {
	if _, ok := cloudflareConnectivityDirectoryServiceResource("", cloudflareConnectivityDirectoryService{ServiceID: "service-456"}); ok {
		t.Fatal("expected service without account ID to be skipped")
	}
	if _, ok := cloudflareConnectivityDirectoryServiceResource("account-123", cloudflareConnectivityDirectoryService{}); ok {
		t.Fatal("expected service without service ID to be skipped")
	}
	resource, ok := cloudflareConnectivityDirectoryServiceResource("account-123", cloudflareConnectivityDirectoryService{ID: "id-fallback"})
	if !ok {
		t.Fatal("expected legacy id fallback to be accepted")
	}
	if got := resource.InstanceState.Attributes["service_id"]; got != "id-fallback" {
		t.Fatalf("service_id = %q, want id-fallback", got)
	}
}

func TestAppendConnectivityDirectoryServiceResourcesPaginates(t *testing.T) {
	api := newCloudflareNetworkEdgeTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/account-123/connectivity/directory/services" {
			t.Fatalf("path = %q, want /accounts/account-123/connectivity/directory/services", r.URL.Path)
		}
		switch r.URL.Query().Get("cursor") {
		case "":
			if got := r.URL.Query().Get("page"); got != "1" {
				t.Fatalf("page query = %q, want 1", got)
			}
			writeCloudflareNetworkEdgeTestResponse(t, w, []map[string]string{
				{"service_id": "service-1", "name": "api", "type": "http"},
			}, map[string]interface{}{
				"cursors": map[string]string{"after": "cursor-2"},
			})
		case "cursor-2":
			writeCloudflareNetworkEdgeTestResponse(t, w, []map[string]string{
				{"service_id": "service-2", "name": "ssh", "type": "tcp"},
			}, map[string]interface{}{
				"cursors": map[string]string{},
			})
		default:
			t.Fatalf("cursor query = %q, want empty or cursor-2", r.URL.Query().Get("cursor"))
		}
	}))

	generator := &ConnectivityGenerator{}
	if err := generator.appendConnectivityDirectoryServiceResources(context.Background(), api, "account-123"); err != nil {
		t.Fatalf("appendConnectivityDirectoryServiceResources() error = %v", err)
	}
	if len(generator.Resources) != 2 {
		t.Fatalf("resource count = %d, want 2", len(generator.Resources))
	}
	if got := generator.Resources[0].InstanceState.Meta["import_id"]; got != "account-123/service-1" {
		t.Fatalf("first import_id = %q, want account-123/service-1", got)
	}
	if got := generator.Resources[1].InstanceState.Meta["import_id"]; got != "account-123/service-2" {
		t.Fatalf("second import_id = %q, want account-123/service-2", got)
	}
}
