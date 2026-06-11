// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"testing"

	cf "github.com/chenrui333/terraformer/providers/cloudflare/internal/cloudflarev7"
)

func TestCloudflareZoneCertificateResourceUsesCompositeImportID(t *testing.T) {
	zone := cf.Zone{ID: "zone-123", Name: "example.com"}
	resource, ok := cloudflareCertificatePackResource(zone, cloudflareCertificateRawResource{
		"id":                    "pack-456",
		"certificate_authority": "lets_encrypt",
		"type":                  "advanced",
		"validation_method":     "txt",
		"validity_days":         float64(90),
		"status":                "active",
	})
	if !ok {
		t.Fatal("expected certificate pack resource")
	}
	if resource.InstanceInfo.Type != "cloudflare_certificate_pack" {
		t.Fatalf("resource type = %q, want cloudflare_certificate_pack", resource.InstanceInfo.Type)
	}
	if got := resource.InstanceState.ID; got != "pack-456" {
		t.Fatalf("resource ID = %q, want pack-456", got)
	}
	if got := resource.InstanceState.Attributes["zone_id"]; got != "zone-123" {
		t.Fatalf("zone_id = %q, want zone-123", got)
	}
	if got := resource.InstanceState.Attributes["validity_days"]; got != "90" {
		t.Fatalf("validity_days = %q, want 90", got)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "zone-123/pack-456" {
		t.Fatalf("import_id = %q, want zone-123/pack-456", got)
	}
}

func TestCloudflareCertificatePackResourcePreservesBranding(t *testing.T) {
	for _, cloudflareBranding := range []bool{true, false} {
		resource, ok := cloudflareCertificatePackResource(cf.Zone{ID: "zone-123", Name: "example.com"}, cloudflareCertificateRawResource{
			"id":                    "pack-456",
			"certificate_authority": "lets_encrypt",
			"type":                  "advanced",
			"validation_method":     "txt",
			"validity_days":         float64(90),
			"cloudflare_branding":   cloudflareBranding,
			"status":                "active",
		})
		if !ok {
			t.Fatal("expected certificate pack resource")
		}
		want := strconv.FormatBool(cloudflareBranding)
		if got := resource.InstanceState.Attributes["cloudflare_branding"]; got != want {
			t.Fatalf("cloudflare_branding attribute = %q, want %s", got, want)
		}
		if got := resource.AdditionalFields["cloudflare_branding"]; got != cloudflareBranding {
			t.Fatalf("cloudflare_branding AdditionalFields = %#v, want %t", got, cloudflareBranding)
		}
	}
}

func TestCloudflareOriginCACertificateResourcePreservesHostnameOrder(t *testing.T) {
	resource, ok := cloudflareOriginCACertificateResource(cf.Zone{ID: "zone-123", Name: "example.com"}, cloudflareCertificateRawResource{
		"id":           "origin-456",
		"csr":          "-----BEGIN CERTIFICATE REQUEST-----",
		"request_type": "origin-rsa",
		"hostnames":    []interface{}{"z.example.com", "a.example.com"},
	})
	if !ok {
		t.Fatal("expected origin CA certificate resource")
	}
	if got := resource.InstanceState.Attributes["hostnames.0"]; got != "z.example.com" {
		t.Fatalf("hostnames.0 = %q, want z.example.com", got)
	}
	if got := resource.InstanceState.Attributes["hostnames.1"]; got != "a.example.com" {
		t.Fatalf("hostnames.1 = %q, want a.example.com", got)
	}
	hostnames, ok := resource.AdditionalFields["hostnames"].([]interface{})
	if !ok {
		t.Fatalf("AdditionalFields[hostnames] = %#v, want []interface{}", resource.AdditionalFields["hostnames"])
	}
	if got := hostnames[0]; got != "z.example.com" {
		t.Fatalf("AdditionalFields[hostnames][0] = %#v, want z.example.com", got)
	}
	if got := hostnames[1]; got != "a.example.com" {
		t.Fatalf("AdditionalFields[hostnames][1] = %#v, want a.example.com", got)
	}
}

func TestCloudflareCertificateOptionalDiscoveryErrorHandlesAuthErrors(t *testing.T) {
	authenticationErr := cf.NewAuthenticationError(&cf.Error{ErrorMessages: []string{"not authorized"}})
	if !cloudflareCertificateOptionalDiscoveryError(&authenticationErr) {
		t.Fatal("permission-gated authentication errors should be treated as optional")
	}

	authenticationErrWithoutMarker := cf.NewAuthenticationError(&cf.Error{ErrorMessages: []string{"invalid token"}})
	if cloudflareCertificateOptionalDiscoveryError(&authenticationErrWithoutMarker) {
		t.Fatal("credential authentication errors should propagate")
	}
}

func TestCloudflareCertificateAuthorityHostnameAssociationsResource(t *testing.T) {
	resource, ok := cloudflareCertificateAuthorityHostnameAssociationsResource(
		cf.Zone{ID: "zone-123", Name: "example.com"},
		"mtls-456",
		[]cf.HostnameAssociation{"api.example.com", "admin.example.com"},
	)
	if !ok {
		t.Fatal("expected hostname association resource")
	}
	if resource.InstanceInfo.Type != "cloudflare_certificate_authorities_hostname_associations" {
		t.Fatalf("resource type = %q, want cloudflare_certificate_authorities_hostname_associations", resource.InstanceInfo.Type)
	}
	if got := resource.InstanceState.ID; got != "zone-123" {
		t.Fatalf("resource ID = %q, want zone-123", got)
	}
	if got := resource.InstanceState.Attributes["mtls_certificate_id"]; got != "mtls-456" {
		t.Fatalf("mtls_certificate_id = %q, want mtls-456", got)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "zone-123/mtls-456" {
		t.Fatalf("import_id = %q, want zone-123/mtls-456", got)
	}
	if got := resource.InstanceState.Attributes["hostnames.#"]; got != "2" {
		t.Fatalf("hostnames.# = %q, want 2", got)
	}
	if _, ok := cloudflareCertificateAuthorityHostnameAssociationsResource(
		cf.Zone{ID: "zone-123", Name: "example.com"},
		"",
		nil,
	); ok {
		t.Fatal("expected empty hostname associations to be skipped")
	}
}

func TestAppendCertificateAuthorityHostnameAssociationsResourcesSkipsNonCAMTLSCertificates(t *testing.T) {
	var requestedMTLSIDs []string
	api := newCloudflareNetworkEdgeTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/zones/zone-123/certificate_authorities/hostname_associations" {
			t.Fatalf("path = %q, want /zones/zone-123/certificate_authorities/hostname_associations", r.URL.Path)
		}
		mtlsCertificateID := r.URL.Query().Get("mtls_certificate_id")
		requestedMTLSIDs = append(requestedMTLSIDs, mtlsCertificateID)
		switch mtlsCertificateID {
		case "":
			writeCloudflareNetworkEdgeTestResponse(t, w, map[string][]string{"hostnames": {"managed.example.com"}}, nil)
		case "ca-cert":
			writeCloudflareNetworkEdgeTestResponse(t, w, map[string][]string{"hostnames": {"ca.example.com"}}, nil)
		default:
			t.Fatalf("unexpected mtls_certificate_id query = %q", mtlsCertificateID)
		}
	}))

	var generator CertificatesGenerator
	err := generator.appendCertificateAuthorityHostnameAssociationsResources(
		context.Background(),
		api,
		cf.Zone{ID: "zone-123", Name: "example.com"},
		[]cf.MTLSCertificate{
			{ID: "leaf-cert", CA: false},
			{ID: "ca-cert", CA: true},
		},
	)
	if err != nil {
		t.Fatalf("appendCertificateAuthorityHostnameAssociationsResources() error = %v", err)
	}
	wantMTLSIDs := []string{"", "ca-cert"}
	if !reflect.DeepEqual(requestedMTLSIDs, wantMTLSIDs) {
		t.Fatalf("requested mtls_certificate_id values = %#v, want %#v", requestedMTLSIDs, wantMTLSIDs)
	}
	if len(generator.Resources) != 2 {
		t.Fatalf("resource count = %d, want 2", len(generator.Resources))
	}
}

