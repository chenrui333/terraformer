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
