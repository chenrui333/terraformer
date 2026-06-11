// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"net/url"
	"reflect"
	"sort"
	"testing"

	cf "github.com/chenrui333/terraformer/providers/cloudflare/internal/cloudflarev7"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestCloudflareProviderSupportedServices(t *testing.T) {
	services := (&CloudflareProvider{}).GetSupportedService()
	got := make([]string, 0, len(services))
	for service := range services {
		got = append(got, service)
	}
	sort.Strings(got)

	want := []string{
		"access",
		"account_member",
		"certificates",
		"connectivity",
		"dns",
		"email_routing",
		"firewall",
		"lists",
		"load_balancing",
		"logpush",
		"magic_wan",
		"media_platform",
		"network_edge",
		"notifications",
		"page_rule",
		"pages",
		"ruleset",
		"security",
		"settings",
		"storage",
		"tunnel",
		"turnstile",
		"waiting_room",
		"web_analytics",
		"workers",
		"zero_trust_device_dlp",
		"zero_trust_gateway",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("supported services = %#v, want %#v", got, want)
	}
}

func TestCloudflareProviderInitServiceConfiguresService(t *testing.T) {
	provider := &CloudflareProvider{}

	if err := provider.InitService("settings", true); err != nil {
		t.Fatalf("InitService() error = %v", err)
	}
	if provider.Service.GetName() != "settings" {
		t.Fatalf("service name = %q, want settings", provider.Service.GetName())
	}
	if provider.Service.GetProviderName() != "cloudflare" {
		t.Fatalf("provider name = %q, want cloudflare", provider.Service.GetProviderName())
	}
}

func TestCloudflareProviderInitServiceRejectsUnsupportedService(t *testing.T) {
	provider := &CloudflareProvider{}

	err := provider.InitService("missing", false)
	if err == nil {
		t.Fatal("expected unsupported service error")
	}
	if got, want := err.Error(), "cloudflare: missing not supported service"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
	if provider.Service != nil {
		t.Fatalf("service = %#v, want nil", provider.Service)
	}
}

func TestCloudflareAuthHelpersRequireCredentials(t *testing.T) {
	t.Setenv("CLOUDFLARE_API_KEY", "")
	t.Setenv("CLOUDFLARE_EMAIL", "")
	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	service := &CloudflareService{}

	if _, err := service.initializeAPI(); err == nil {
		t.Fatal("expected initializeAPI to require credentials")
	}
	if _, err := service.cloudflareV7Options(); err == nil {
		t.Fatal("expected cloudflareV7Options to require credentials")
	}

	t.Setenv("CLOUDFLARE_API_TOKEN", "token")
	if _, err := service.initializeAPI(); err != nil {
		t.Fatalf("initializeAPI() error = %v", err)
	}
	if options, err := service.cloudflareV7Options(); err != nil {
		t.Fatalf("cloudflareV7Options() error = %v", err)
	} else if len(options) != 1 {
		t.Fatalf("cloudflareV7Options() returned %d options, want 1", len(options))
	}
}

func TestCloudflareAccountIDRequired(t *testing.T) {
	service := &CloudflareService{}
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")

	if _, err := service.accountIDRequired(); err == nil {
		t.Fatal("expected account ID error")
	}

	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "account-123")
	if accountID, err := service.accountIDRequired(); err != nil {
		t.Fatalf("accountIDRequired() error = %v", err)
	} else if accountID != "account-123" {
		t.Fatalf("account ID = %q, want account-123", accountID)
	}
}

func TestCloudflareResourceAndImportIDHelpers(t *testing.T) {
	if got, want := cloudflareResourceName("account", "", "resource"), "account_resource"; got != want {
		t.Fatalf("cloudflareResourceName() = %q, want %q", got, want)
	}

	resource := terraformutils.NewSimpleResource("id", "name", "cloudflare_test", "cloudflare", nil)
	setCloudflareImportID(&resource, "account/id")
	if got := resource.InstanceState.Meta["import_id"]; got != "account/id" {
		t.Fatalf("import_id = %q, want account/id", got)
	}
}

func TestCloudflarePaginationHelpers(t *testing.T) {
	query, err := url.ParseQuery(cloudflarePaginationQuery(2, ""))
	if err != nil {
		t.Fatalf("parse page query: %v", err)
	}
	if query.Get("page") != "2" {
		t.Fatalf("page query = %q, want 2", query.Get("page"))
	}
	if query.Get("per_page") != "50" {
		t.Fatalf("per_page query = %q, want 50", query.Get("per_page"))
	}

	query, err = url.ParseQuery(cloudflarePaginationQuery(1, "cursor-123"))
	if err != nil {
		t.Fatalf("parse cursor query: %v", err)
	}
	if query.Get("cursor") != "cursor-123" {
		t.Fatalf("cursor query = %q, want cursor-123", query.Get("cursor"))
	}
	if query.Get("page") != "" {
		t.Fatalf("page query with cursor = %q, want empty", query.Get("page"))
	}

	page := 1
	cursor := ""
	if !cloudflareAdvancePagination(&cf.ResultInfo{Cursor: "next"}, &page, &cursor) {
		t.Fatal("expected cursor pagination to advance")
	}
	if cursor != "next" {
		t.Fatalf("cursor = %q, want next", cursor)
	}
	if cloudflareAdvancePagination(&cf.ResultInfo{Cursor: "next"}, &page, &cursor) {
		t.Fatal("expected repeated cursor to stop pagination")
	}

	page = 1
	cursor = ""
	if !cloudflareAdvancePaginationWithItemCount(&cf.ResultInfo{Page: 1, PerPage: 10}, &page, &cursor, 10) {
		t.Fatal("expected full page item count to advance pagination")
	}
	if page != 2 {
		t.Fatalf("page = %d, want 2", page)
	}
	if cloudflareAdvancePaginationWithItemCount(&cf.ResultInfo{Page: 2, PerPage: 10}, &page, &cursor, 9) {
		t.Fatal("expected short page item count to stop pagination")
	}
}
