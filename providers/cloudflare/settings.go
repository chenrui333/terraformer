// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
	cf "github.com/cloudflare/cloudflare-go"
)

type SettingsGenerator struct {
	CloudflareService
}

type cloudflareAccountDNSInternalView struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Zones []string `json:"zones"`
}

type cloudflareDNSFirewall struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type cloudflareDNSZoneTransferACL struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type cloudflareDNSZoneTransferPeer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type cloudflareDNSZoneTransferConfig struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Peers []string `json:"peers"`
}

type cloudflareEnabledSetting struct {
	Enabled bool `json:"enabled"`
}

type cloudflareValueSetting struct {
	Value string `json:"value"`
}

type cloudflareManagedTransformsSetting struct {
	ManagedRequestHeaders  []cloudflareManagedTransformHeader `json:"managed_request_headers"`
	ManagedResponseHeaders []cloudflareManagedTransformHeader `json:"managed_response_headers"`
}

type cloudflareManagedTransformHeader struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

func (g *SettingsGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}

	if accountID := g.accountID(); accountID != "" {
		if err := g.appendAccountSettingsResources(ctx, api, accountID); err != nil {
			return err
		}
	}

	zones, err := cloudflareZones(ctx, api)
	if err != nil {
		return err
	}
	for _, zone := range zones {
		if err := g.appendZoneSettingsResources(ctx, api, zone); err != nil {
			return err
		}
	}
	return nil
}

func (g *SettingsGenerator) appendAccountSettingsResources(ctx context.Context, api *cf.API, accountID string) error {
	if err := g.appendAccountDNSInternalViewResources(ctx, api, accountID); err != nil {
		return err
	}
	if err := g.appendDNSFirewallResources(ctx, api, accountID); err != nil {
		return err
	}
	if err := g.appendDNSZoneTransferACLResources(ctx, api, accountID); err != nil {
		return err
	}
	if err := g.appendDNSZoneTransferPeerResources(ctx, api, accountID); err != nil {
		return err
	}
	return nil
}

func (g *SettingsGenerator) appendZoneSettingsResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	checks := []func(context.Context, *cf.API, cf.Zone) error{
		g.appendArgoSmartRoutingResource,
		g.appendArgoTieredCachingResource,
		g.appendAuthenticatedOriginPullsSettingsResource,
		g.appendCustomHostnameFallbackOriginResource,
		g.appendLeakedCredentialCheckResource,
		g.appendLogpullRetentionResource,
		g.appendManagedTransformsResource,
		g.appendRegionalTieredCacheResource,
		g.appendTieredCacheResource,
		g.appendTotalTLSResource,
		g.appendUniversalSSLSettingResource,
		g.appendURLNormalizationSettingsResource,
		g.appendWaitingRoomSettingsResource,
		g.appendZoneCacheReserveResource,
		g.appendZoneCacheVariantsResource,
		g.appendZoneHoldResource,
		g.appendDNSZoneTransfersIncomingResource,
		g.appendDNSZoneTransfersOutgoingResource,
	}
	for _, check := range checks {
		if err := check(ctx, api, zone); err != nil {
			return err
		}
	}
	return nil
}

