// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"strings"
	"testing"
)

func TestAWSServiceBuildBaseConfigReturnsRegionEnvError(t *testing.T) {
	service := &AWSService{}
	service.SetArgs(map[string]interface{}{
		"profile": "",
		"region":  "bad\x00region",
	})

	_, err := service.buildBaseConfig()
	if err == nil {
		t.Fatal("expected region env error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "failed to set env AWS_REGION") {
		t.Fatalf("buildBaseConfig error = %q, want AWS_REGION context", msg)
	}
	if strings.Contains(msg, "bad") {
		t.Fatalf("buildBaseConfig error = %q, want env value redacted", msg)
	}
}
