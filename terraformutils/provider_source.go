// SPDX-License-Identifier: Apache-2.0

package terraformutils

import (
	"fmt"
	"strings"
)

var providerSources = map[string]string{
	"alicloud":      "aliyun/alicloud",
	"auth0":         "auth0/auth0",
	"azuredevops":   "microsoft/azuredevops",
	"cloudflare":    "cloudflare/cloudflare",
	"commercetools": "labd/commercetools",
	"datadog":       "DataDog/datadog",
	"digitalocean":  "digitalocean/digitalocean",
	"fastly":        "fastly/fastly",
	"github":        "integrations/github",
	"gitlab":        "gitlabhq/gitlab",
	"gmailfilter":   "yamamoto-febc/gmailfilter",
	"grafana":       "grafana/grafana",
	"heroku":        "heroku/heroku",
	"honeycombio":   "honeycombio/honeycombio",
	"ibm":           "IBM-Cloud/ibm",
	"ionoscloud":    "ionos-cloud/ionoscloud",
	"keycloak":      "keycloak/keycloak",
	"launchdarkly":  "launchdarkly/launchdarkly",
	"linode":        "linode/linode",
	"logzio":        "logzio/logzio",
	"mackerel":      "mackerelio-labs/mackerel",
	"metal":         "equinix/metal",
	"mikrotik":      "ddelnano/mikrotik",
	"myrasec":       "Myra-Security-GmbH/myrasec",
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