func TestCloudflareCertificateImportPolicies(t *testing.T) {
	if !cloudflareCertificatePackImportable(cloudflareCertificateRawResource{
		"id":                    "pack-123",
		"certificate_authority": "lets_encrypt",
		"type":                  "advanced",
		"validation_method":     "txt",
		"validity_days":         float64(90),
		"status":                "active",
	}) {
		t.Fatal("active certificate pack with required fields should import")
	}
	if cloudflareCertificatePackImportable(cloudflareCertificateRawResource{
		"id":                    "pack-123",
		"certificate_authority": "lets_encrypt",
		"type":                  "advanced",
		"validation_method":     "txt",
		"validity_days":         float64(90),
		"status":                "pending_deletion",
	}) {
		t.Fatal("pending deletion certificate pack should be skipped")
	}
	if cloudflareCertificatePackImportable(cloudflareCertificateRawResource{
		"id":                    "pack-123",
		"certificate_authority": "lets_encrypt",
		"type":                  "universal",
		"validation_method":     "txt",
		"validity_days":         float64(90),
		"status":                "active",
	}) {
		t.Fatal("non-advanced certificate pack should be skipped")
	}
	if !cloudflareClientCertificateImportable(cloudflareCertificateRawResource{
		"id":            "client-123",
		"csr":           "-----BEGIN CERTIFICATE REQUEST-----",
		"validity_days": float64(365),
		"status":        "active",
	}) {
		t.Fatal("client certificate with public CSR and validity should import")
	}
	if cloudflareClientCertificateImportable(cloudflareCertificateRawResource{
		"id":            "client-123",
		"validity_days": float64(365),
		"status":        "active",
	}) {
		t.Fatal("client certificate without required CSR should be skipped")
	}
	if !cloudflareCustomOriginTrustStoreImportable(cloudflareCertificateRawResource{
		"id":          "trust-123",
		"certificate": "-----BEGIN CERTIFICATE-----",
		"status":      "active",
	}) {
		t.Fatal("custom origin trust store with certificate should import")
	}
	if cloudflareCustomOriginTrustStoreImportable(cloudflareCertificateRawResource{
		"id":     "trust-123",
		"status": "active",
	}) {
		t.Fatal("custom origin trust store without certificate should be skipped")
	}
	if !cloudflareOriginCACertificateImportable(cloudflareCertificateRawResource{
		"id":           "origin-123",
		"csr":          "-----BEGIN CERTIFICATE REQUEST-----",
		"request_type": "origin-rsa",
		"hostnames":    []interface{}{"example.com"},
	}) {
		t.Fatal("origin CA certificate with CSR, request type, and hostnames should import")
	}
	if cloudflareOriginCACertificateImportable(cloudflareCertificateRawResource{
		"id":           "origin-123",
		"csr":          "-----BEGIN CERTIFICATE REQUEST-----",
		"request_type": "origin-rsa",
		"hostnames":    []interface{}{"example.com"},
		"revoked_at":   "2026-05-17T00:00:00Z",
	}) {
		t.Fatal("revoked origin CA certificate should be skipped")
	}
}

