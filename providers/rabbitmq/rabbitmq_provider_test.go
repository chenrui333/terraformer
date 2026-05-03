// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"strings"
	"testing"
)

func TestRBTProviderInitRequiresArgs(t *testing.T) {
	provider := RBTProvider{
		endpoint: "https://old.example.com",
		username: "old-user",
		password: "old-password",
	}

	err := provider.Init([]string{"https://rabbitmq.example.com", "guest"})
	if err == nil {
		t.Fatal("expected missing args error")
	}
	if !strings.Contains(err.Error(), "endpoint, username, and password are required") {
		t.Fatalf("Init error = %q, want missing RabbitMQ args", err)
	}
	if provider.endpoint != "" {
		t.Fatalf("endpoint = %q, want empty after failed init", provider.endpoint)
	}
	if provider.username != "" {
		t.Fatalf("username = %q, want empty after failed init", provider.username)
	}
	if provider.password != "" {
		t.Fatalf("password = %q, want empty after failed init", provider.password)
	}
}

func TestRBTProviderInitStoresArgs(t *testing.T) {
	var provider RBTProvider

	if err := provider.Init([]string{"https://rabbitmq.example.com", "guest", "secret"}); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.endpoint != "https://rabbitmq.example.com" {
		t.Fatalf("endpoint = %q, want https://rabbitmq.example.com", provider.endpoint)
	}
	if provider.username != "guest" {
		t.Fatalf("username = %q, want guest", provider.username)
	}
	if provider.password != "secret" {
		t.Fatalf("password = %q, want secret", provider.password)
	}
}
