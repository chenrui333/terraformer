// SPDX-License-Identifier: Apache-2.0

package cloudflarev7

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRawPreservesQueryStringAndAuthHeader(t *testing.T) {
	api := newTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/account-123/workers/workers" {
			t.Fatalf("path = %q, want /accounts/account-123/workers/workers", r.URL.Path)
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Fatalf("page query = %q, want 2", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want Bearer test-token", got)
		}
		writeTestResponse(t, w, []map[string]string{{"id": "worker-1"}}, map[string]int{
			"page":        2,
			"per_page":    50,
			"total_pages": 2,
		})
	}))

	response, err := api.Raw(context.Background(), http.MethodGet, "/accounts/account-123/workers/workers?page=2&per_page=50", nil, nil)
	if err != nil {
		t.Fatalf("Raw() error = %v", err)
	}
	if response.ResultInfo == nil || response.ResultInfo.Page != 2 {
		t.Fatalf("ResultInfo = %#v, want page 2", response.ResultInfo)
	}
}

func TestListCertificateAuthoritiesHostnameAssociationsAcceptsWrappedResult(t *testing.T) {
	api := newTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/zones/zone-123/certificate_authorities/hostname_associations" {
			t.Fatalf("path = %q, want hostname association path", r.URL.Path)
		}
		if got := r.URL.Query().Get("mtls_certificate_id"); got != "cert-123" {
			t.Fatalf("mtls_certificate_id = %q, want cert-123", got)
		}
		writeTestResponse(t, w, map[string][]string{
			"hostnames": {"api.example.com", "admin.example.com"},
		}, nil)
	}))

	hostnames, err := api.ListCertificateAuthoritiesHostnameAssociations(
		context.Background(),
		ZoneIdentifier("zone-123"),
		ListCertificateAuthoritiesHostnameAssociationsParams{MTLSCertificateID: "cert-123"},
	)
	if err != nil {
		t.Fatalf("ListCertificateAuthoritiesHostnameAssociations() error = %v", err)
	}
	if got, want := len(hostnames), 2; got != want {
		t.Fatalf("hostname count = %d, want %d", got, want)
	}
	if hostnames[0] != "api.example.com" || hostnames[1] != "admin.example.com" {
		t.Fatalf("hostnames = %#v, want wrapped hostnames", hostnames)
	}
}

func TestRawMapsHTTPErrorToTypedCloudflareError(t *testing.T) {
	api := newTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		writeTestResponse(t, w, nil, nil, ResponseInfo{Message: "missing permission"})
	}))

	_, err := api.Raw(context.Background(), http.MethodGet, "/accounts/account-123/forbidden", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var authorizationErr *AuthorizationError
	if !errors.As(err, &authorizationErr) {
		t.Fatalf("error type = %T, want *AuthorizationError", err)
	}
	if got := authorizationErr.ErrorMessages(); len(got) != 1 || got[0] != "missing permission" {
		t.Fatalf("ErrorMessages() = %#v, want missing permission", got)
	}
}

func TestResultInfoHasMorePagesUsesTotalCountFallback(t *testing.T) {
	for _, tt := range []struct {
		name string
		info ResultInfo
		want bool
	}{
		{
			name: "more pages from total count",
			info: ResultInfo{Page: 1, PerPage: 100, Total: 101},
			want: true,
		},
		{
			name: "last page from total count",
			info: ResultInfo{Page: 2, PerPage: 100, Total: 101},
			want: false,
		},
		{
			name: "total pages wins",
			info: ResultInfo{Page: 1, PerPage: 100, TotalPages: 1, Total: 101},
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.HasMorePages(); got != tt.want {
				t.Fatalf("HasMorePages() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestListDNSRecordsAutoPaginatesWhenPaginationOmitted(t *testing.T) {
	pages := make([]string, 0, 2)
	api := newTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/zones/zone-123/dns_records" {
			t.Fatalf("path = %q, want DNS records path", r.URL.Path)
		}
		page := r.URL.Query().Get("page")
		if page == "" {
			page = "1"
		}
		pages = append(pages, page)
		if got := r.URL.Query().Get("per_page"); got != "100" {
			t.Fatalf("per_page query = %q, want 100", got)
		}

		switch page {
		case "1":
			writeTestResponse(t, w, []DNSRecord{{ID: "record-1", Name: "a.example.com", Type: "A"}}, map[string]int{
				"page":        1,
				"per_page":    100,
				"total_count": 101,
			})
		case "2":
			writeTestResponse(t, w, []DNSRecord{{ID: "record-2", Name: "b.example.com", Type: "A"}}, map[string]int{
				"page":        2,
				"per_page":    100,
				"total_count": 101,
			})
		default:
			t.Fatalf("unexpected page query %q", page)
		}
	}))

	records, info, err := api.ListDNSRecords(context.Background(), ZoneIdentifier("zone-123"), ListDNSRecordsParams{})
	if err != nil {
		t.Fatalf("ListDNSRecords() error = %v", err)
	}
	if got, want := len(records), 2; got != want {
		t.Fatalf("record count = %d, want %d", got, want)
	}
	if records[0].ID != "record-1" || records[1].ID != "record-2" {
		t.Fatalf("records = %#v, want records from both pages", records)
	}
	if info == nil || info.Page != 2 {
		t.Fatalf("ResultInfo = %#v, want final page 2", info)
	}
	if got, want := len(pages), 2; got != want {
		t.Fatalf("request count = %d, want %d", got, want)
	}
}

