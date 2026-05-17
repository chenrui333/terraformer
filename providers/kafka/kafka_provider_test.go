// SPDX-License-Identifier: Apache-2.0

package kafka

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestProviderInitRequiresBootstrapServers(t *testing.T) {
	provider := &Provider{}

	err := provider.Init([]string{EncodeConfig(Config{KafkaVersion: defaultKafkaVersion, Timeout: defaultKafkaTimeout})})
	if err == nil {
		t.Fatal("expected missing bootstrap servers error")
	}
	if !strings.Contains(err.Error(), "bootstrap servers are required") {
		t.Fatalf("Init error = %q, want bootstrap servers requirement", err)
	}
}

func TestProviderSafeConfigHandling(t *testing.T) {
	t.Setenv("KAFKA_SASL_OAUTH_TOKEN", "oauth-token")
	t.Setenv("KAFKA_SASL_PASSWORD", "sasl-password")
	t.Setenv("KAFKA_CLIENT_KEY", "tls-private-key")
	t.Setenv("KAFKA_CLIENT_KEY_PASSPHRASE", "tls-private-key-passphrase")
	t.Setenv("AWS_ACCESS_KEY_ID", "aws-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "aws-secret-key")
	t.Setenv("AWS_SESSION_TOKEN", "aws-session-token")
	t.Setenv("AWS_CREDS_DEBUG", "true")

	config := Config{
		BootstrapServers: []string{"broker1.example.com:9092"},
		KafkaVersion:     defaultKafkaVersion,
		TLSEnabled:       true,
		SASLMechanism:    "plain",
		SASLUsername:     "terraformer",
		SASLPassword:     "sasl-password",
		ClientKey:        "tls-private-key",
		Timeout:          defaultKafkaTimeout,
	}

	encoded := EncodeConfig(config)
	for _, secret := range forbiddenTestSecrets() {
		if strings.Contains(encoded, secret) {
			t.Fatalf("encoded config contains secret %q: %s", secret, encoded)
		}
	}

	provider := &Provider{}
	if err := provider.Init([]string{encoded}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	rendered := fmt.Sprintf("%#v", provider.GetProviderData())
	for _, secret := range forbiddenTestSecrets() {
		if strings.Contains(rendered, secret) {
			t.Fatalf("provider data contains secret %q: %s", secret, rendered)
		}
	}
	if !strings.Contains(rendered, "broker1.example.com:9092") {
		t.Fatalf("provider data missing bootstrap server: %s", rendered)
	}

	refreshConfig := provider.GetConfig()
	for key, want := range map[string]string{
		"sasl_password":         "sasl-password",
		"client_key":            "tls-private-key",
		"client_key_passphrase": "tls-private-key-passphrase",
		"sasl_aws_access_key":   "aws-access-key",
		"sasl_aws_secret_key":   "aws-secret-key",
		"sasl_aws_token":        "aws-session-token",
	} {
		assertStringConfigAttr(t, refreshConfig, key, want)
	}
	if got := refreshConfig.GetAttr("sasl_aws_creds_debug"); !got.RawEquals(cty.BoolVal(true)) {
		t.Fatalf("sasl_aws_creds_debug = %#v, want true", got)
	}
}

func TestProviderSupportedServices(t *testing.T) {
	provider := &Provider{}
	services := provider.GetSupportedService()
	if _, ok := services["acls"]; !ok {
		t.Fatalf("acls service not registered: %#v", services)
	}
	if _, ok := services["topics"]; !ok {
		t.Fatalf("topics service not registered: %#v", services)
	}
	if err := provider.InitService("acls", false); err != nil {
		t.Fatalf("InitService(acls) error = %v", err)
	}
	if provider.GetService() == nil {
		t.Fatal("InitService(acls) did not set service")
	}
	if err := provider.InitService("topics", false); err != nil {
		t.Fatalf("InitService(topics) error = %v", err)
	}
	if provider.GetService() == nil {
		t.Fatal("InitService(topics) did not set service")
	}
}

func forbiddenTestSecrets() []string {
	return []string{
		"sasl-password",
		"tls-private-key",
		"tls-private-key-passphrase",
		"aws-access-key",
		"aws-secret-key",
		"aws-session-token",
		"oauth-token",
	}
}

func assertStringConfigAttr(t *testing.T, config cty.Value, key, want string) {
	t.Helper()
	got := config.GetAttr(key)
	if got.AsString() != want {
		t.Fatalf("%s = %q, want %q", key, got.AsString(), want)
	}
}