func (g *SettingsGenerator) appendAccountDNSInternalViewResources(ctx context.Context, api *cf.API, accountID string) error {
	views, err := cloudflareListPaginated[cloudflareAccountDNSInternalView](ctx, api, fmt.Sprintf("/accounts/%s/dns_settings/views", accountID))
	if err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("list account DNS internal views for account %q: %w", accountID, err)
	}
	for _, view := range views {
		if view.ID == "" {
			continue
		}
		resource := terraformutils.NewResource(
			view.ID,
			cloudflareResourceName(accountID, "internal_view", view.Name, view.ID),
			"cloudflare_account_dns_settings_internal_view",
			"cloudflare",
			map[string]string{"account_id": accountID},
			[]string{},
			map[string]interface{}{},
		)
		setCloudflareImportID(&resource, accountID+"/"+view.ID)
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *SettingsGenerator) appendDNSFirewallResources(ctx context.Context, api *cf.API, accountID string) error {
	clusters, err := cloudflareListPaginated[cloudflareDNSFirewall](ctx, api, fmt.Sprintf("/accounts/%s/dns_firewall", accountID))
	if err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("list DNS firewall clusters for account %q: %w", accountID, err)
	}
	for _, cluster := range clusters {
		if cluster.ID == "" {
			continue
		}
		resource := terraformutils.NewResource(
			cluster.ID,
			cloudflareResourceName(accountID, "dns_firewall", cluster.Name, cluster.ID),
			"cloudflare_dns_firewall",
			"cloudflare",
			map[string]string{"account_id": accountID},
			[]string{},
			map[string]interface{}{},
		)
		setCloudflareImportID(&resource, accountID+"/"+cluster.ID)
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *SettingsGenerator) appendDNSZoneTransferACLResources(ctx context.Context, api *cf.API, accountID string) error {
	acls, err := cloudflareListPaginated[cloudflareDNSZoneTransferACL](ctx, api, fmt.Sprintf("/accounts/%s/secondary_dns/acls", accountID))
	if err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("list secondary DNS ACLs for account %q: %w", accountID, err)
	}
	for _, acl := range acls {
		if acl.ID == "" {
			continue
		}
		resource := terraformutils.NewResource(
			acl.ID,
			cloudflareResourceName(accountID, "dns_zone_transfers_acl", acl.Name, acl.ID),
			"cloudflare_dns_zone_transfers_acl",
			"cloudflare",
			map[string]string{"account_id": accountID},
			[]string{},
			map[string]interface{}{},
		)
		setCloudflareImportID(&resource, accountID+"/"+acl.ID)
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *SettingsGenerator) appendDNSZoneTransferPeerResources(ctx context.Context, api *cf.API, accountID string) error {
	peers, err := cloudflareListPaginated[cloudflareDNSZoneTransferPeer](ctx, api, fmt.Sprintf("/accounts/%s/secondary_dns/peers", accountID))
	if err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("list secondary DNS peers for account %q: %w", accountID, err)
	}
	for _, peer := range peers {
		if peer.ID == "" {
			continue
		}
		resource := terraformutils.NewResource(
			peer.ID,
			cloudflareResourceName(accountID, "dns_zone_transfers_peer", peer.Name, peer.ID),
			"cloudflare_dns_zone_transfers_peer",
			"cloudflare",
			map[string]string{"account_id": accountID},
			[]string{},
			map[string]interface{}{},
		)
		setCloudflareImportID(&resource, accountID+"/"+peer.ID)
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *SettingsGenerator) appendArgoSmartRoutingResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	setting, err := api.ArgoSmartRouting(ctx, zone.ID)
	if err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("get Argo Smart Routing setting for zone %q: %w", zone.ID, err)
	}
	if !cloudflareSettingIsOn(setting.Value) {
		return nil
	}
	g.appendZoneSingletonSettingResource(zone, "cloudflare_argo_smart_routing", "argo_smart_routing")
	return nil
}

func (g *SettingsGenerator) appendArgoTieredCachingResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	setting, err := api.ArgoTieredCaching(ctx, zone.ID)
	if err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("get Argo Tiered Caching setting for zone %q: %w", zone.ID, err)
	}
	if !cloudflareSettingIsOn(setting.Value) {
		return nil
	}
	g.appendZoneSingletonSettingResource(zone, "cloudflare_argo_tiered_caching", "argo_tiered_caching")
	return nil
}

func (g *SettingsGenerator) appendAuthenticatedOriginPullsSettingsResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	setting, err := api.GetPerZoneAuthenticatedOriginPullsStatus(ctx, zone.ID)
	if err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("get Authenticated Origin Pulls setting for zone %q: %w", zone.ID, err)
	}
	if !setting.Enabled {
		return nil
	}
	g.appendZoneSingletonSettingResource(zone, "cloudflare_authenticated_origin_pulls_settings", "authenticated_origin_pulls_settings")
	return nil
}