func TestListDNSRecordsDoesNotAutoPaginateWhenPaginationExplicit(t *testing.T) {
	requestCount := 0
	api := newTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if got := r.URL.Query().Get("page"); got != "1" {
			t.Fatalf("page query = %q, want 1", got)
		}
		if got := r.URL.Query().Get("per_page"); got != "1" {
			t.Fatalf("per_page query = %q, want 1", got)
		}
		writeTestResponse(t, w, []DNSRecord{{ID: "record-1", Name: "a.example.com", Type: "A"}}, map[string]int{
			"page":        1,
			"per_page":    1,
			"total_pages": 2,
		})
	}))

	records, info, err := api.ListDNSRecords(
		context.Background(),
		ZoneIdentifier("zone-123"),
		ListDNSRecordsParams{ResultInfo: ResultInfo{Page: 1, PerPage: 1}},
	)
	if err != nil {
		t.Fatalf("ListDNSRecords() error = %v", err)
	}
	if got, want := len(records), 1; got != want {
		t.Fatalf("record count = %d, want %d", got, want)
	}
	if info == nil || !info.HasMorePages() {
		t.Fatalf("ResultInfo = %#v, want single explicit page result with more pages", info)
	}
	if requestCount != 1 {
		t.Fatalf("request count = %d, want 1", requestCount)
	}
}

func TestListDNSRecordsDoesNotForcePageForExplicitCursor(t *testing.T) {
	api := newTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("page"); got != "" {
			t.Fatalf("page query = %q, want empty for cursor pagination", got)
		}
		if got := r.URL.Query().Get("after"); got != "cursor-after" {
			t.Fatalf("after query = %q, want cursor-after", got)
		}
		if got := r.URL.Query().Get("per_page"); got != "100" {
			t.Fatalf("per_page query = %q, want 100", got)
		}
		writeTestResponse(t, w, []DNSRecord{{ID: "record-1", Name: "a.example.com", Type: "A"}}, nil)
	}))

	records, _, err := api.ListDNSRecords(
		context.Background(),
		ZoneIdentifier("zone-123"),
		ListDNSRecordsParams{ResultInfo: ResultInfo{Cursors: ResultInfoCursors{After: "cursor-after"}}},
	)
	if err != nil {
		t.Fatalf("ListDNSRecords() error = %v", err)
	}
	if got, want := len(records), 1; got != want {
		t.Fatalf("record count = %d, want %d", got, want)
	}
}

func TestListZonesAutoPaginatesWhenNamesOmitted(t *testing.T) {
	pages := make([]string, 0, 2)
	api := newTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/zones" {
			t.Fatalf("path = %q, want /zones", r.URL.Path)
		}
		if got := r.URL.Query().Get("per_page"); got != "50" {
			t.Fatalf("per_page query = %q, want 50", got)
		}
		page := r.URL.Query().Get("page")
		pages = append(pages, page)
		switch page {
		case "1":
			writeTestResponse(t, w, []Zone{{ID: "zone-1", Name: "example.com"}}, map[string]int{
				"page":        1,
				"per_page":    50,
				"total_pages": 2,
			})
		case "2":
			writeTestResponse(t, w, []Zone{{ID: "zone-2", Name: "example.org"}}, map[string]int{
				"page":        2,
				"per_page":    50,
				"total_pages": 2,
			})
		default:
			t.Fatalf("unexpected page query %q", page)
		}
	}))

	zones, err := api.ListZones(context.Background())
	if err != nil {
		t.Fatalf("ListZones() error = %v", err)
	}
	if got, want := len(zones), 2; got != want {
		t.Fatalf("zone count = %d, want %d", got, want)
	}
	if zones[0].ID != "zone-1" || zones[1].ID != "zone-2" {
		t.Fatalf("zones = %#v, want both pages", zones)
	}
	if got, want := len(pages), 2; got != want {
		t.Fatalf("request count = %d, want %d", got, want)
	}
}

