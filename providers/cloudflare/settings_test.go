// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"testing"
	"time"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

func TestCloudflareProviderSupportsSettingsService(t *testing.T) {
	services := (&CloudflareProvider{}).GetSupportedService()
	if _, ok := services["settings"]; !ok {
		t.Fatal("Cloudflare provider is missing settings service")
	}
}

func TestAppendZoneSingletonSettingResource(t *testing.T) {
	generator := &SettingsGenerator{}
	zone := cf.Zone{ID: "zone-123", Name: "example.com"}

	generator.appendZoneSingletonSettingResource(zone, "cloudflare_argo_smart_routing", "argo_smart_routing")

	if len(generator.Resources) != 1 {
		t.Fatalf("resources length = %d, want 1", len(generator.Resources))
	}
	resource := generator.Resources[0]
	if resource.InstanceInfo.Type != "cloudflare_argo_smart_routing" {
		t.Fatalf("resource type = %q, want cloudflare_argo_smart_routing", resource.InstanceInfo.Type)
	}
	if got, want := resource.ResourceName, terraformutils.TfSanitize(cloudflareResourceName("example.com", "argo_smart_routing")); got != want {
		t.Fatalf("resource name = %q, want %s", got, want)
	}
	if resource.InstanceState.ID != "zone-123" {
		t.Fatalf("resource ID = %q, want zone-123", resource.InstanceState.ID)
	}
	if resource.InstanceState.Attributes["zone_id"] != "zone-123" {
		t.Fatalf("zone_id attribute = %q, want zone-123", resource.InstanceState.Attributes["zone_id"])
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "zone-123" {
		t.Fatalf("import_id = %v, want zone-123", got)
	}
}

func TestCloudflareSettingsDefaultImportPolicy(t *testing.T) {
	if !cloudflareSettingIsOn("on") {
		t.Fatal("cloudflareSettingIsOn(on) = false, want true")
	}
	if cloudflareSettingIsOn("off") {
		t.Fatal("cloudflareSettingIsOn(off) = true, want false")
	}
	if !cloudflareUniversalSSLSettingShouldImport(cf.UniversalSSLSetting{Enabled: false}) {
		t.Fatal("disabled Universal SSL should be imported")
	}
	if cloudflareUniversalSSLSettingShouldImport(cf.UniversalSSLSetting{Enabled: true}) {
		t.Fatal("enabled Universal SSL should be skipped as the normal default")
	}
	if cloudflareURLNormalizationSettingsShouldImport(cf.URLNormalizationSettings{Scope: "incoming", Type: "rfc3986"}) {
		t.Fatal("default URL normalization should be skipped")
	}
	if !cloudflareURLNormalizationSettingsShouldImport(cf.URLNormalizationSettings{Scope: "incoming", Type: "cloudflare"}) {
		t.Fatal("non-default URL normalization type should be imported")
	}
	if !cloudflareURLNormalizationSettingsShouldImport(cf.URLNormalizationSettings{Scope: "both", Type: "rfc3986"}) {
		t.Fatal("non-default URL normalization should be imported")
	}
	if !cloudflareManagedTransformsConfigured(cloudflareManagedTransformsSetting{
		ManagedRequestHeaders: []cloudflareManagedTransformHeader{{ID: "add_true_client_ip_headers", Enabled: true}},
	}) {
		t.Fatal("enabled managed transform should be imported")
	}
	if cloudflareManagedTransformsConfigured(cloudflareManagedTransformsSetting{
		ManagedRequestHeaders:  []cloudflareManagedTransformHeader{{ID: "add_true_client_ip_headers", Enabled: false}},
		ManagedResponseHeaders: []cloudflareManagedTransformHeader{{ID: "add_security_headers", Enabled: false}},
	}) {
		t.Fatal("all-disabled managed transforms should be skipped")
	}
	if !cloudflareZoneCacheVariantsConfigured(cf.ZoneCacheVariantsValues{Avif: []string{"image/avif"}}) {
		t.Fatal("non-empty cache variants should be imported")
	}
	if cloudflareZoneCacheVariantsConfigured(cf.ZoneCacheVariantsValues{}) {
		t.Fatal("empty cache variants should be skipped")
	}
}

func TestCloudflareZoneHoldConfigured(t *testing.T) {
	hold := true
	includeSubdomains := true
	holdAfter := time.Now().UTC()

	for _, tt := range []struct {
		name    string
		setting cf.ZoneHold
		want    bool
	}{
		{name: "empty", setting: cf.ZoneHold{}, want: false},
		{name: "hold", setting: cf.ZoneHold{Hold: &hold}, want: true},
		{name: "include subdomains", setting: cf.ZoneHold{IncludeSubdomains: &includeSubdomains}, want: true},
		{name: "hold after", setting: cf.ZoneHold{HoldAfter: &holdAfter}, want: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := cloudflareZoneHoldConfigured(tt.setting); got != tt.want {
				t.Fatalf("cloudflareZoneHoldConfigured() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestCloudflareDNSZoneTransferConfigured(t *testing.T) {
	for _, tt := range []struct {
		name    string
		setting cloudflareDNSZoneTransferConfig
		want    bool
	}{
		{name: "empty", setting: cloudflareDNSZoneTransferConfig{}, want: false},
		{name: "id only", setting: cloudflareDNSZoneTransferConfig{ID: "transfer-123"}, want: false},
		{name: "name only", setting: cloudflareDNSZoneTransferConfig{Name: "example.com"}, want: false},
		{name: "peers", setting: cloudflareDNSZoneTransferConfig{Peers: []string{"peer-123"}}, want: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := cloudflareDNSZoneTransferConfigured(tt.setting); got != tt.want {
				t.Fatalf("cloudflareDNSZoneTransferConfigured() = %t, want %t", got, tt.want)
			}
		})
	}
}
