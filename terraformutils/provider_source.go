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

import (
	"fmt"
	"strings"
)

var providerSources = map[string]string{
	"alicloud":      "aliyun/alicloud",
	"auth0":         "auth0/auth0",
	"cloudflare":    "cloudflare/cloudflare",
	"commercetools": "labd/commercetools",
	"datadog":       "DataDog/datadog",
	"digitalocean":  "digitalocean/digitalocean",
	"github":        "integrations/github",
	"gitlab":        "gitlabhq/gitlab",
	"gmailfilter":   "yamamoto-febc/terraform-provider-gmailfilter",
	"grafana":       "grafana/grafana",
	"heroku":        "heroku/heroku",
	"honeycombio":   "honeycombio/honeycombio",
	"ibm":           "IBM-Cloud/ibm",
	"ionoscloud":    "ionos-cloud/ionoscloud",
	"keycloak":      "keycloak/keycloak",
	"launchdarkly":  "launchdarkly/launchdarkly",
	"linode":        "linode/linode",
	"logzio":        "logzio/logzio",
	"metal":         "equinix/metal",
	"mikrotik":      "ddelnano/mikrotik",
	"newrelic":      "newrelic/newrelic",
	"ns1":           "ns1-terraform/ns1",
	"octopusdeploy": "OctopusDeployLabs/octopusdeploy",
	"okta":          "okta/okta",
	"opal":          "opalsecurity/opal",
	"openstack":     "terraform-provider-openstack/openstack",
	"opsgenie":      "opsgenie/opsgenie",
	"pagerduty":     "PagerDuty/pagerduty",
	"panos":         "PaloAltoNetworks/panos",
	"rabbitmq":      "cyrilgdn/rabbitmq",
	"tencentcloud":  "tencentcloudstack/tencentcloud",
	"vultr":         "vultr/vultr",
	"xenorchestra":  "terra-farm/xenorchestra",
	"yandex":        "yandex-cloud/yandex",
}

func ProviderSource(providerName string) string {
	if strings.Contains(providerName, "/") {
		return providerName
	}
	if source, ok := providerSources[providerName]; ok {
		return source
	}
	return "hashicorp/" + providerName
}

func ProviderConfigAddress(providerName string) string {
	return fmt.Sprintf("provider[\"registry.terraform.io/%s\"]", ProviderSource(providerName))
}
