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
	if msg := err.Error(); !strings.Contains(msg, "failed to set env BAD=KEY=\"value\"") {
		t.Fatalf("error = %q, want env key and value context", msg)
	}
}
