// SPDX-License-Identifier: Apache-2.0

package gmailfilter

import (
	"os"
	"strings"
	"testing"
)

func TestGmailfilterProviderInitReturnsCredentialEnvError(t *testing.T) {
	const probe = "REDACT_PROBE_GMAIL_CREDENTIALS"
	provider := GmailfilterProvider{
		credentials:           "old-credentials",
		impersonatedUserEmail: "old@example.com",
	}

	err := provider.Init([]string{probe + "\x00credentials"})
	if err == nil {
		t.Fatal("expected credentials env error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "failed to set env GOOGLE_CREDENTIALS") {
		t.Fatalf("Init error = %q, want GOOGLE_CREDENTIALS context", msg)
	}
	if strings.Contains(msg, probe) {
		t.Fatalf("Init error = %q, want credentials value redacted", msg)
	}
	if provider.credentials != "" {
		t.Fatalf("credentials = %q, want empty after failed init", provider.credentials)
	}
	if provider.impersonatedUserEmail != "" {
		t.Fatalf("impersonatedUserEmail = %q, want empty after failed init", provider.impersonatedUserEmail)
	}
}

func TestGmailfilterProviderInitReturnsImpersonatedUserEnvError(t *testing.T) {
	const probe = "REDACT_PROBE_GMAIL_USER"
	t.Setenv("GOOGLE_CREDENTIALS", "previous-credentials")
	provider := GmailfilterProvider{
		credentials:           "old-credentials",
		impersonatedUserEmail: "old@example.com",
	}

	err := provider.Init([]string{"credentials", probe + "\x00email"})
	if err == nil {
		t.Fatal("expected impersonated user env error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "failed to set env IMPERSONATED_USER_EMAIL") {
		t.Fatalf("Init error = %q, want IMPERSONATED_USER_EMAIL context", msg)
	}
	if strings.Contains(msg, probe) {
		t.Fatalf("Init error = %q, want email value redacted", msg)
	}
	if provider.credentials != "" {
		t.Fatalf("credentials = %q, want empty after failed init", provider.credentials)
	}
	if provider.impersonatedUserEmail != "" {
		t.Fatalf("impersonatedUserEmail = %q, want empty after failed init", provider.impersonatedUserEmail)
	}
	if got := os.Getenv("GOOGLE_CREDENTIALS"); got != "previous-credentials" {
		t.Fatalf("GOOGLE_CREDENTIALS = %q, want previous-credentials after failed init", got)
	}
}
