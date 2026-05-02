// SPDX-License-Identifier: Apache-2.0

package vault

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	vaultapi "github.com/hashicorp/vault/api"
)

func TestCreateSecretBackendRoleResourcesReturnsListError(t *testing.T) {
	generator := newVaultTestGenerator(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/sys/mounts":
			writeVaultMounts(t, w, "aws/", "aws")
		default:
			http.Error(w, "list failed", http.StatusServiceUnavailable)
		}
	})
	generator.mountType = "aws"

	err := generator.createSecretBackendRoleResources()
	if err == nil {
		t.Fatal("expected secret backend role list error")
	}
	if !strings.Contains(err.Error(), "list vault secret backend roles at aws/roles") {
		t.Fatalf("error = %q, want wrapped secret backend role list error", err)
	}
}

func TestCreateAuthBackendEntityResourcesReturnsListError(t *testing.T) {
	generator := newVaultTestGenerator(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/sys/auth":
			writeVaultMounts(t, w, "approle/", "approle")
		default:
			http.Error(w, "list failed", http.StatusServiceUnavailable)
		}
	})
	generator.mountType = "approle"

	err := generator.createAuthBackendEntityResources("role", "role")
	if err == nil {
		t.Fatal("expected auth backend entity list error")
	}
	if !strings.Contains(err.Error(), "list vault auth backend role at /auth/approle/role") {
		t.Fatalf("error = %q, want wrapped auth backend entity list error", err)
	}
}

func TestCreateGenericSecretResourcesReturnsListError(t *testing.T) {
	generator := newVaultTestGenerator(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/sys/mounts":
			writeVaultMounts(t, w, "secret/", "kv")
		default:
			http.Error(w, "list failed", http.StatusServiceUnavailable)
		}
	})
	generator.mountType = "kv"

	err := generator.createGenericSecretResources()
	if err == nil {
		t.Fatal("expected generic secret list error")
	}
	if !strings.Contains(err.Error(), "list vault generic secrets at secret/") {
		t.Fatalf("error = %q, want wrapped generic secret list error", err)
	}
}

func newVaultTestGenerator(t *testing.T, handler http.HandlerFunc) *ServiceGenerator {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := vaultapi.NewClient(&vaultapi.Config{Address: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	return &ServiceGenerator{client: client}
}

func writeVaultMounts(t *testing.T, w http.ResponseWriter, mountPath, mountType string) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			mountPath: map[string]interface{}{
				"type": mountType,
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
}
