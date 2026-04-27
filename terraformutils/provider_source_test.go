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

	// Keep this table in sync with provider GetName values so Terraform 0.13+
	// generated configs and states do not silently fall back to the wrong namespace.
	testCases := map[string]string{
		"alicloud":                    "aliyun/alicloud",
		"auth0":                       "auth0/auth0",
		"aws":                         "hashicorp/aws",
		"azuread":                     "hashicorp/azuread",
		"azuredevops":                 "microsoft/azuredevops",
		"azurerm":                     "hashicorp/azurerm",
		"cloudflare":                  "cloudflare/cloudflare",
		"commercetools":               "labd/commercetools",
		"datadog":                     "DataDog/datadog",
		"digitalocean":                "digitalocean/digitalocean",
		"fastly":                      "fastly/fastly",
		"github":                      "integrations/github",
		"gitlab":                      "gitlabhq/gitlab",
		"gmailfilter":                 "yamamoto-febc/terraform-provider-gmailfilter",
		"google":                      "hashicorp/google",
		"google-beta":                 "hashicorp/google-beta",
		"grafana":                     "grafana/grafana",
		"heroku":                      "heroku/heroku",
		"honeycombio":                 "honeycombio/honeycombio",
		"ibm":                         "IBM-Cloud/ibm",
		"ionoscloud":                  "ionos-cloud/ionoscloud",
		"keycloak":                    "keycloak/keycloak",
		"kubernetes":                  "hashicorp/kubernetes",
		"launchdarkly":                "launchdarkly/launchdarkly",
		"linode":                      "linode/linode",
		"logzio":                      "logzio/logzio",
		"mackerel":                    "mackerelio-labs/mackerel",
		"metal":                       "equinix/metal",
		"mikrotik":                    "ddelnano/mikrotik",
		"myrasec":                     "Myra-Security-GmbH/myrasec",
		"newrelic":                    "newrelic/newrelic",
		"ns1":                         "ns1-terraform/ns1",
		"octopusdeploy":               "OctopusDeployLabs/octopusdeploy",
		"okta":                        "okta/okta",
		"opal":                        "opalsecurity/opal",
		"openstack":                   "terraform-provider-openstack/openstack",
		"opsgenie":                    "opsgenie/opsgenie",
		"pagerduty":                   "PagerDuty/pagerduty",
		"panos":                       "PaloAltoNetworks/panos",
		"rabbitmq":                    "cyrilgdn/rabbitmq",
		"registry.example.com/custom": "registry.example.com/custom",
		"tencentcloud":                "tencentcloudstack/tencentcloud",
		"vault":                       "hashicorp/vault",
		"vultr":                       "vultr/vultr",
		"xenorchestra":                "terra-farm/xenorchestra",
		"yandex":                      "yandex-cloud/yandex",
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
