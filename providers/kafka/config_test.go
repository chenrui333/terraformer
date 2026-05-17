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
	handlerErrors := make(chan string, 8)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenRequests++
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		if r.Method != http.MethodPost {
			failTokenRequest(w, handlerErrors, "method = %s, want POST", r.Method)
			return
		}
		if err := r.ParseForm(); err != nil {
			failTokenRequest(w, handlerErrors, "ParseForm() error = %v", err)
			return
		}
		if got := r.PostForm.Get("grant_type"); got != "client_credentials" {
			failTokenRequest(w, handlerErrors, "grant_type = %q, want client_credentials", got)
			return
		}
		if got := r.PostForm.Get("scope"); got != "read write" {
			failTokenRequest(w, handlerErrors, "scope = %q, want read write", got)
			return
		}
		if user, pass, ok := r.BasicAuth(); ok {
			if user != "client-id" || pass != "client-secret" {
				failTokenRequest(w, handlerErrors, "basic auth = %q/%q, want client-id/client-secret", user, pass)
				return
			}
		} else if r.PostForm.Get("client_id") != "client-id" || r.PostForm.Get("client_secret") != "client-secret" {
			failTokenRequest(w, handlerErrors, "client credentials not sent in basic auth or form: %v", r.PostForm)
			return
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
		assertNoTokenRequestErrors(t, handlerErrors)
		t.Fatalf("Token() error = %v", err)
	}
	assertNoTokenRequestErrors(t, handlerErrors)
	if token.Token != "issued-token" {
		t.Fatalf("token = %q, want issued-token", token.Token)
	}
	if tokenRequests != 1 {
		t.Fatalf("token requests = %d, want 1", tokenRequests)
	}
}

func failTokenRequest(w http.ResponseWriter, errs chan<- string, format string, args ...interface{}) {
	select {
	case errs <- fmt.Sprintf(format, args...):
	default:
	}
	http.Error(w, "invalid token request", http.StatusBadRequest)
}

func assertNoTokenRequestErrors(t *testing.T, errs <-chan string) {
	t.Helper()
	select {
	case msg := <-errs:
		t.Fatalf("token request handler error: %s", msg)
	default:
	}
}

func TestOAuthBearerRejectsPremintedTokenOnly(t *testing.T) {
	t.Setenv("KAFKA_SASL_MECHANISM", "oauthbearer")
	t.Setenv("KAFKA_SASL_OAUTH_TOKEN", "preminted-token")
	t.Setenv("KAFKA_SASL_TOKEN_URL", "")
	t.Setenv("TOKEN_URL", "")

	config := ConfigFromEnv()
	_, err := config.newSaramaConfig()
	if err == nil {
		t.Fatal("expected missing oauthbearer token configuration error")
	}
	if !strings.Contains(err.Error(), "KAFKA_SASL_TOKEN_URL") {
		t.Fatalf("error = %q, want token URL requirement", err)
	}
}

func TestSASLCredentialsRequiredForPlainAndSCRAM(t *testing.T) {
	for _, mechanism := range []string{"plain", "scram-sha256", "scram-sha512"} {
		t.Run(mechanism, func(t *testing.T) {
			config := Config{
				KafkaVersion:     defaultKafkaVersion,
				Timeout:          defaultKafkaTimeout,
				SASLMechanism:    mechanism,
				SASLUsername:     "terraformer",
				BootstrapServers: []string{"broker1.example.com:9092"},
			}
			_, err := config.newSaramaConfig()
			if err == nil {
				t.Fatal("expected missing sasl password error")
			}
			if !strings.Contains(err.Error(), "sasl username and password are required") {
				t.Fatalf("error = %q, want sasl credential requirement", err)
			}
		})
	}
}

func TestTLSRequiresClientCertAndKeyTogether(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		clientCert string
		clientKey  string
	}{
		{name: "missing key", clientCert: "cert"},
		{name: "missing cert", clientKey: "key"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := newTLSConfig(testCase.clientCert, testCase.clientKey, "", "")
			if err == nil {
				t.Fatal("expected partial mTLS config error")
			}
			if !strings.Contains(err.Error(), "client certificate and client key must be provided together") {
				t.Fatalf("error = %q, want client certificate and key requirement", err)
			}
		})
	}
}
