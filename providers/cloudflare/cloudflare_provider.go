// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"errors"

	"github.com/chenrui333/terraformer/terraformutils"
)

type CloudflareProvider struct { //nolint
	terraformutils.Provider
}

func (p *CloudflareProvider) Init(_ []string) error {
	return nil
}

func (p *CloudflareProvider) GetName() string {
	return "cloudflare"
}

func (p *CloudflareProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func (CloudflareProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *CloudflareProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"access":                &AccessGenerator{},
		"dns":                   &DNSGenerator{},
		"firewall":              &FirewallGenerator{},
		"page_rule":             &PageRulesGenerator{},
		"account_member":        &AccountMemberGenerator{},
		"certificates":          &CertificatesGenerator{},
		"lists":                 &ListsGenerator{},
		"email_routing":         &EmailRoutingGenerator{},
		"load_balancing":        &LoadBalancingGenerator{},
		"logpush":               &LogpushGenerator{},
		"magic_wan":             &MagicWANGenerator{},
		"media_platform":        &MediaPlatformGenerator{},
		"network_edge":          &NetworkEdgeGenerator{},
		"notifications":         &NotificationsGenerator{},
		"pages":                 &PagesGenerator{},
		"ruleset":               &RulesetGenerator{},
		"security":              &SecurityGenerator{},
		"settings":              &SettingsGenerator{},
		"storage":               &StorageGenerator{},
		"turnstile":             &TurnstileGenerator{},
		"tunnel":                &TunnelGenerator{},
		"waiting_room":          &WaitingRoomGenerator{},
		"web_analytics":         &WebAnalyticsGenerator{},
		"workers":               &WorkersGenerator{},
		"zero_trust_device_dlp": &ZeroTrustDeviceDLPGenerator{},
		"zero_trust_gateway":    &ZeroTrustGatewayGenerator{},
	}
}

func (p *CloudflareProvider) InitService(serviceName string, verbose bool) error {
	if !terraformutils.SelectProviderService(&p.Provider, p.GetSupportedService(), serviceName, verbose, p.GetName()) {
		return errors.New("cloudflare: " + serviceName + " not supported service")
	}

	return nil
}
