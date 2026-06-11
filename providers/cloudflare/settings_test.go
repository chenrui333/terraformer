// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	cf "github.com/chenrui333/terraformer/providers/cloudflare/internal/cloudflarev7"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
	"github.com/zclconf/go-cty/cty"
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
	trueValue := true

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
		ManagedRequestHeaders: []cloudflareManagedTransformHeader{{Enabled: true}},
	}) {
		t.Fatal("enabled managed transform without an ID should be skipped")
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
	if !cloudflareZoneDNSSECShouldImport(cloudflareZoneDNSSECSetting{Status: "active"}) {
		t.Fatal("active DNSSEC should be imported")
	}
	if cloudflareZoneDNSSECShouldImport(cloudflareZoneDNSSECSetting{Status: "disabled"}) {
		t.Fatal("disabled DNSSEC without explicit options should be skipped")
	}
	if !cloudflareZoneDNSSECShouldImport(cloudflareZoneDNSSECSetting{Status: "pending"}) {
		t.Fatal("pending DNSSEC should be imported")
	}
	if !cloudflareZoneDNSSECShouldImport(cloudflareZoneDNSSECSetting{Status: "pending", DNSSECMultiSigner: &trueValue}) {
		t.Fatal("explicit DNSSEC options should be imported for transitional statuses")
	}
	if !cloudflareZoneDNSSECShouldImport(cloudflareZoneDNSSECSetting{Status: "pending-disabled"}) {
		t.Fatal("pending-disabled DNSSEC should be imported")
	}
	if cloudflareZoneDNSSECShouldImport(cloudflareZoneDNSSECSetting{Status: "unknown", DNSSECMultiSigner: &trueValue}) {
		t.Fatal("unknown DNSSEC status should be skipped")
	}
	if !cloudflareZoneDNSSECShouldImport(cloudflareZoneDNSSECSetting{Status: "disabled", DNSSECMultiSigner: &trueValue}) {
		t.Fatal("explicit DNSSEC options should be imported")
	}
}

func TestCloudflareOptionalSettingsMissing(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "not found", err: testCloudflareNotFoundError("not found"), want: true},
		{name: "legacy forbidden", err: testCloudflareForbiddenError("permission denied"), want: true},
		{name: "authorization forbidden", err: testCloudflareAuthorizationError("permission denied"), want: true},
		{name: "request error", err: testCloudflareRequestError("bad request"), want: false},
		{name: "generic error", err: errors.New("boom"), want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := cloudflareOptionalSettingsMissing(tt.err); got != tt.want {
				t.Fatalf("cloudflareOptionalSettingsMissing() = %t, want %t", got, tt.want)
			}
		})
	}
}

func testCloudflareForbiddenError(messages ...string) error {
	responseInfo := make([]cf.ResponseInfo, 0, len(messages))
	for _, message := range messages {
		responseInfo = append(responseInfo, cf.ResponseInfo{Message: message})
	}
	err := cf.NewAuthenticationError(&cf.Error{
		Errors:        responseInfo,
		ErrorMessages: messages,
	})
	return &err
}

func testCloudflareAuthorizationError(messages ...string) error {
	responseInfo := make([]cf.ResponseInfo, 0, len(messages))
	for _, message := range messages {
		responseInfo = append(responseInfo, cf.ResponseInfo{Message: message})
	}
	err := cf.NewAuthorizationError(&cf.Error{
		Errors:        responseInfo,
		ErrorMessages: messages,
	})
	return &err
}

func TestAppendLeakedCredentialCheckZoneResourcePreservesID(t *testing.T) {
	generator := &SettingsGenerator{}
	zone := cf.Zone{ID: "zone-123", Name: "example.com"}

	generator.appendLeakedCredentialCheckZoneResource(zone)

	if len(generator.Resources) != 1 {
		t.Fatalf("resources length = %d, want 1", len(generator.Resources))
	}
	resource := generator.Resources[0]
	if resource.InstanceInfo.Type != "cloudflare_leaked_credential_check" {
		t.Fatalf("resource type = %q, want cloudflare_leaked_credential_check", resource.InstanceInfo.Type)
	}
	if resource.InstanceState.ID != "zone-123" {
		t.Fatalf("resource ID = %q, want zone-123", resource.InstanceState.ID)
	}
	if resource.InstanceState.Attributes["enabled"] != "true" {
		t.Fatalf("enabled attribute = %q, want true", resource.InstanceState.Attributes["enabled"])
	}
	if got := resource.InstanceState.Meta[tfcompat.MetaKeyPreserveIDAfterRefresh]; got != true {
		t.Fatalf("preserve ID metadata = %#v, want true", got)
	}
}