func (g *SettingsGenerator) appendCustomHostnameFallbackOriginResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	setting, err := api.CustomHostnameFallbackOrigin(ctx, zone.ID)
	if err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("get custom hostname fallback origin for zone %q: %w", zone.ID, err)
	}
	if setting.Origin == "" {
		return nil
	}
	g.appendZoneSingletonSettingResource(zone, "cloudflare_custom_hostname_fallback_origin", "custom_hostname_fallback_origin")
	return nil
}

func (g *SettingsGenerator) appendLeakedCredentialCheckResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	setting := cloudflareEnabledSetting{}
	if err := cloudflareReadRawSetting(ctx, api, fmt.Sprintf("/zones/%s/leaked-credential-checks", zone.ID), &setting); err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("get leaked credential check setting for zone %q: %w", zone.ID, err)
	}
	if !setting.Enabled {
		return nil
	}
	g.appendLeakedCredentialCheckZoneResource(zone)
	return nil
}

func (g *SettingsGenerator) appendLeakedCredentialCheckZoneResource(zone cf.Zone) {
	resource := cloudflareZoneSingletonSettingResourceWithAttributes(
		zone,
		"cloudflare_leaked_credential_check",
		"leaked_credential_check",
		map[string]string{"enabled": "true"},
	)
	resource.InstanceState.Meta[tfcompat.MetaKeyPreserveIDAfterRefresh] = true
	g.Resources = append(g.Resources, resource)
}

func (g *SettingsGenerator) appendLogpullRetentionResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	setting, err := api.GetLogpullRetentionFlag(ctx, zone.ID)
	if err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("get Logpull retention setting for zone %q: %w", zone.ID, err)
	}
	if setting == nil || !setting.Flag {
		return nil
	}
	g.appendZoneSingletonSettingResource(zone, "cloudflare_logpull_retention", "logpull_retention")
	return nil
}

func (g *SettingsGenerator) appendManagedTransformsResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	setting := cloudflareManagedTransformsSetting{}
	if err := cloudflareReadRawSetting(ctx, api, fmt.Sprintf("/zones/%s/managed_headers", zone.ID), &setting); err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("get Managed Transforms setting for zone %q: %w", zone.ID, err)
	}
	if !cloudflareManagedTransformsConfigured(setting) {
		return nil
	}
	attributes, additionalFields := cloudflareManagedTransformsState(setting)
	g.appendZoneSingletonSettingResourceWithAttributesAndAdditionalFields(
		zone,
		"cloudflare_managed_transforms",
		"managed_transforms",
		attributes,
		additionalFields,
	)
	return nil
}

func (g *SettingsGenerator) appendRegionalTieredCacheResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	setting, err := api.GetRegionalTieredCache(ctx, cf.ZoneIdentifier(zone.ID), cf.GetRegionalTieredCacheParams{})
	if err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("get Regional Tiered Cache setting for zone %q: %w", zone.ID, err)
	}
	if !cloudflareSettingIsOn(setting.Value) {
		return nil
	}
	g.appendZoneSingletonSettingResource(zone, "cloudflare_regional_tiered_cache", "regional_tiered_cache")
	return nil
}

func (g *SettingsGenerator) appendTieredCacheResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	setting := cloudflareValueSetting{}
	if err := cloudflareReadRawSetting(ctx, api, fmt.Sprintf("/zones/%s/cache/tiered_cache_smart_topology_enable", zone.ID), &setting); err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("get Smart Tiered Cache setting for zone %q: %w", zone.ID, err)
	}
	if !cloudflareSettingIsOn(setting.Value) {
		return nil
	}
	g.appendZoneSingletonSettingResource(zone, "cloudflare_tiered_cache", "tiered_cache")
	return nil
}

