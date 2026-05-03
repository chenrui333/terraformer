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
	if msg := err.Error(); !strings.Contains(msg, `failed to set env AWS_REGION="bad\x00region"`) {
		t.Fatalf("buildBaseConfig error = %q, want AWS_REGION context", msg)
	}
}
