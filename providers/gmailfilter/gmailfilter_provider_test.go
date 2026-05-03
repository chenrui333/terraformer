// SPDX-License-Identifier: Apache-2.0

package gmailfilter

import (
	"strings"
	"testing"
)

func TestGmailfilterProviderInitReturnsCredentialEnvError(t *testing.T) {
	var provider GmailfilterProvider

	err := provider.Init([]string{"bad\x00credentials"})
	if err == nil {
		t.Fatal("expected credentials env error")
	}
	if msg := err.Error(); !strings.Contains(msg, `failed to set env GOOGLE_CREDENTIALS="bad\x00credentials"`) {
		t.Fatalf("Init error = %q, want GOOGLE_CREDENTIALS context", msg)
	}
}

func TestGmailfilterProviderInitReturnsImpersonatedUserEnvError(t *testing.T) {
	var provider GmailfilterProvider

	err := provider.Init([]string{"credentials", "bad\x00email"})
	if err == nil {
		t.Fatal("expected impersonated user env error")
	}
	if msg := err.Error(); !strings.Contains(msg, `failed to set env IMPERSONATED_USER_EMAIL="bad\x00email"`) {
		t.Fatalf("Init error = %q, want IMPERSONATED_USER_EMAIL context", msg)
	}
}