func TestFirewallListMethodsAutoPaginateWhenPaginationOmitted(t *testing.T) {
	requests := map[string][]string{}
	api := newTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("per_page"); got != "50" {
			t.Fatalf("per_page query = %q, want 50", got)
		}
		page := r.URL.Query().Get("page")
		requests[r.URL.Path] = append(requests[r.URL.Path], page)
		var result []map[string]string
		switch page {
		case "1":
			result = []map[string]string{{"id": "first"}}
		case "2":
			result = []map[string]string{{"id": "second"}}
		default:
			t.Fatalf("unexpected page query %q for %s", page, r.URL.Path)
		}
		writeTestResponse(t, w, result, map[string]int{
			"page":        mustAtoi(t, page),
			"per_page":    50,
			"total_pages": 2,
		})
	}))

	lockdowns, info, err := api.ListZoneLockdowns(context.Background(), ZoneIdentifier("zone-123"), LockdownListParams{})
	if err != nil {
		t.Fatalf("ListZoneLockdowns() error = %v", err)
	}
	if got, want := len(lockdowns), 2; got != want || lockdowns[0].ID != "first" || lockdowns[1].ID != "second" {
		t.Fatalf("lockdowns = %#v, want both pages", lockdowns)
	}
	if info == nil || info.Page != 2 {
		t.Fatalf("lockdown ResultInfo = %#v, want page 2", info)
	}

	filters, info, err := api.Filters(context.Background(), ZoneIdentifier("zone-123"), FilterListParams{})
	if err != nil {
		t.Fatalf("Filters() error = %v", err)
	}
	if got, want := len(filters), 2; got != want || filters[0].ID != "first" || filters[1].ID != "second" {
		t.Fatalf("filters = %#v, want both pages", filters)
	}
	if info == nil || info.Page != 2 {
		t.Fatalf("filters ResultInfo = %#v, want page 2", info)
	}

	rules, info, err := api.FirewallRules(context.Background(), ZoneIdentifier("zone-123"), FirewallRuleListParams{})
	if err != nil {
		t.Fatalf("FirewallRules() error = %v", err)
	}
	if got, want := len(rules), 2; got != want || rules[0].ID != "first" || rules[1].ID != "second" {
		t.Fatalf("firewall rules = %#v, want both pages", rules)
	}
	if info == nil || info.Page != 2 {
		t.Fatalf("firewall rules ResultInfo = %#v, want page 2", info)
	}

	for _, path := range []string{
		"/zones/zone-123/firewall/lockdowns",
		"/zones/zone-123/filters",
		"/zones/zone-123/firewall/rules",
	} {
		if got := requests[path]; len(got) != 2 || got[0] != "1" || got[1] != "2" {
			t.Fatalf("requests[%s] = %#v, want pages 1 and 2", path, got)
		}
	}
}

func TestListAllRateLimitsPaginatesUntilShortPage(t *testing.T) {
	pages := make([]string, 0, 2)
	api := newTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/zones/zone-123/rate_limits" {
			t.Fatalf("path = %q, want rate limits path", r.URL.Path)
		}
		if got := r.URL.Query().Get("per_page"); got != "100" {
			t.Fatalf("per_page query = %q, want 100", got)
		}
		page := r.URL.Query().Get("page")
		pages = append(pages, page)
		switch page {
		case "1":
			writeTestResponse(t, w, []RateLimit{{ID: "limit-1"}}, map[string]int{
				"count":    100,
				"page":     1,
				"per_page": 100,
			})
		case "2":
			writeTestResponse(t, w, []RateLimit{{ID: "limit-2"}}, map[string]int{
				"count":    1,
				"page":     2,
				"per_page": 100,
			})
		default:
			t.Fatalf("unexpected page query %q", page)
		}
	}))

	limits, err := api.ListAllRateLimits(context.Background(), "zone-123")
	if err != nil {
		t.Fatalf("ListAllRateLimits() error = %v", err)
	}
	if got, want := len(limits), 2; got != want {
		t.Fatalf("rate limit count = %d, want %d", got, want)
	}
	if limits[0].ID != "limit-1" || limits[1].ID != "limit-2" {
		t.Fatalf("rate limits = %#v, want both pages", limits)
	}
	if got, want := len(pages), 2; got != want {
		t.Fatalf("request count = %d, want %d", got, want)
	}
}

