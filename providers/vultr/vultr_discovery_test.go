// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr/v3"
)

func TestVultrProviderSupportedServicesUseV3Names(t *testing.T) {
	provider := &VultrProvider{}
	services := provider.GetSupportedService()

	wantServices := []string{
		"bare_metal_server",
		"block_storage",
		"dns_domain",
		"firewall_group",
		"instance",
		"reserved_ip",
		"snapshot",
		"ssh_key",
		"startup_script",
		"user",
		"vpc",
	}
	for _, service := range wantServices {
		if _, ok := services[service]; !ok {
			t.Fatalf("supported services missing %q; got %v", service, sortedVultrServiceNames(services))
		}
	}
	for _, service := range []string{"server", "network"} {
		if _, ok := services[service]; ok {
			t.Fatalf("legacy service %q should not be registered; got %v", service, sortedVultrServiceNames(services))
		}
	}
	if got := provider.GetName(); got != "vultr" {
		t.Fatalf("GetName() = %q, want vultr", got)
	}

	provider.apiKey = "test-token"
	if err := provider.InitService("instance", false); err != nil {
		t.Fatalf("InitService(instance) returned error: %v", err)
	}
	if got := provider.Service.GetArgs()["api_key"]; got != "test-token" {
		t.Fatalf("InitService(instance) api_key = %v, want test-token", got)
	}
	if err := provider.InitService("server", false); err == nil {
		t.Fatal("InitService(server) returned nil error, want unsupported service error")
	}
}

func TestVultrServiceGenerateClientValidatesAPIKey(t *testing.T) {
	tests := []struct {
		name string
		args map[string]interface{}
	}{
		{name: "missing", args: nil},
		{name: "empty", args: map[string]interface{}{"api_key": ""}},
		{name: "non string", args: map[string]interface{}{"api_key": 123}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &VultrService{}
			service.SetArgs(tt.args)

			_, err := service.generateClient()
			if err == nil {
				t.Fatal("generateClient() returned nil error, want validation error")
			}
			if !strings.Contains(err.Error(), "api_key") {
				t.Fatalf("generateClient() error = %q, want api_key context", err)
			}
		})
	}
}

func TestVultrServiceGenerateClientUsesBearerToken(t *testing.T) {
	var seen atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.Add(1)
		if r.URL.Path != "/v2/instances" {
			t.Errorf("request path = %q, want /v2/instances", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization header = %q, want Bearer test-token", got)
			http.Error(w, "unexpected authorization", http.StatusUnauthorized)
			return
		}
		writeVultrJSON(w, http.StatusOK, map[string]interface{}{
			"instances": []map[string]interface{}{},
			"meta":      emptyVultrMeta(),
		})
	}))
	defer server.Close()

	service := &VultrService{}
	service.SetArgs(map[string]interface{}{"api_key": "test-token"})
	client, err := service.generateClient()
	if err != nil {
		t.Fatalf("generateClient() returned error: %v", err)
	}
	configureTestVultrClient(t, client, server.URL)

	_, _, resp, err := client.Instance.List(context.Background(), &govultr.ListOptions{PerPage: 100})
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("Instance.List returned error: %v", err)
	}
	if got := seen.Load(); got != 1 {
		t.Fatalf("request count = %d, want 1", got)
	}
}

func TestServerGeneratorInitResourcesPaginatesInstances(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/instances" {
			t.Errorf("request path = %q, want /v2/instances", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		switch cursor := r.URL.Query().Get("cursor"); cursor {
		case "":
			if !assertVultrListQuery(t, w, r, "") {
				return
			}
			writeVultrJSON(w, http.StatusOK, map[string]interface{}{
				"instances": []map[string]interface{}{{"id": "instance-1"}},
				"meta":      vultrMeta("cursor-2"),
			})
		case "cursor-2":
			if !assertVultrListQuery(t, w, r, "cursor-2") {
				return
			}
			writeVultrJSON(w, http.StatusOK, map[string]interface{}{
				"instances": []map[string]interface{}{{"id": "instance-2"}},
				"meta":      emptyVultrMeta(),
			})
		default:
			t.Errorf("cursor = %q, want empty or cursor-2", cursor)
			http.Error(w, "unexpected cursor", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	generator := &ServerGenerator{}
	if err := generator.initResources(newTestVultrClient(t, server)); err != nil {
		t.Fatalf("initResources returned error: %v", err)
	}

	if got := len(generator.Resources); got != 2 {
		t.Fatalf("resource count = %d, want 2", got)
	}
	assertVultrResource(t, generator.Resources[0], "instance-1", "instance-1", "vultr_instance")
	assertVultrResource(t, generator.Resources[1], "instance-2", "instance-2", "vultr_instance")
}

func TestServerGeneratorInitResourcesHandlesEmptyPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/instances" {
			t.Errorf("request path = %q, want /v2/instances", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		if !assertVultrListQuery(t, w, r, "") {
			return
		}
		writeVultrJSON(w, http.StatusOK, map[string]interface{}{
			"instances": []map[string]interface{}{},
			"meta":      emptyVultrMeta(),
		})
	}))
	defer server.Close()

	generator := &ServerGenerator{}
	if err := generator.initResources(newTestVultrClient(t, server)); err != nil {
		t.Fatalf("initResources returned error: %v", err)
	}
	if got := len(generator.Resources); got != 0 {
		t.Fatalf("resource count = %d, want 0", got)
	}
}

