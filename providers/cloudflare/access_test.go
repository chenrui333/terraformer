// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	cf "github.com/chenrui333/terraformer/providers/cloudflare/internal/cloudflarev7"
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

func TestCloudflareAccessShortLivedCertificateResourceUsesScopedImportID(t *testing.T) {
	resource, ok := cloudflareAccessShortLivedCertificateResource(
		"accounts",
		"account-123",
		"app-456",
	)
	if !ok {
		t.Fatal("expected Access short-lived certificate resource")
	}
	if resource.InstanceInfo.Type != "cloudflare_zero_trust_access_short_lived_certificate" {
		t.Fatalf("resource type = %q, want cloudflare_zero_trust_access_short_lived_certificate", resource.InstanceInfo.Type)
	}
	if got := resource.InstanceState.Attributes["account_id"]; got != "account-123" {
		t.Fatalf("account_id = %q, want account-123", got)
	}
	if got := resource.InstanceState.Attributes["app_id"]; got != "app-456" {
		t.Fatalf("app_id = %q, want app-456", got)
	}
	if got := resource.InstanceState.ID; got != "app-456" {
		t.Fatalf("resource ID = %q, want app-456", got)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "accounts/account-123/app-456" {
		t.Fatalf("import_id = %q, want accounts/account-123/app-456", got)
	}
	if _, ok := cloudflareAccessShortLivedCertificateResource(
		"zones",
		"",
		"app-456",
	); ok {
		t.Fatal("expected missing scope ID to be skipped")
	}
}

func TestGetAccessShortLivedCertificateUsesApplicationPath(t *testing.T) {
	api := newCloudflareNetworkEdgeTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/account-123/access/apps/app-456/ca" {
			t.Fatalf("path = %q, want /accounts/account-123/access/apps/app-456/ca", r.URL.Path)
		}
		writeCloudflareNetworkEdgeTestResponse(t, w, map[string]string{
			"aud":        "aud-789",
			"public_key": "-----BEGIN PUBLIC KEY-----",
		}, nil)
	}))

	certificate, err := getAccessShortLivedCertificate(context.Background(), api, cf.AccountIdentifier("account-123"), "app-456")
	if err != nil {
		t.Fatalf("getAccessShortLivedCertificate() error = %v", err)
	}
	if certificate.Aud != "aud-789" {
		t.Fatalf("certificate aud = %q, want aud-789", certificate.Aud)
	}
	if certificate.PublicKey != "-----BEGIN PUBLIC KEY-----" {
		t.Fatalf("certificate public key = %q, want -----BEGIN PUBLIC KEY-----", certificate.PublicKey)
	}
}

func TestCloudflareAccessShortLivedCertificateOptionalError(t *testing.T) {
	notFoundErr := cf.NewNotFoundError(&cf.Error{ErrorMessages: []string{"not found"}})
	if !cloudflareAccessShortLivedCertificateOptionalError(&notFoundErr) {
		t.Fatal("per-application CA not found should be optional")
	}
	requestErr := cf.NewRequestError(&cf.Error{ErrorMessages: []string{"missing permission"}})
	if !cloudflareAccessShortLivedCertificateOptionalError(&requestErr) {
		t.Fatal("permission-gated CA lookup should be optional")
	}
}

func TestAppendScopedAccessResourcesListsApplicationsOnce(t *testing.T) {
	appListCalls := 0
	api := newCloudflareNetworkEdgeTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/accounts/account-123/access/apps":
			appListCalls++
			writeCloudflareNetworkEdgeTestResponse(t, w, []map[string]string{{
				"id":   "app-456",
				"name": "internal-app",
			}}, nil)
		case "/accounts/account-123/access/groups",
			"/accounts/account-123/access/identity_providers",
			"/accounts/account-123/access/certificates",
			"/accounts/account-123/access/service_tokens":
			writeCloudflareNetworkEdgeTestResponse(t, w, []map[string]string{}, nil)
		case "/accounts/account-123/access/apps/app-456/ca":
			writeCloudflareNetworkEdgeTestResponse(t, w, map[string]string{
				"aud":        "aud-789",
				"public_key": "-----BEGIN PUBLIC KEY-----",
			}, nil)
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))

	var generator AccessGenerator
	if err := generator.appendScopedAccessResources(context.Background(), api, cf.AccountIdentifier("account-123"), "accounts"); err != nil {
		t.Fatalf("appendScopedAccessResources() error = %v", err)
	}
	if appListCalls != 1 {
		t.Fatalf("application list calls = %d, want 1", appListCalls)
	}
	shortLivedCertificates := 0
	for _, resource := range generator.Resources {
		if resource.InstanceInfo.Type != "cloudflare_zero_trust_access_short_lived_certificate" {
			continue
		}
		shortLivedCertificates++
		if got := resource.InstanceState.ID; got != "app-456" {
			t.Fatalf("short-lived certificate ID = %q, want app-456", got)
		}
		if got := resource.InstanceState.Meta["import_id"]; got != "accounts/account-123/app-456" {
			t.Fatalf("short-lived certificate import_id = %q, want accounts/account-123/app-456", got)
		}
	}
	if shortLivedCertificates != 1 {
		t.Fatalf("short-lived certificate resources = %d, want 1", shortLivedCertificates)
	}
}
