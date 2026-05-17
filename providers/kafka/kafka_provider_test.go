// SPDX-License-Identifier: Apache-2.0

package kafka

import (
	"fmt"
	"strings"
	"testing"
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

	config := Config{
		BootstrapServers:    []string{"broker1.example.com:9092"},
		KafkaVersion:        defaultKafkaVersion,
		TLSEnabled:          true,
		SASLMechanism:       "plain",
		SASLUsername:        "terraformer",
		SASLPassword:        "sasl-password",
		ClientKey:           "tls-private-key",
		SASLAWSAccessKey:    "aws-access-key",
		SASLAWSSecretKey:    "aws-secret-key",
		SASLAWSSessionToken: "aws-session-token",
		Timeout:             defaultKafkaTimeout,
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
}

func TestProviderSupportedServices(t *testing.T) {
	provider := &Provider{}
	services := provider.GetSupportedService()
	if _, ok := services["topics"]; !ok {
		t.Fatalf("topics service not registered: %#v", services)
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
		"aws-access-key",
		"aws-secret-key",
		"aws-session-token",
		"oauth-token",
	}
}
