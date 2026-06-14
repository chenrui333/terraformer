// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	managementclient "github.com/auth0/go-auth0/v2/management/client"
	managementoption "github.com/auth0/go-auth0/v2/management/option"
	"github.com/chenrui333/terraformer/terraformutils"
)

type auth0ResourceExpectation struct {
	id       string
	name     string
	typeName string
}

func TestAuth0ProviderSupportedServicesUseCurrentResourceNames(t *testing.T) {
	services := (&Auth0Provider{}).GetSupportedService()
	wantServices := []string{
		"auth0_action",
		"auth0_branding",
		"auth0_client",
		"auth0_client_grant",
		"auth0_custom_domain",
		"auth0_email_provider",
		"auth0_hook",
		"auth0_log_stream",
		"auth0_prompt",
		"auth0_resource_server",
		"auth0_role",
		"auth0_rule",
		"auth0_rule_config",
		"auth0_tenant",
		"auth0_trigger_actions",
		"auth0_user",
	}

	for _, service := range wantServices {
		if _, ok := services[service]; !ok {
			t.Fatalf("supported services missing %s; got %v", service, sortedAuth0ServiceNames(services))
		}
	}

	for _, staleService := range []string{"auth0_email", "auth0_trigger_binding"} {
		if _, ok := services[staleService]; ok {
			t.Fatalf("supported services still includes stale service %s", staleService)
		}
	}

	docs, err := os.ReadFile("../../docs/auth0.md")
	if err != nil {
		t.Fatalf("read auth0 docs: %v", err)
	}
	docText := string(docs)
	for _, service := range wantServices {
		quotedService := string(rune(0x60)) + service + string(rune(0x60))
		if !strings.Contains(docText, quotedService) {
			t.Fatalf("docs/auth0.md missing %s", quotedService)
		}
	}
	for _, staleService := range []string{"auth0_email", "auth0_trigger_binding"} {
		quotedService := string(rune(0x60)) + staleService + string(rune(0x60))
		if strings.Contains(docText, quotedService) {
			t.Fatalf("docs/auth0.md still documents stale service %s", quotedService)
		}
	}
}

func TestAuth0ServiceGenerateClientValidatesArgs(t *testing.T) {
	tests := []struct {
		name string
		args map[string]interface{}
	}{
		{
			name: "missing args",
			args: nil,
		},
		{
			name: "empty domain",
			args: map[string]interface{}{"domain": "", "client_id": "client", "client_secret": "secret"},
		},
		{
			name: "non-string client id",
			args: map[string]interface{}{"domain": "tenant.auth0.com", "client_id": 123, "client_secret": "secret"},
		},
		{
			name: "empty client secret",
			args: map[string]interface{}{"domain": "tenant.auth0.com", "client_id": "client", "client_secret": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := Auth0Service{}
			service.SetArgs(tt.args)

			client, err := service.generateClient()
			if err == nil {
				t.Fatalf("generateClient() error = nil, client = %#v", client)
			}
			if !strings.Contains(err.Error(), "auth0:") {
				t.Fatalf("generateClient() error = %q, want auth0 context", err)
			}
		})
	}
}

func TestClientGeneratorInitResourcesPaginates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, ok := assertAuth0ListRequest(t, r, "/api/v2/clients")
		if !ok {
			writeAuth0JSON(t, w, http.StatusBadRequest, map[string]interface{}{"message": "bad request"})
			return
		}

		switch page {
		case 0:
			writeAuth0JSON(t, w, http.StatusOK, map[string]interface{}{
				"clients": []map[string]interface{}{{"client_id": "client-1", "name": "Primary"}},
				"start":   0,
				"limit":   50,
				"total":   2,
			})
		case 1:
			writeAuth0JSON(t, w, http.StatusOK, map[string]interface{}{
				"clients": []map[string]interface{}{{"client_id": "client-2", "name": "Secondary"}},
				"start":   50,
				"limit":   50,
				"total":   2,
			})
		default:
			writeAuth0JSON(t, w, http.StatusOK, map[string]interface{}{
				"clients": []map[string]interface{}{},
				"start":   page * 50,
				"limit":   50,
				"total":   2,
			})
		}
	}))
	t.Cleanup(server.Close)

	generator := ClientGenerator{}
	generator.SetArgs(map[string]interface{}{managementClientArg: newTestAuth0ManagementClient(t, server)})

	err := generator.InitResources()
	if err != nil {
		t.Fatalf("InitResources() error = %v", err)
	}
	resources := generator.Resources

	assertAuth0Resources(t, resources, []auth0ResourceExpectation{
		{id: "client-1", name: "tfer--client-1_Primary", typeName: "auth0_client"},
		{id: "client-2", name: "tfer--client-2_Secondary", typeName: "auth0_client"},
	})
}

func TestResourceServerGeneratorInitResourcesPaginates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, ok := assertAuth0ListRequest(t, r, "/api/v2/resource-servers")
		if !ok {
			writeAuth0JSON(t, w, http.StatusBadRequest, map[string]interface{}{"message": "bad request"})
			return
		}

		switch page {
		case 0:
			writeAuth0JSON(t, w, http.StatusOK, map[string]interface{}{
				"resource_servers": []map[string]interface{}{{"id": "api-1", "name": "API One", "identifier": "https://api-one.example.com"}},
				"start":            0,
				"limit":            50,
				"total":            2,
			})
		case 1:
			writeAuth0JSON(t, w, http.StatusOK, map[string]interface{}{
				"resource_servers": []map[string]interface{}{{"id": "api-2", "name": "API Two", "identifier": "https://api-two.example.com"}},
				"start":            50,
				"limit":            50,
				"total":            2,
			})
		default:
			writeAuth0JSON(t, w, http.StatusOK, map[string]interface{}{
				"resource_servers": []map[string]interface{}{},
				"start":            page * 50,
				"limit":            50,
				"total":            2,
			})
		}
	}))
	t.Cleanup(server.Close)

	generator := ResourceServerGenerator{}
	generator.SetArgs(map[string]interface{}{managementClientArg: newTestAuth0ManagementClient(t, server)})

	err := generator.InitResources()
	if err != nil {
		t.Fatalf("InitResources() error = %v", err)
	}
	resources := generator.Resources

	assertAuth0Resources(t, resources, []auth0ResourceExpectation{
		{id: "api-1", name: "tfer--api-1_API-0020-One", typeName: "auth0_resource_server"},
		{id: "api-2", name: "tfer--api-2_API-0020-Two", typeName: "auth0_resource_server"},
	})
}