func (g *SettingsGenerator) appendTotalTLSResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	setting := cloudflareEnabledSetting{}
	if err := cloudflareReadRawSetting(ctx, api, fmt.Sprintf("/zones/%s/acm/total_tls", zone.ID), &setting); err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("get Total TLS setting for zone %q: %w", zone.ID, err)
	}
	if !setting.Enabled {
		return nil
	}
	g.appendZoneSingletonSettingResource(zone, "cloudflare_total_tls", "total_tls")
	return nil
}

func (g *SettingsGenerator) appendUniversalSSLSettingResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	setting, err := api.UniversalSSLSettingDetails(ctx, zone.ID)
	if err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("get Universal SSL setting for zone %q: %w", zone.ID, err)
	}
	if !cloudflareUniversalSSLSettingShouldImport(setting) {
		return nil
	}
	g.appendZoneSingletonSettingResource(zone, "cloudflare_universal_ssl_setting", "universal_ssl_setting")
	return nil
}

func (g *SettingsGenerator) appendURLNormalizationSettingsResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	setting, err := api.URLNormalizationSettings(ctx, cf.ZoneIdentifier(zone.ID))
	if err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("get URL Normalization setting for zone %q: %w", zone.ID, err)
	}
	if !cloudflareURLNormalizationSettingsShouldImport(setting) {
		return nil
	}
	g.appendZoneSingletonSettingResource(zone, "cloudflare_url_normalization_settings", "url_normalization_settings")
	return nil
}

func (g *SettingsGenerator) appendWaitingRoomSettingsResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	setting, err := api.GetWaitingRoomSettings(ctx, cf.ZoneIdentifier(zone.ID))
	if err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("get Waiting Room setting for zone %q: %w", zone.ID, err)
	}
	if !setting.SearchEngineCrawlerBypass {
		return nil
	}
	g.appendZoneSingletonSettingResource(zone, "cloudflare_waiting_room_settings", "waiting_room_settings")
	return nil
}

func (g *SettingsGenerator) appendZoneCacheReserveResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	setting, err := api.GetCacheReserve(ctx, cf.ZoneIdentifier(zone.ID), cf.GetCacheReserveParams{})
	if err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("get Cache Reserve setting for zone %q: %w", zone.ID, err)
	}
	if !cloudflareSettingIsOn(setting.Value) {
		return nil
	}
	g.appendZoneSingletonSettingResource(zone, "cloudflare_zone_cache_reserve", "zone_cache_reserve")
	return nil
}

func (g *SettingsGenerator) appendZoneCacheVariantsResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	setting, err := api.ZoneCacheVariants(ctx, zone.ID)
	if err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("get Cache Variants setting for zone %q: %w", zone.ID, err)
	}
	if !cloudflareZoneCacheVariantsConfigured(setting.Value) {
		return nil
	}
	g.appendZoneSingletonSettingResource(zone, "cloudflare_zone_cache_variants", "zone_cache_variants")
	return nil
}

func (g *SettingsGenerator) appendZoneHoldResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	setting, err := api.GetZoneHold(ctx, cf.ZoneIdentifier(zone.ID), cf.GetZoneHoldParams{})
	if err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("get Zone Hold setting for zone %q: %w", zone.ID, err)
	}
	if !cloudflareZoneHoldConfigured(setting) {
		return nil
	}
	g.appendZoneSingletonSettingResourceWithAttributes(
		zone,
		"cloudflare_zone_hold",
		"zone_hold",
		cloudflareZoneHoldAttributes(setting),
	)
	return nil
}

