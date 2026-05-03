// SPDX-License-Identifier: Apache-2.0

package gmailfilter

import (
	"strings"
	"testing"
)

func TestGmailfilterProviderInitReturnsCredentialEnvError(t *testing.T) {
	const probe = "REDACT_PROBE_GMAIL_CREDENTIALS"
	var provider GmailfilterProvider

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
}

func TestGmailfilterProviderInitReturnsImpersonatedUserEnvError(t *testing.T) {
	const probe = "REDACT_PROBE_GMAIL_USER"
	var provider GmailfilterProvider

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
}
