// SPDX-License-Identifier: Apache-2.0

package tencentcloud

import (
	"strings"
	"testing"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

func TestTencentCloudProviderInitRequiresRegion(t *testing.T) {
	provider := TencentCloudProvider{
		region: "old-region",
		credential: common.Credential{
			SecretId: "old-secret-id",
		},
	}

	err := provider.Init(nil)
	if err == nil {
		t.Fatal("expected missing region error")
	}
	if !strings.Contains(err.Error(), "expected 1 init arg") {
		t.Fatalf("Init error = %q, want missing region", err)
	}
	if provider.region != "" {
		t.Fatalf("region = %q, want empty after failed init", provider.region)
	}
	if provider.credential.SecretId != "" {
		t.Fatalf("credential.SecretId = %q, want empty after failed init", provider.credential.SecretId)
	}
}