func (g *SettingsGenerator) appendDNSZoneTransfersIncomingResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	setting := cloudflareDNSZoneTransferConfig{}
	if err := cloudflareReadRawSetting(ctx, api, fmt.Sprintf("/zones/%s/secondary_dns/incoming", zone.ID), &setting); err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("get secondary DNS incoming transfer for zone %q: %w", zone.ID, err)
	}
	if !cloudflareDNSZoneTransferConfigured(setting) {
		return nil
	}
	g.appendZoneSingletonSettingResource(zone, "cloudflare_dns_zone_transfers_incoming", "dns_zone_transfers_incoming")
	return nil
}

func (g *SettingsGenerator) appendDNSZoneTransfersOutgoingResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	setting := cloudflareDNSZoneTransferConfig{}
	if err := cloudflareReadRawSetting(ctx, api, fmt.Sprintf("/zones/%s/secondary_dns/outgoing", zone.ID), &setting); err != nil {
		if cloudflareOptionalSettingsMissing(err) {
			return nil
		}
		return fmt.Errorf("get secondary DNS outgoing transfer for zone %q: %w", zone.ID, err)
	}
	if !cloudflareDNSZoneTransferConfigured(setting) {
		return nil
	}
	g.appendZoneSingletonSettingResource(zone, "cloudflare_dns_zone_transfers_outgoing", "dns_zone_transfers_outgoing")
	return nil
}

func (g *SettingsGenerator) appendZoneSingletonSettingResource(zone cf.Zone, resourceType, resourceNameSuffix string) {
	g.appendZoneSingletonSettingResourceWithAttributes(zone, resourceType, resourceNameSuffix, map[string]string{})
}

func (g *SettingsGenerator) appendZoneSingletonSettingResourceWithAttributes(zone cf.Zone, resourceType, resourceNameSuffix string, attributes map[string]string) {
	resource := cloudflareZoneSingletonSettingResourceWithAttributesAndAdditionalFields(zone, resourceType, resourceNameSuffix, attributes, map[string]interface{}{})
	g.Resources = append(g.Resources, resource)
}

func (g *SettingsGenerator) appendZoneSingletonSettingResourceWithAttributesAndAdditionalFields(
	zone cf.Zone,
	resourceType string,
	resourceNameSuffix string,
	attributes map[string]string,
	additionalFields map[string]interface{},
) {
	resource := cloudflareZoneSingletonSettingResourceWithAttributesAndAdditionalFields(zone, resourceType, resourceNameSuffix, attributes, additionalFields)
	g.Resources = append(g.Resources, resource)
}

func cloudflareZoneSingletonSettingResourceWithAttributes(zone cf.Zone, resourceType, resourceNameSuffix string, attributes map[string]string) terraformutils.Resource {
	return cloudflareZoneSingletonSettingResourceWithAttributesAndAdditionalFields(zone, resourceType, resourceNameSuffix, attributes, map[string]interface{}{})
}

func cloudflareZoneSingletonSettingResourceWithAttributesAndAdditionalFields(
	zone cf.Zone,
	resourceType string,
	resourceNameSuffix string,
	attributes map[string]string,
	additionalFields map[string]interface{},
) terraformutils.Resource {
	if attributes == nil {
		attributes = map[string]string{}
	}
	if additionalFields == nil {
		additionalFields = map[string]interface{}{}
	}
	attributes["zone_id"] = zone.ID

	resource := terraformutils.NewResource(
		zone.ID,
		cloudflareResourceName(zone.Name, resourceNameSuffix),
		resourceType,
		"cloudflare",
		attributes,
		[]string{},
		additionalFields,
	)
	setCloudflareImportID(&resource, zone.ID)
	return resource
}

func cloudflareListPaginated[T any](ctx context.Context, api *cf.API, endpoint string) ([]T, error) {
	resources := []T{}
	page, cursor := 1, ""
	for {
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("%s?%s", endpoint, cloudflarePaginationQuery(page, cursor)),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pageResources []T
		if err := json.Unmarshal(response.Result, &pageResources); err != nil {
			return nil, err
		}
		resources = append(resources, pageResources...)
		if !cloudflareAdvancePagination(response.ResultInfo, &page, &cursor) {
			break
		}
	}
	return resources, nil
}

