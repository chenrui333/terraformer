// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"net/http"
	"reflect"
	"testing"
)

func TestAppendAccountAccessInfrastructureTargetResources(t *testing.T) {
	api := newCloudflareSecurityTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/account-123/infrastructure/targets" {
			t.Fatalf("path = %q, want /accounts/account-123/infrastructure/targets", r.URL.Path)
		}
		switch r.URL.Query().Get("page") {
		case "1":
			writeCloudflareSecurityTestResponse(t, w, []map[string]string{
				{"id": "target-1", "hostname": "ssh.example.com"},
			}, map[string]int{
				"page":        1,
				"per_page":    cloudflarePageSize,
				"total_pages": 2,
			})
		case "2":
			writeCloudflareSecurityTestResponse(t, w, []map[string]string{
				{"id": "target-2", "hostname": "db.example.com"},
			}, map[string]int{
				"page":        2,
				"per_page":    cloudflarePageSize,
				"total_pages": 2,
			})
		default:
			t.Fatalf("page query = %q, want 1 or 2", r.URL.Query().Get("page"))
		}
	}))
	g := &AccessGenerator{}

	if err := g.appendAccountAccessInfrastructureTargetResources(context.Background(), api, "account-123"); err != nil {
		t.Fatalf("appendAccountAccessInfrastructureTargetResources() error = %v", err)
	}

	got := map[string]string{}
	for _, resource := range g.Resources {
		if resource.InstanceInfo.Type != "cloudflare_zero_trust_access_infrastructure_target" {
			t.Fatalf("resource type = %q, want cloudflare_zero_trust_access_infrastructure_target", resource.InstanceInfo.Type)
		}
		if gotAccountID := resource.InstanceState.Attributes["account_id"]; gotAccountID != "account-123" {
			t.Fatalf("account_id = %q, want account-123", gotAccountID)
		}
		got[resource.InstanceState.ID] = resource.InstanceState.Meta["import_id"].(string)
	}
	want := map[string]string{
		"target-1": "account-123/target-1",
		"target-2": "account-123/target-2",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resources = %#v, want %#v", got, want)
	}
}
