// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateRequestReturnsHTTPStatusErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not authorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	service := &RBTService{}
	service.SetArgs(map[string]interface{}{
		"endpoint": server.URL,
		"username": "guest",
		"password": "guest",
	})

	_, err := service.generateRequest("/api/queues")
	if err == nil {
		t.Fatal("expected HTTP error")
	}
	if !strings.Contains(err.Error(), "401 Unauthorized") {
		t.Fatalf("expected status in error, got %q", err)
	}
	if !strings.Contains(err.Error(), "not authorized") {
		t.Fatalf("expected response body in error, got %q", err)
	}
}

func TestGenerateRequestReturnsBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "guest" || password != "guest" {
			t.Errorf("expected basic auth credentials")
		}
		if r.URL.Path != "/api/queues" {
			t.Errorf("expected request path /api/queues, got %q", r.URL.Path)
		}
		_, _ = w.Write([]byte("{\"ok\":true}"))
	}))
	defer server.Close()

	service := &RBTService{}
	service.SetArgs(map[string]interface{}{
		"endpoint": server.URL,
		"username": "guest",
		"password": "guest",
	})

	body, err := service.generateRequest("/api/queues")
	if err != nil {
		t.Fatalf("expected request to succeed: %v", err)
	}
	if string(body) != "{\"ok\":true}" {
		t.Fatalf("expected response body, got %q", body)
	}
}