func TestRunCloudflareCertificateDiscoveriesContinuesAfterOptionalError(t *testing.T) {
	resources := []cloudflareCertificateRawResource{}
	err := runCloudflareCertificateDiscoveries([]cloudflareCertificateDiscovery{
		{
			name:  "permission gated",
			scope: "zone-123",
			discover: func() error {
				requestErr := cf.NewRequestError(&cf.Error{ErrorMessages: []string{"missing permission"}})
				return &requestErr
			},
		},
		{
			name:  "succeeds",
			scope: "zone-123",
			discover: func() error {
				resources = append(resources, cloudflareCertificateRawResource{"id": "ok"})
				return nil
			},
		},
	})
	if err != nil {
		t.Fatalf("runCloudflareCertificateDiscoveries() error = %v, want nil", err)
	}
	if len(resources) != 1 {
		t.Fatalf("resource count = %d, want 1", len(resources))
	}

	err = runCloudflareCertificateDiscoveries([]cloudflareCertificateDiscovery{
		{
			name:  "fails",
			scope: "zone-123",
			discover: func() error {
				return errors.New("temporary Cloudflare failure")
			},
		},
	})
	if err == nil {
		t.Fatal("expected non-optional discovery error")
	}
}

func TestListCloudflareCertificateResourcesPaginates(t *testing.T) {
	api := newCloudflareNetworkEdgeTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/zones/zone-123/client_certificates" {
			t.Fatalf("path = %q, want /zones/zone-123/client_certificates", r.URL.Path)
		}
		switch r.URL.Query().Get("cursor") {
		case "":
			if got := r.URL.Query().Get("page"); got != "1" {
				t.Fatalf("page query = %q, want 1", got)
			}
			writeCloudflareNetworkEdgeTestResponse(t, w, []map[string]string{{"id": "cert-1"}}, map[string]interface{}{
				"cursors": map[string]string{"after": "cursor-2"},
			})
		case "cursor-2":
			writeCloudflareNetworkEdgeTestResponse(t, w, []map[string]string{{"id": "cert-2"}}, map[string]interface{}{
				"cursors": map[string]string{},
			})
		default:
			t.Fatalf("cursor query = %q, want empty or cursor-2", r.URL.Query().Get("cursor"))
		}
	}))

	resources, err := listCloudflareCertificateResources(context.Background(), api, "/zones/zone-123/client_certificates")
	if err != nil {
		t.Fatalf("listCloudflareCertificateResources() error = %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("resource count = %d, want 2", len(resources))
	}
	if got := cloudflareCertificateString(resources[0], "id"); got != "cert-1" {
		t.Fatalf("first id = %q, want cert-1", got)
	}
	if got := cloudflareCertificateString(resources[1], "id"); got != "cert-2" {
		t.Fatalf("second id = %q, want cert-2", got)
	}
}

func TestCloudflareCertificateUnsupportedResourcesMetadata(t *testing.T) {
	data, err := os.ReadFile("unsupported_resources.json")
	if err != nil {
		t.Fatalf("read unsupported resources: %v", err)
	}
	var metadata cloudflareUnsupportedResourcesFile
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("decode unsupported resources: %v", err)
	}
	statuses := map[string]string{}
	for _, resource := range metadata.Resources {
		statuses[resource.Resource] = resource.Status
	}
	for resource, wantStatus := range map[string]string{
		"cloudflare_authenticated_origin_pulls_certificate":          "secret-required",
		"cloudflare_authenticated_origin_pulls_hostname_certificate": "secret-required",
		"cloudflare_custom_ssl":                                      "secret-required",
		"cloudflare_hostname_tls_setting":                            "deferred",
		"cloudflare_keyless_certificate":                             "unsupported",
	} {
		if got := statuses[resource]; got != wantStatus {
			t.Fatalf("%s status = %q, want %q", resource, got, wantStatus)
		}
	}
}
