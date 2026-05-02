// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/dns/v1"
	"google.golang.org/api/option"
)

func TestCreateZonesResourcesReturnsManagedZoneListError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "{\"error\":{\"message\":\"service unavailable\"}}", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	dnsService := newTestDNSService(ctx, t, server.URL+"/")
	_, err := (CloudDNSGenerator{}).createZonesResources(ctx, dnsService, "test-project")
	if err == nil {
		t.Fatal("expected dns managed zone list error")
	}
	if !strings.Contains(err.Error(), "list dns managed zones") {
		t.Fatalf("expected wrapped dns managed zone list error, got %q", err)
	}
}

func TestCreateZonesResourcesReturnsRecordListError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/rrsets") {
			http.Error(w, "{\"error\":{\"message\":\"service unavailable\"}}", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"managedZones\":[{\"name\":\"test-zone\"}]}"))
	}))
	t.Cleanup(server.Close)

	dnsService := newTestDNSService(ctx, t, server.URL+"/")
	_, err := (CloudDNSGenerator{}).createZonesResources(ctx, dnsService, "test-project")
	if err == nil {
		t.Fatal("expected dns record list error")
	}
	if !strings.Contains(err.Error(), "list dns records for test-zone") {
		t.Fatalf("expected wrapped dns record list error, got %q", err)
	}
}

func newTestDNSService(ctx context.Context, t *testing.T, endpoint string) *dns.Service {
	t.Helper()

	dnsService, err := dns.NewService(ctx, option.WithEndpoint(endpoint), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}
	return dnsService
}