func cloudflareReadRawSetting(ctx context.Context, api *cf.API, path string, target interface{}) error {
	response, err := api.Raw(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(response.Result, target)
}

func cloudflareOptionalSettingsMissing(err error) bool {
	return cloudflareNotFoundError(err)
}

func cloudflareSettingIsOn(value string) bool {
	return strings.EqualFold(value, "on")
}

func cloudflareManagedTransformsConfigured(setting cloudflareManagedTransformsSetting) bool {
	for _, header := range setting.ManagedRequestHeaders {
		if header.Enabled && header.ID != "" {
			return true
		}
	}
	for _, header := range setting.ManagedResponseHeaders {
		if header.Enabled && header.ID != "" {
			return true
		}
	}
	return false
}

func cloudflareManagedTransformsAttributes(setting cloudflareManagedTransformsSetting) map[string]string {
	attributes, _ := cloudflareManagedTransformsState(setting)
	return attributes
}

func cloudflareManagedTransformsState(setting cloudflareManagedTransformsSetting) (map[string]string, map[string]interface{}) {
	attributes := map[string]string{}
	additionalFields := map[string]interface{}{}
	if cloudflareAddEnabledManagedTransformAttributes(attributes, "managed_request_headers", setting.ManagedRequestHeaders) == 0 {
		additionalFields["managed_request_headers"] = []interface{}{}
	}
	if cloudflareAddEnabledManagedTransformAttributes(attributes, "managed_response_headers", setting.ManagedResponseHeaders) == 0 {
		additionalFields["managed_response_headers"] = []interface{}{}
	}
	return attributes, additionalFields
}

func cloudflareAddEnabledManagedTransformAttributes(attributes map[string]string, prefix string, headers []cloudflareManagedTransformHeader) int {
	index := 0
	for _, header := range headers {
		if !header.Enabled || header.ID == "" {
			continue
		}
		key := fmt.Sprintf("%s.%d", prefix, index)
		attributes[key+".id"] = header.ID
		attributes[key+".enabled"] = strconv.FormatBool(header.Enabled)
		index++
	}
	attributes[prefix+".#"] = strconv.Itoa(index)
	return index
}

func cloudflareUniversalSSLSettingShouldImport(setting cf.UniversalSSLSetting) bool {
	return !setting.Enabled
}

func cloudflareURLNormalizationSettingsShouldImport(setting cf.URLNormalizationSettings) bool {
	if setting.Scope == "" && setting.Type == "" {
		return false
	}
	return setting.Scope != "incoming" || setting.Type != "rfc3986"
}

func cloudflareZoneCacheVariantsConfigured(value cf.ZoneCacheVariantsValues) bool {
	return len(value.Avif) > 0 ||
		len(value.Bmp) > 0 ||
		len(value.Gif) > 0 ||
		len(value.Jpeg) > 0 ||
		len(value.Jpg) > 0 ||
		len(value.Jp2) > 0 ||
		len(value.Jpg2) > 0 ||
		len(value.Png) > 0 ||
		len(value.Tif) > 0 ||
		len(value.Tiff) > 0 ||
		len(value.Webp) > 0
}

func cloudflareZoneHoldConfigured(setting cf.ZoneHold) bool {
	return (setting.Hold != nil && *setting.Hold) ||
		(setting.IncludeSubdomains != nil && *setting.IncludeSubdomains) ||
		setting.HoldAfter != nil
}

func cloudflareZoneHoldAttributes(setting cf.ZoneHold) map[string]string {
	attributes := map[string]string{}
	if setting.IncludeSubdomains != nil {
		attributes["include_subdomains"] = strconv.FormatBool(*setting.IncludeSubdomains)
	}
	return attributes
}

func cloudflareDNSZoneTransferConfigured(setting cloudflareDNSZoneTransferConfig) bool {
	return len(setting.Peers) > 0
}
