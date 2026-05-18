// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"net/http"
	"testing"

	cf "github.com/cloudflare/cloudflare-go"
)

func TestCloudflareAccessShortLivedCertificateResourceUsesScopedImportID(t *testing.T) {
	resource, ok := cloudflareAccessShortLivedCertificateResource(
		"accounts",
		"account-123",
		"app-456",
		cloudflareAccessShortLivedCertificate{ID: "ca-789"},
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
		cloudflareAccessShortLivedCertificate{ID: "app-456"},
	); ok {
		t.Fatal("expected missing scope ID to be skipped")
	}
}

func TestGetAccessShortLivedCertificateUsesApplicationPath(t *testing.T) {
	api := newCloudflareNetworkEdgeTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/account-123/access/apps/app-456/ca" {
			t.Fatalf("path = %q, want /accounts/account-123/access/apps/app-456/ca", r.URL.Path)
		}
		writeCloudflareNetworkEdgeTestResponse(t, w, map[string]string{"id": "ca-789"}, nil)
	}))

	certificate, err := getAccessShortLivedCertificate(context.Background(), api, cf.AccountIdentifier("account-123"), "app-456")
	if err != nil {
		t.Fatalf("getAccessShortLivedCertificate() error = %v", err)
	}
	if certificate.ID != "ca-789" {
		t.Fatalf("certificate ID = %q, want ca-789", certificate.ID)
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
