// SPDX-License-Identifier: Apache-2.0

package tencentcloud

import (
	"strings"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

func TestCosInitResourcesReturnsServiceURLParseError(t *testing.T) {
	generator := &CosGenerator{
		TencentCloudService: TencentCloudService{
			Service: terraformutils.Service{
				Args: map[string]interface{}{
					"region":     "bad\nregion",
					"credential": common.Credential{},
				},
			},
		},
	}

	err := generator.InitResources()
	if err == nil {
		t.Fatal("expected COS service URL parse error")
	}
	if !strings.Contains(err.Error(), "parse Tencent COS service URL") {
		t.Fatalf("error = %q, want COS service URL parse context", err)
	}
}
