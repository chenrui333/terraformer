// SPDX-License-Identifier: Apache-2.0

package terraformutils

import (
	"strings"
	"testing"
)

func TestSetEnvWrapsMutationError(t *testing.T) {
	err := SetEnv("BAD=KEY", "value")
	if err == nil {
		t.Fatal("expected SetEnv to fail")
	}
	msg := err.Error()
	if !strings.Contains(msg, "failed to set env BAD=KEY") {
		t.Fatalf("error = %q, want env key context", msg)
	}
	if strings.Contains(msg, "value") {
		t.Fatalf("error = %q, want env value redacted", msg)
	}
}
