// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"strings"
	"testing"
)

func TestAWSProviderInitRequiresArgs(t *testing.T) {
	var provider AWSProvider

	err := provider.Init([]string{MainRegionPublicPartition})
	if err == nil {
		t.Fatal("expected missing args error")
	}
	if !strings.Contains(err.Error(), "expected 2 init args") {
		t.Fatalf("Init error = %q, want missing AWS args", err)
	}
}
