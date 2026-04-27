// Copyright 2026 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package terraformutils

import "testing"

func TestProviderSource(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		"aws":                         "hashicorp/aws",
		"cloudflare":                  "cloudflare/cloudflare",
		"github":                      "integrations/github",
		"keycloak":                    "keycloak/keycloak",
		"registry.example.com/custom": "registry.example.com/custom",
		"tencentcloud":                "tencentcloudstack/tencentcloud",
	}

	for provider, want := range testCases {
		t.Run(provider, func(t *testing.T) {
			t.Parallel()
			if got := ProviderSource(provider); got != want {
				t.Fatalf("ProviderSource(%q) = %q, want %q", provider, got, want)
			}
		})
	}
}

func TestProviderConfigAddress(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		"aws":        "provider[\"registry.terraform.io/hashicorp/aws\"]",
		"cloudflare": "provider[\"registry.terraform.io/cloudflare/cloudflare\"]",
	}

	for provider, want := range testCases {
		t.Run(provider, func(t *testing.T) {
			t.Parallel()
			if got := ProviderConfigAddress(provider); got != want {
				t.Fatalf("ProviderConfigAddress(%q) = %q, want %q", provider, got, want)
			}
		})
	}
}
