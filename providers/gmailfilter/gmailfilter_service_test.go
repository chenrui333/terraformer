// SPDX-License-Identifier: Apache-2.0

package gmailfilter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fakePrivateKey = "-----BEGIN PRIVATE KEY-----fake-gmailfilter-secret-----END PRIVATE KEY-----"

func TestGetTokenSourceErrorsOmitCredentialContents(t *testing.T) {
	invalidJSON := `{"type":"service_account","private_key":"` + fakePrivateKey + `"`
	credsFile := filepath.Join(t.TempDir(), "creds.json")
	if err := os.WriteFile(credsFile, []byte(invalidJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	s := &GmailfilterService{}
	for name, creds := range map[string]string{
		"inline contents":  invalidJSON,
		"credentials file": credsFile,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := s.getTokenSource(creds, "user@example.com")
			if err == nil {
				t.Fatal("expected invalid credentials error")
			}
			if strings.Contains(err.Error(), "fake-gmailfilter-secret") {
				t.Fatalf("error leaks credential contents: %v", err)
			}
		})
	}
}