func TestServerGeneratorInitResourcesReturnsSecondPageError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/instances" {
			t.Errorf("request path = %q, want /v2/instances", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		switch cursor := r.URL.Query().Get("cursor"); cursor {
		case "":
			if !assertVultrListQuery(t, w, r, "") {
				return
			}
			writeVultrJSON(w, http.StatusOK, map[string]interface{}{
				"instances": []map[string]interface{}{{"id": "instance-1"}},
				"meta":      vultrMeta("cursor-2"),
			})
		case "cursor-2":
			if !assertVultrListQuery(t, w, r, "cursor-2") {
				return
			}
			writeVultrJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "backend unavailable"})
		default:
			t.Errorf("cursor = %q, want empty or cursor-2", cursor)
			http.Error(w, "unexpected cursor", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	generator := &ServerGenerator{}
	err := generator.initResources(newTestVultrClient(t, server))
	if err == nil {
		t.Fatal("initResources returned nil error, want second page error")
	}
	if !strings.Contains(err.Error(), "list vultr instances") {
		t.Fatalf("initResources error = %q, want list context", err)
	}
}

func TestVultrResourceMappingUsesV3TerraformTypes(t *testing.T) {
	instances := ServerGenerator{}.createResources([]govultr.Instance{{ID: "instance-id"}})
	if got := len(instances); got != 1 {
		t.Fatalf("instance resource count = %d, want 1", got)
	}
	assertVultrResource(t, instances[0], "instance-id", "instance-id", "vultr_instance")

	vpcs := NetworkGenerator{}.createResources([]govultr.VPC{{ID: "vpc-id"}})
	if got := len(vpcs); got != 1 {
		t.Fatalf("VPC resource count = %d, want 1", got)
	}
	assertVultrResource(t, vpcs[0], "vpc-id", "vpc-id", "vultr_vpc")
}

func TestDNSDomainGeneratorMapsDomainsAndRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/domains":
			if !assertVultrListQuery(t, w, r, "") {
				return
			}
			writeVultrJSON(w, http.StatusOK, map[string]interface{}{
				"domains": []map[string]interface{}{{"domain": "example.com"}},
				"meta":    emptyVultrMeta(),
			})
		case "/v2/domains/example.com/records":
			if !assertVultrListQuery(t, w, r, "") {
				return
			}
			writeVultrJSON(w, http.StatusOK, map[string]interface{}{
				"records": []map[string]interface{}{{"id": "record-1", "type": "A", "name": "www", "data": "192.0.2.1"}},
				"meta":    emptyVultrMeta(),
			})
		default:
			t.Errorf("request path = %q, want DNS domain or records path", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	}))
	defer server.Close()

	generator := &DNSDomainGenerator{}
	client := newTestVultrClient(t, server)
	domains, err := generator.loadDNSDomains(client)
	if err != nil {
		t.Fatalf("loadDNSDomains returned error: %v", err)
	}
	if got := len(domains); got != 1 {
		t.Fatalf("domain count = %d, want 1", got)
	}
	if err := generator.loadDNSRecords(client, "example.com"); err != nil {
		t.Fatalf("loadDNSRecords returned error: %v", err)
	}

	if got := len(generator.Resources); got != 2 {
		t.Fatalf("resource count = %d, want 2", got)
	}
	assertVultrResource(t, generator.Resources[0], "example.com", "example.com", "vultr_dns_domain")
	assertVultrResource(t, generator.Resources[1], "record-1", "record-1", "vultr_dns_record")
	if got := generator.Resources[1].InstanceState.Attributes["domain"]; got != "example.com" {
		t.Fatalf("DNS record domain attribute = %q, want example.com", got)
	}
}

func sortedVultrServiceNames(services map[string]terraformutils.ServiceGenerator) []string {
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func newTestVultrClient(t *testing.T, server *httptest.Server) *govultr.Client {
	t.Helper()
	client := govultr.NewClient(server.Client())
	configureTestVultrClient(t, client, server.URL)
	return client
}

func configureTestVultrClient(t *testing.T, client *govultr.Client, baseURL string) {
	t.Helper()
	client.SetRetryLimit(0)
	client.SetRateLimit(0)
	if err := client.SetBaseURL(baseURL); err != nil {
		t.Fatalf("SetBaseURL(%q) returned error: %v", baseURL, err)
	}
}

func assertVultrListQuery(t *testing.T, w http.ResponseWriter, r *http.Request, wantCursor string) bool {
	if r.Method != http.MethodGet {
		t.Errorf("request method = %q, want GET", r.Method)
		http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		return false
	}
	if got := r.URL.Query().Get("per_page"); got != "100" {
		t.Errorf("per_page query = %q, want 100", got)
		http.Error(w, "unexpected per_page", http.StatusBadRequest)
		return false
	}
	if got := r.URL.Query().Get("cursor"); got != wantCursor {
		t.Errorf("cursor query = %q, want %q", got, wantCursor)
		http.Error(w, "unexpected cursor", http.StatusBadRequest)
		return false
	}
	return true
}

func assertVultrResource(t *testing.T, resource terraformutils.Resource, wantID, wantName, wantType string) {
	t.Helper()
	if got := resource.InstanceState.ID; got != wantID {
		t.Fatalf("resource ID = %q, want %q", got, wantID)
	}
	if got := resource.ResourceName; got != terraformutils.TfSanitize(wantName) {
		t.Fatalf("resource name = %q, want sanitized %q", got, wantName)
	}
	if got := resource.InstanceInfo.Type; got != wantType {
		t.Fatalf("resource type = %q, want %q", got, wantType)
	}
}

func writeVultrJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func vultrMeta(next string) map[string]interface{} {
	return map[string]interface{}{
		"total": 1,
		"links": map[string]string{
			"next": next,
			"prev": "",
		},
	}
}

func emptyVultrMeta() map[string]interface{} {
	return vultrMeta("")
}
