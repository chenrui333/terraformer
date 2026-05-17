// SPDX-License-Identifier: Apache-2.0

package kafka

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IBM/sarama"
)

func TestOAuthBearerUsesTokenURLProvider(t *testing.T) {
	var tokenRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenRequests++
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if got := r.PostForm.Get("grant_type"); got != "client_credentials" {
			t.Fatalf("grant_type = %q, want client_credentials", got)
		}
		if got := r.PostForm.Get("scope"); got != "read write" {
			t.Fatalf("scope = %q, want read write", got)
		}
		if user, pass, ok := r.BasicAuth(); ok {
			if user != "client-id" || pass != "client-secret" {
				t.Fatalf("basic auth = %q/%q, want client-id/client-secret", user, pass)
			}
		} else if r.PostForm.Get("client_id") != "client-id" || r.PostForm.Get("client_secret") != "client-secret" {
			t.Fatalf("client credentials not sent in basic auth or form: %v", r.PostForm)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{\"access_token\":\"issued-token\",\"token_type\":\"bearer\",\"expires_in\":3600}")
	}))
	defer server.Close()

	config := Config{
		KafkaVersion:     defaultKafkaVersion,
		Timeout:          defaultKafkaTimeout,
		SASLMechanism:    "oauthbearer",
		SASLUsername:     "client-id",
		SASLPassword:     "client-secret",
		SASLTokenURL:     server.URL,
		SASLOAuthScopes:  []string{"read", "write"},
		BootstrapServers: []string{"broker1.example.com:9092"},
	}
	saramaConfig, err := config.newSaramaConfig()
	if err != nil {
		t.Fatalf("newSaramaConfig() error = %v", err)
	}
	if saramaConfig.Net.SASL.Mechanism != sarama.SASLTypeOAuth {
		t.Fatalf("SASL mechanism = %q, want %q", saramaConfig.Net.SASL.Mechanism, sarama.SASLTypeOAuth)
	}
	token, err := saramaConfig.Net.SASL.TokenProvider.Token()
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if token.Token != "issued-token" {
		t.Fatalf("token = %q, want issued-token", token.Token)
	}
	if tokenRequests != 1 {
		t.Fatalf("token requests = %d, want 1", tokenRequests)
	}
}

func TestOAuthBearerStillAcceptsPremintedToken(t *testing.T) {
	config := Config{
		KafkaVersion:         defaultKafkaVersion,
		Timeout:              defaultKafkaTimeout,
		SASLMechanism:        "oauthbearer",
		SASLOAuthBearerToken: "preminted-token",
		BootstrapServers:     []string{"broker1.example.com:9092"},
	}
	saramaConfig, err := config.newSaramaConfig()
	if err != nil {
		t.Fatalf("newSaramaConfig() error = %v", err)
	}
	token, err := saramaConfig.Net.SASL.TokenProvider.Token()
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if token.Token != "preminted-token" {
		t.Fatalf("token = %q, want preminted-token", token.Token)
	}
}

func TestOAuthBearerRequiresTokenURLOrPremintedToken(t *testing.T) {
	config := Config{
		KafkaVersion:     defaultKafkaVersion,
		Timeout:          defaultKafkaTimeout,
		SASLMechanism:    "oauthbearer",
		BootstrapServers: []string{"broker1.example.com:9092"},
	}
	_, err := config.newSaramaConfig()
	if err == nil {
		t.Fatal("expected missing oauthbearer token configuration error")
	}
	if !strings.Contains(err.Error(), "KAFKA_SASL_OAUTH_TOKEN or KAFKA_SASL_TOKEN_URL") {
		t.Fatalf("error = %q, want token URL or preminted token requirement", err)
	}
}