func TestCloudflareManagedTransformsAttributes(t *testing.T) {
	attributes := cloudflareManagedTransformsAttributes(cloudflareManagedTransformsSetting{
		ManagedRequestHeaders: []cloudflareManagedTransformHeader{
			{ID: "add_true_client_ip_headers", Enabled: true},
			{ID: "remove_visitor_ip_headers", Enabled: false},
			{Enabled: true},
		},
		ManagedResponseHeaders: []cloudflareManagedTransformHeader{
			{ID: "add_security_headers", Enabled: true},
		},
	})

	want := map[string]string{
		"managed_request_headers.#":          "1",
		"managed_request_headers.0.id":       "add_true_client_ip_headers",
		"managed_request_headers.0.enabled":  "true",
		"managed_response_headers.#":         "1",
		"managed_response_headers.0.id":      "add_security_headers",
		"managed_response_headers.0.enabled": "true",
	}
	for key, wantValue := range want {
		if got := attributes[key]; got != wantValue {
			t.Fatalf("attribute %s = %q, want %q", key, got, wantValue)
		}
	}
	if _, ok := attributes["managed_request_headers.1.id"]; ok {
		t.Fatal("disabled or invalid request transform should not be seeded")
	}
}

func TestCloudflareManagedTransformsStatePreservesEmptyRequiredSets(t *testing.T) {
	zone := cf.Zone{ID: "zone-123", Name: "example.com"}
	headerType := cty.Object(map[string]cty.Type{
		"enabled": cty.String,
		"id":      cty.String,
	})
	impliedType := cty.Object(map[string]cty.Type{
		"managed_request_headers":  cty.Set(headerType),
		"managed_response_headers": cty.Set(headerType),
		"zone_id":                  cty.String,
	})

	for _, tt := range []struct {
		name       string
		setting    cloudflareManagedTransformsSetting
		emptyKey   string
		presentKey string
	}{
		{
			name: "request only",
			setting: cloudflareManagedTransformsSetting{
				ManagedRequestHeaders: []cloudflareManagedTransformHeader{{ID: "add_true_client_ip_headers", Enabled: true}},
			},
			emptyKey:   "managed_response_headers",
			presentKey: "managed_request_headers",
		},
		{
			name: "response only",
			setting: cloudflareManagedTransformsSetting{
				ManagedResponseHeaders: []cloudflareManagedTransformHeader{{ID: "add_security_headers", Enabled: true}},
			},
			emptyKey:   "managed_request_headers",
			presentKey: "managed_response_headers",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			attributes, additionalFields := cloudflareManagedTransformsState(tt.setting)
			resource := cloudflareZoneSingletonSettingResourceWithAttributesAndAdditionalFields(
				zone,
				"cloudflare_managed_transforms",
				"managed_transforms",
				attributes,
				additionalFields,
			)

			if got := resource.InstanceState.Attributes[tt.emptyKey+".#"]; got != "0" {
				t.Fatalf("%s.# = %q, want 0", tt.emptyKey, got)
			}
			additionalValue, ok := resource.AdditionalFields[tt.emptyKey].([]interface{})
			if !ok || len(additionalValue) != 0 {
				t.Fatalf("AdditionalFields[%s] = %#v, want empty list", tt.emptyKey, resource.AdditionalFields[tt.emptyKey])
			}

			parser := terraformutils.NewFlatmapParser(resource.InstanceState.Attributes, nil, nil)
			if err := resource.ParseTFstate(parser, impliedType); err != nil {
				t.Fatalf("ParseTFstate() error = %v", err)
			}
			emptyValue, ok := resource.Item[tt.emptyKey].([]interface{})
			if !ok || len(emptyValue) != 0 {
				t.Fatalf("Item[%s] = %#v, want empty list", tt.emptyKey, resource.Item[tt.emptyKey])
			}
			presentValue, ok := resource.Item[tt.presentKey].([]interface{})
			if !ok || len(presentValue) != 1 {
				t.Fatalf("Item[%s] = %#v, want one transform", tt.presentKey, resource.Item[tt.presentKey])
			}

			hcl, err := terraformutils.HclPrintResource([]terraformutils.Resource{resource}, map[string]interface{}{}, "hcl", true)
			if err != nil {
				t.Fatalf("HclPrintResource() error = %v", err)
			}
			if want := tt.emptyKey + " = []"; !strings.Contains(string(hcl), want) {
				t.Fatalf("generated HCL does not contain %q:\n%s", want, string(hcl))
			}
		})
	}
}

