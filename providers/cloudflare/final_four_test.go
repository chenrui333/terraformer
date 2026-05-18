// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCloudflareFinalFourUnsupportedResourcesMetadata(t *testing.T) {
	data, err := os.ReadFile("unsupported_resources.json")
	if err != nil {
		t.Fatalf("read unsupported resources: %v", err)
	}
	var metadata cloudflareUnsupportedResourcesFile
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("decode unsupported resources: %v", err)
	}

	statuses := map[string]string{}
	for _, resource := range metadata.Resources {
		statuses[resource.Resource] = resource.Status
	}
	for _, resource := range []string{
		"cloudflare_ai_gateway_dynamic_routing",
		"cloudflare_zero_trust_tunnel_cloudflared_config",
	} {
		if got := statuses[resource]; got != "deferred" {
			t.Fatalf("%s status = %q, want deferred", resource, got)
		}
	}
	for _, implemented := range []string{
		"cloudflare_connectivity_directory_service",
		"cloudflare_zero_trust_tunnel_cloudflared_route",
	} {
		if _, ok := statuses[implemented]; ok {
			t.Fatalf("implemented resource %s should not have unsupported metadata", implemented)
		}
	}
}