func TestListMagicTransitResourcesUnwrapResultObjects(t *testing.T) {
	api := newTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/accounts/account-123/magic/gre_tunnels":
			writeTestResponse(t, w, map[string][]MagicTransitGRETunnel{
				"gre_tunnels": {{ID: "gre-1", Name: "gre tunnel"}},
			}, nil)
		case "/accounts/account-123/magic/ipsec_tunnels":
			writeTestResponse(t, w, map[string][]MagicTransitIPsecTunnel{
				"ipsec_tunnels": {{ID: "ipsec-1", Name: "ipsec tunnel"}},
			}, nil)
		case "/accounts/account-123/magic/routes":
			writeTestResponse(t, w, map[string][]MagicTransitStaticRoute{
				"routes": {{ID: "route-1", Prefix: "192.0.2.0/24"}},
			}, nil)
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))

	greTunnels, err := api.ListMagicTransitGRETunnels(context.Background(), "account-123")
	if err != nil {
		t.Fatalf("ListMagicTransitGRETunnels() error = %v", err)
	}
	if got, want := len(greTunnels), 1; got != want || greTunnels[0].ID != "gre-1" {
		t.Fatalf("GRE tunnels = %#v, want gre-1", greTunnels)
	}

	ipsecTunnels, err := api.ListMagicTransitIPsecTunnels(context.Background(), "account-123")
	if err != nil {
		t.Fatalf("ListMagicTransitIPsecTunnels() error = %v", err)
	}
	if got, want := len(ipsecTunnels), 1; got != want || ipsecTunnels[0].ID != "ipsec-1" {
		t.Fatalf("IPsec tunnels = %#v, want ipsec-1", ipsecTunnels)
	}

	routes, err := api.ListMagicTransitStaticRoutes(context.Background(), "account-123")
	if err != nil {
		t.Fatalf("ListMagicTransitStaticRoutes() error = %v", err)
	}
	if got, want := len(routes), 1; got != want || routes[0].ID != "route-1" {
		t.Fatalf("routes = %#v, want route-1", routes)
	}
}

func TestListPagesProjectsAcceptsStringDomains(t *testing.T) {
	api := newTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/account-123/pages/projects" {
			t.Fatalf("path = %q, want Pages projects path", r.URL.Path)
		}
		writeTestResponse(t, w, []map[string]interface{}{
			{
				"id":                "project-1",
				"name":              "example-pages",
				"subdomain":         "example-pages.pages.dev",
				"domains":           []string{"www.example.com"},
				"production_branch": "main",
			},
		}, map[string]int{
			"page":        1,
			"per_page":    50,
			"total_pages": 1,
		})
	}))

	projects, _, err := api.ListPagesProjects(
		context.Background(),
		AccountIdentifier("account-123"),
		ListPagesProjectsParams{PaginationOptions: PaginationOptions{Page: 1, PerPage: 50}},
	)
	if err != nil {
		t.Fatalf("ListPagesProjects() error = %v", err)
	}
	if got, want := len(projects), 1; got != want {
		t.Fatalf("project count = %d, want %d", got, want)
	}
	if got := projects[0].Domains; len(got) != 1 || got[0] != "www.example.com" {
		t.Fatalf("domains = %#v, want string domain", got)
	}
}

func newTestAPI(t *testing.T, handler http.Handler) *API {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	api, err := NewWithAPIToken("test-token", BaseURL(server.URL), UsingRetryPolicy(0, 0, 0))
	if err != nil {
		t.Fatalf("NewWithAPIToken() error = %v", err)
	}
	return api
}

func writeTestResponse(t *testing.T, w http.ResponseWriter, result interface{}, resultInfo interface{}, errors ...ResponseInfo) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")

	payload := map[string]interface{}{
		"success": len(errors) == 0,
		"result":  result,
	}
	if resultInfo != nil {
		payload["result_info"] = resultInfo
	}
	if len(errors) > 0 {
		payload["errors"] = errors
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func mustAtoi(t *testing.T, value string) int {
	t.Helper()
	if value == "1" {
		return 1
	}
	if value == "2" {
		return 2
	}
	t.Fatalf("unexpected integer string %q", value)
	return 0
}