func TestRoleGeneratorInitResourcesHandlesEmptyFirstPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := assertAuth0ListRequest(t, r, "/api/v2/roles")
		if !ok {
			writeAuth0JSON(t, w, http.StatusBadRequest, map[string]interface{}{"message": "bad request"})
			return
		}

		writeAuth0JSON(t, w, http.StatusOK, map[string]interface{}{
			"roles": []map[string]interface{}{},
			"start": 0,
			"limit": 50,
			"total": 0,
		})
	}))
	t.Cleanup(server.Close)

	generator := RoleGenerator{}
	generator.SetArgs(map[string]interface{}{managementClientArg: newTestAuth0ManagementClient(t, server)})

	err := generator.InitResources()
	if err != nil {
		t.Fatalf("InitResources() error = %v", err)
	}
	resources := generator.Resources
	if len(resources) != 0 {
		t.Fatalf("InitResources() resources length = %d, want 0", len(resources))
	}
}

func TestRoleGeneratorInitResourcesReturnsSecondPageError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, ok := assertAuth0ListRequest(t, r, "/api/v2/roles")
		if !ok {
			writeAuth0JSON(t, w, http.StatusBadRequest, map[string]interface{}{"message": "bad request"})
			return
		}

		switch page {
		case 0:
			writeAuth0JSON(t, w, http.StatusOK, map[string]interface{}{
				"roles": []map[string]interface{}{{"id": "role-1", "name": "Admins"}},
				"start": 0,
				"limit": 50,
				"total": 2,
			})
		case 1:
			writeAuth0JSON(t, w, http.StatusInternalServerError, map[string]interface{}{
				"error":      "server_error",
				"message":    "roles page failed",
				"statusCode": 500,
			})
		default:
			writeAuth0JSON(t, w, http.StatusOK, map[string]interface{}{
				"roles": []map[string]interface{}{},
				"start": page * 50,
				"limit": 50,
				"total": 2,
			})
		}
	}))
	t.Cleanup(server.Close)

	generator := RoleGenerator{}
	generator.SetArgs(map[string]interface{}{managementClientArg: newTestAuth0ManagementClient(t, server)})

	err := generator.InitResources()
	if err == nil {
		t.Fatal("InitResources() error = nil, want second page error")
	}
	if !strings.Contains(err.Error(), "roles page failed") {
		t.Fatalf("InitResources() error = %q, want roles page failed", err)
	}
}

func newTestAuth0ManagementClient(t *testing.T, server *httptest.Server) *managementclient.Management {
	t.Helper()

	client := managementclient.NewWithOptions(
		managementoption.WithBaseURL(server.URL+"/api/v2"),
		managementoption.WithHTTPClient(server.Client()),
		managementoption.WithToken("test-token"),
		managementoption.WithoutRetries(),
	)
	return client
}

func sortedAuth0ServiceNames(services map[string]terraformutils.ServiceGenerator) []string {
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func assertAuth0ListRequest(t *testing.T, r *http.Request, path string) (int, bool) {
	t.Helper()

	ok := true
	if r.Method != http.MethodGet {
		t.Errorf("request method = %s, want GET", r.Method)
		ok = false
	}
	if r.URL.Path != path {
		t.Errorf("request path = %s, want %s", r.URL.Path, path)
		ok = false
	}
	if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
		t.Errorf("authorization header = %q, want bearer test token", got)
		ok = false
	}
	if got := r.URL.Query().Get("per_page"); got != "50" {
		t.Errorf("per_page query = %q, want 50", got)
		ok = false
	}
	if got := r.URL.Query().Get("include_totals"); got != "true" {
		t.Errorf("include_totals query = %q, want true", got)
		ok = false
	}

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		t.Errorf("page query = %q, want integer: %v", r.URL.Query().Get("page"), err)
		ok = false
	}
	return page, ok
}

func assertAuth0Resources(t *testing.T, resources []terraformutils.Resource, want []auth0ResourceExpectation) {
	t.Helper()

	if len(resources) != len(want) {
		t.Fatalf("resources length = %d, want %d: %#v", len(resources), len(want), resources)
	}
	for i := range want {
		got := resources[i]
		if got.ResourceName != want[i].name {
			t.Fatalf("resources[%d].ResourceName = %q, want %q", i, got.ResourceName, want[i].name)
		}
		if got.InstanceState.ID != want[i].id {
			t.Fatalf("resources[%d].InstanceState.ID = %q, want %q", i, got.InstanceState.ID, want[i].id)
		}
		if got.InstanceInfo.Type != want[i].typeName {
			t.Fatalf("resources[%d].InstanceInfo.Type = %q, want %q", i, got.InstanceInfo.Type, want[i].typeName)
		}
	}
}

func writeAuth0JSON(t *testing.T, w http.ResponseWriter, status int, payload interface{}) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Errorf("write JSON response: %v", err)
	}
}