func TestCloudflareZoneHoldConfigured(t *testing.T) {
	hold := true
	released := false
	includeSubdomains := true
	now := time.Date(2026, time.May, 17, 12, 0, 0, 0, time.UTC)
	futureHoldAfter := now.Add(time.Hour)
	pastHoldAfter := now.Add(-time.Hour)

	for _, tt := range []struct {
		name    string
		setting cf.ZoneHold
		want    bool
	}{
		{name: "empty", setting: cf.ZoneHold{}, want: false},
		{name: "active hold", setting: cf.ZoneHold{Hold: &hold}, want: true},
		{name: "include subdomains without hold signal", setting: cf.ZoneHold{IncludeSubdomains: &includeSubdomains}, want: true},
		{name: "future hold after without hold signal", setting: cf.ZoneHold{HoldAfter: &futureHoldAfter}, want: true},
		{name: "released hold without hold after", setting: cf.ZoneHold{Hold: &released, IncludeSubdomains: &includeSubdomains}, want: false},
		{name: "released hold with past hold after", setting: cf.ZoneHold{Hold: &released, IncludeSubdomains: &includeSubdomains, HoldAfter: &pastHoldAfter}, want: false},
		{name: "released hold with future hold after", setting: cf.ZoneHold{Hold: &released, HoldAfter: &futureHoldAfter}, want: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := cloudflareZoneHoldConfiguredAt(tt.setting, now); got != tt.want {
				t.Fatalf("cloudflareZoneHoldConfigured() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestCloudflareZoneDNSSECResource(t *testing.T) {
	trueValue := true
	falseValue := false
	zone := cf.Zone{ID: "zone-123", Name: "example.com"}
	resource := cloudflareZoneDNSSECResource(zone, cloudflareZoneDNSSECSetting{
		Status:            "ACTIVE",
		DNSSECMultiSigner: &trueValue,
		DNSSECPresigned:   &falseValue,
	})

	if resource.InstanceInfo.Type != "cloudflare_zone_dnssec" {
		t.Fatalf("resource type = %q, want cloudflare_zone_dnssec", resource.InstanceInfo.Type)
	}
	if got, want := resource.ResourceName, terraformutils.TfSanitize(cloudflareResourceName("example.com", "zone_dnssec")); got != want {
		t.Fatalf("resource name = %q, want %s", got, want)
	}
	if got := resource.InstanceState.Attributes["zone_id"]; got != "zone-123" {
		t.Fatalf("zone_id = %q, want zone-123", got)
	}
	if got := resource.InstanceState.Attributes["status"]; got != "active" {
		t.Fatalf("status = %q, want active", got)
	}
	if got := resource.InstanceState.Attributes["dnssec_multi_signer"]; got != "true" {
		t.Fatalf("dnssec_multi_signer = %q, want true", got)
	}
	if got := resource.InstanceState.Attributes["dnssec_presigned"]; got != "false" {
		t.Fatalf("dnssec_presigned = %q, want false", got)
	}
	if _, ok := resource.InstanceState.Attributes["dnssec_use_nsec3"]; ok {
		t.Fatal("nil dnssec_use_nsec3 should not be seeded")
	}
	for _, key := range cloudflareZoneDNSSECComputedKeys {
		if !cloudflareResourceIgnoresKey(resource, key) {
			t.Fatalf("DNSSEC resource should ignore computed key %q", key)
		}
	}
	if cloudflareResourceIgnoresKey(resource, "^status$") {
		t.Fatal("configurable DNSSEC status should not be ignored")
	}

	transitionalResource := cloudflareZoneDNSSECResource(zone, cloudflareZoneDNSSECSetting{
		Status:         "pending",
		DNSSECUseNsec3: &trueValue,
	})
	if got := transitionalResource.InstanceState.Attributes["status"]; got != "active" {
		t.Fatalf("transitional DNSSEC status = %q, want active", got)
	}
	if got := transitionalResource.InstanceState.Attributes["dnssec_use_nsec3"]; got != "true" {
		t.Fatalf("dnssec_use_nsec3 = %q, want true", got)
	}
	if cloudflareResourceIgnoresKey(transitionalResource, "^status$") {
		t.Fatal("transitional DNSSEC desired status should not be ignored")
	}
	if got := transitionalResource.AdditionalFields["status"]; got != "active" {
		t.Fatalf("transitional DNSSEC AdditionalFields status = %#v, want active", got)
	}

	transitionalResource.InstanceState.Attributes["status"] = "pending"
	parser := terraformutils.NewFlatmapParser(transitionalResource.InstanceState.Attributes, nil, nil)
	impliedType := cty.Object(map[string]cty.Type{
		"status":  cty.String,
		"zone_id": cty.String,
	})
	if err := transitionalResource.ParseTFstate(parser, impliedType); err != nil {
		t.Fatalf("ParseTFstate() error = %v", err)
	}
	if got := transitionalResource.Item["status"]; got != "active" {
		t.Fatalf("parsed transitional DNSSEC status = %q, want active", got)
	}

	disablingResource := cloudflareZoneDNSSECResource(zone, cloudflareZoneDNSSECSetting{Status: "pending-disabled"})
	if got := disablingResource.InstanceState.Attributes["status"]; got != "disabled" {
		t.Fatalf("pending-disabled DNSSEC status = %q, want disabled", got)
	}
}

func TestCloudflareZoneSettingShouldImport(t *testing.T) {
	for _, tt := range []struct {
		name    string
		setting cloudflareZoneSetting
		want    bool
	}{
		{name: "empty", setting: cloudflareZoneSetting{}, want: false},
		{
			name: "supported modified editable string",
			setting: cloudflareZoneSetting{
				ID:         "always_use_https",
				Editable:   true,
				ModifiedOn: "2026-05-17T12:00:00Z",
				Value:      json.RawMessage(`"on"`),
			},
			want: true,
		},
		{
			name: "supported modified default value",
			setting: cloudflareZoneSetting{
				ID:         "always_use_https",
				Editable:   true,
				ModifiedOn: "2026-05-17T12:00:00Z",
				Value:      json.RawMessage(`"off"`),
			},
			want: true,
		},
		{
			name: "supported non default without modified timestamp",
			setting: cloudflareZoneSetting{
				ID:       "always_use_https",
				Editable: true,
				Value:    json.RawMessage(`"on"`),
			},
			want: false,
		},
		{
			name: "supported but not editable",
			setting: cloudflareZoneSetting{
				ID:         "always_use_https",
				ModifiedOn: "2026-05-17T12:00:00Z",
				Value:      json.RawMessage(`"on"`),
			},
			want: false,
		},
		{
			name: "unsupported setting",
			setting: cloudflareZoneSetting{
				ID:         "development_mode",
				Editable:   true,
				ModifiedOn: "2026-05-17T12:00:00Z",
				Value:      json.RawMessage(`"on"`),
			},
			want: false,
		},
		{
			name: "non string value",
			setting: cloudflareZoneSetting{
				ID:         "always_use_https",
				Editable:   true,
				ModifiedOn: "2026-05-17T12:00:00Z",
				Value:      json.RawMessage(`14400`),
			},
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := cloudflareZoneSettingShouldImport(tt.setting); got != tt.want {
				t.Fatalf("cloudflareZoneSettingShouldImport() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestCloudflareZoneSettingSkipsUnmodifiedDefaultResponses(t *testing.T) {
	for _, tt := range []struct {
		settingID    string
		defaultValue string
	}{
		{settingID: "always_online", defaultValue: "off"},
		{settingID: "brotli", defaultValue: "off"},
		{settingID: "tls_1_3", defaultValue: "on"},
		{settingID: "websockets", defaultValue: "on"},
	} {
		t.Run(tt.settingID, func(t *testing.T) {
			setting := cloudflareZoneSetting{
				ID:       tt.settingID,
				Editable: true,
				Value:    cloudflareZoneSettingTestStringValue(tt.defaultValue),
			}
			if cloudflareZoneSettingShouldImport(setting) {
				t.Fatalf("%s unmodified default value %q should not import", tt.settingID, tt.defaultValue)
			}
		})
	}
}

func TestCloudflareZoneSettingImportsModifiedAllowlistedValues(t *testing.T) {
	for _, tt := range []struct {
		settingID string
		value     string
	}{
		{settingID: "always_online", value: "on"},
		{settingID: "brotli", value: "on"},
		{settingID: "tls_1_3", value: "off"},
		{settingID: "websockets", value: "off"},
	} {
		t.Run(tt.settingID, func(t *testing.T) {
			setting := cloudflareZoneSetting{
				ID:         tt.settingID,
				Editable:   true,
				ModifiedOn: "2026-05-17T12:00:00Z",
				Value:      cloudflareZoneSettingTestStringValue(tt.value),
			}
			if !cloudflareZoneSettingShouldImport(setting) {
				t.Fatalf("%s modified value %q should import", tt.settingID, tt.value)
			}
		})
	}
}

func TestCloudflareZoneSettingRawNullModifiedOnSkipsDefaultResponses(t *testing.T) {
	response := []byte("[{\"id\":\"always_online\",\"editable\":true,\"modified_on\":null,\"value\":\"off\"},{\"id\":\"tls_1_3\",\"editable\":true,\"modified_on\":null,\"value\":\"on\"},{\"id\":\"websockets\",\"editable\":true,\"modified_on\":null,\"value\":\"on\"}]")
	var settings []cloudflareZoneSetting
	if err := json.Unmarshal(response, &settings); err != nil {
		t.Fatalf("unmarshal zone settings response = %v", err)
	}
	for _, setting := range settings {
		if cloudflareZoneSettingShouldImport(setting) {
			t.Fatalf("%s with null modified_on should not import", setting.ID)
		}
	}
}

func TestCloudflareZoneSettingResource(t *testing.T) {
	zone := cf.Zone{ID: "zone-123", Name: "example.com"}
	resource := cloudflareZoneSettingResource(zone, cloudflareZoneSetting{
		ID:         "always_use_https",
		Editable:   true,
		ModifiedOn: "2026-05-17T12:00:00Z",
		Value:      json.RawMessage(`"on"`),
	})

	if resource.InstanceInfo.Type != "cloudflare_zone_setting" {
		t.Fatalf("resource type = %q, want cloudflare_zone_setting", resource.InstanceInfo.Type)
	}
	if got, want := resource.ResourceName, terraformutils.TfSanitize(cloudflareResourceName("example.com", "zone-123", "zone_setting", "always_use_https")); got != want {
		t.Fatalf("resource name = %q, want %s", got, want)
	}
	if got := resource.InstanceState.ID; got != "always_use_https" {
		t.Fatalf("resource ID = %q, want always_use_https", got)
	}
	if got := resource.InstanceState.Attributes["zone_id"]; got != "zone-123" {
		t.Fatalf("zone_id = %q, want zone-123", got)
	}
	if got := resource.InstanceState.Attributes["setting_id"]; got != "always_use_https" {
		t.Fatalf("setting_id = %q, want always_use_https", got)
	}
	if got := resource.InstanceState.Attributes["value"]; got != "on" {
		t.Fatalf("value = %q, want on", got)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "zone-123/always_use_https" {
		t.Fatalf("import_id = %v, want zone-123/always_use_https", got)
	}
	if got := resource.AdditionalFields["value"]; got != "on" {
		t.Fatalf("AdditionalFields value = %#v, want on", got)
	}
	for _, key := range cloudflareZoneSettingIgnoredKeys {
		if !cloudflareResourceIgnoresKey(resource, key) {
			t.Fatalf("zone setting resource should ignore key %q", key)
		}
	}

	resource.InstanceState.Attributes["editable"] = "true"
	resource.InstanceState.Attributes["modified_on"] = "2026-05-17T12:00:00Z"
	parser := terraformutils.NewFlatmapParser(resource.InstanceState.Attributes, cloudflareResourceIgnoreRegexps(resource), nil)
	impliedType := cty.Object(map[string]cty.Type{
		"editable":    cty.Bool,
		"modified_on": cty.String,
		"setting_id":  cty.String,
		"value":       cty.DynamicPseudoType,
		"zone_id":     cty.String,
	})
	if err := resource.ParseTFstate(parser, impliedType); err != nil {
		t.Fatalf("ParseTFstate() error = %v", err)
	}
	if got := resource.Item["value"]; got != "on" {
		t.Fatalf("parsed value = %#v, want on", got)
	}
	if _, ok := resource.Item["editable"]; ok {
		t.Fatal("computed editable field should be ignored")
	}
}

func TestCloudflareZoneSettingRawResponseIgnoresBooleanTimeRemaining(t *testing.T) {
	response := []byte("[{\"id\":\"development_mode\",\"editable\":true,\"modified_on\":\"2026-05-17T12:00:00Z\",\"value\":\"off\",\"time_remaining\":false},{\"id\":\"always_use_https\",\"editable\":true,\"modified_on\":\"2026-05-17T12:00:00Z\",\"value\":\"on\",\"time_remaining\":0}]")
	var settings []cloudflareZoneSetting
	if err := json.Unmarshal(response, &settings); err != nil {
		t.Fatalf("unmarshal zone settings response = %v", err)
	}
	if len(settings) != 2 {
		t.Fatalf("settings length = %d, want 2", len(settings))
	}
	if cloudflareZoneSettingShouldImport(settings[0]) {
		t.Fatal("development_mode should remain outside the zone setting allowlist")
	}
	if !cloudflareZoneSettingShouldImport(settings[1]) {
		t.Fatal("always_use_https non-default value should import")
	}
}

func cloudflareZoneSettingTestStringValue(value string) json.RawMessage {
	return json.RawMessage(`"` + value + `"`)
}

func cloudflareResourceIgnoresKey(resource terraformutils.Resource, key string) bool {
	for _, ignoreKey := range resource.IgnoreKeys {
		if ignoreKey == key {
			return true
		}
	}
	return false
}

func cloudflareResourceIgnoreRegexps(resource terraformutils.Resource) []*regexp.Regexp {
	regexps := make([]*regexp.Regexp, 0, len(resource.IgnoreKeys))
	for _, ignoreKey := range resource.IgnoreKeys {
		regexps = append(regexps, regexp.MustCompile(ignoreKey))
	}
	return regexps
}

func TestCloudflareZoneHoldAttributes(t *testing.T) {
	includeSubdomains := true
	now := time.Date(2026, time.May, 17, 12, 0, 0, 0, time.UTC)
	futureHoldAfter := now.Add(time.Hour)
	pastHoldAfter := now.Add(-time.Hour)
	attributes := cloudflareZoneHoldAttributesAt(cf.ZoneHold{
		IncludeSubdomains: &includeSubdomains,
		HoldAfter:         &futureHoldAfter,
	}, now)
	if got := attributes["include_subdomains"]; got != "true" {
		t.Fatalf("include_subdomains = %q, want true", got)
	}
	if got := attributes["hold_after"]; got != futureHoldAfter.Format(time.RFC3339Nano) {
		t.Fatalf("hold_after = %q, want %q", got, futureHoldAfter.Format(time.RFC3339Nano))
	}
	attributes = cloudflareZoneHoldAttributesAt(cf.ZoneHold{HoldAfter: &pastHoldAfter}, now)
	if _, ok := attributes["hold_after"]; ok {
		t.Fatal("past hold_after should not be seeded")
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
