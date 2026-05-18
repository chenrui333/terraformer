// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type ZeroTrustDeviceDLPGenerator struct {
	CloudflareService
}

type zeroTrustDeviceDLPRawResource map[string]interface{}

type zeroTrustDeviceDLPDiscovery struct {
	name      string
	account   string
	resources *[]terraformutils.Resource
	discover  func() error
}

func zeroTrustDeviceDLPString(resource zeroTrustDeviceDLPRawResource, keys ...string) string {
	for _, key := range keys {
		value, ok := resource[key].(string)
		if ok && value != "" {
			return value
		}
	}
	return ""
}

func zeroTrustDeviceDLPBool(resource zeroTrustDeviceDLPRawResource, key string) bool {
	value, ok := resource[key].(bool)
	return ok && value
}

func zeroTrustDeviceDLPHasListEntries(resource zeroTrustDeviceDLPRawResource, key string) bool {
	entries, ok := resource[key].([]interface{})
	return ok && len(entries) > 0
}

func zeroTrustDeviceDLPResourceType(resource zeroTrustDeviceDLPRawResource) string {
	return strings.ToLower(zeroTrustDeviceDLPString(resource, "type"))
}

func runZeroTrustDeviceDLPDiscoveries(discoveries []zeroTrustDeviceDLPDiscovery) error {
	for _, discovery := range discoveries {
		if discovery.discover == nil {
			continue
		}
		resourceCount := 0
		if discovery.resources != nil {
			resourceCount = len(*discovery.resources)
		}
		if err := discovery.discover(); err != nil {
			if discovery.resources != nil && resourceCount <= len(*discovery.resources) {
				*discovery.resources = (*discovery.resources)[:resourceCount]
			}
			if zeroTrustDeviceDLPOptionalDiscoveryError(err) {
				log.Printf("Skipping Cloudflare Zero Trust device/DLP %s discovery for %s: %v", discovery.name, discovery.account, err)
				continue
			}
			return fmt.Errorf("discover Cloudflare Zero Trust device/DLP %s for %s: %w", discovery.name, discovery.account, err)
		}
	}
	return nil
}

func zeroTrustDeviceDLPOptionalDiscoveryError(err error) bool {
	return zeroTrustGatewayOptionalUnavailableError(err) || cloudflareMediaPlatformOptionalDiscoveryError(err)
}

func zeroTrustDeviceDLPPagedPath(path string, page int, cursor string) string {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return fmt.Sprintf("%s%s%s", path, separator, cloudflarePaginationQuery(page, cursor))
}

func listZeroTrustDeviceDLPResources(ctx context.Context, api *cf.API, path string) ([]zeroTrustDeviceDLPRawResource, error) {
	var resources []zeroTrustDeviceDLPRawResource
	page, cursor := 1, ""
	for {
		response, err := api.Raw(ctx, http.MethodGet, zeroTrustDeviceDLPPagedPath(path, page, cursor), nil, nil)
		if err != nil {
			return nil, err
		}
		if len(response.Result) == 0 || string(response.Result) == "null" {
			return resources, nil
		}

		pageResources, err := zeroTrustDeviceDLPPagedResources(response.Result)
		if err != nil {
			return nil, err
		}
		resources = append(resources, pageResources...)
		if !cloudflareAdvanceMediaPlatformPagination(response.ResultInfo, &page, &cursor, len(pageResources)) {
			break
		}
	}
	return resources, nil
}

func zeroTrustDeviceDLPPagedResources(result json.RawMessage) ([]zeroTrustDeviceDLPRawResource, error) {
	var pageResources []zeroTrustDeviceDLPRawResource
	if err := json.Unmarshal(result, &pageResources); err == nil {
		return pageResources, nil
	}

	var objectResult map[string][]zeroTrustDeviceDLPRawResource
	if err := json.Unmarshal(result, &objectResult); err != nil {
		return nil, err
	}
	for _, key := range []string{"dex_tests", "tests", "rules", "result"} {
		if resources, ok := objectResult[key]; ok {
			return resources, nil
		}
	}
	return nil, fmt.Errorf("unsupported Zero Trust device/DLP list response shape")
}

func readZeroTrustDeviceDLPResource(ctx context.Context, api *cf.API, path string) (zeroTrustDeviceDLPRawResource, bool, error) {
	response, err := api.Raw(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, false, err
	}
	if len(response.Result) == 0 || string(response.Result) == "null" {
		return nil, false, nil
	}

	var resource zeroTrustDeviceDLPRawResource
	if err := json.Unmarshal(response.Result, &resource); err != nil {
		return nil, false, err
	}
	return resource, true, nil
}

func zeroTrustDeviceDLPAccountResource(
	accountID string,
	id string,
	resourceType string,
	resourceNamePrefix string,
	nameParts ...string,
) (terraformutils.Resource, bool) {
	if accountID == "" || id == "" {
		return terraformutils.Resource{}, false
	}
	parts := append([]string{accountID, resourceNamePrefix}, nameParts...)
	parts = append(parts, id)
	resource := terraformutils.NewResource(
		id,
		cloudflareResourceName(parts...),
		resourceType,
		"cloudflare",
		map[string]string{"account_id": accountID},
		[]string{},
		map[string]interface{}{},
	)
	setCloudflareImportID(&resource, accountID+"/"+id)
	return resource, true
}

func zeroTrustDeviceDLPSingletonResource(accountID string, resourceType string, resourceNamePrefix string) (terraformutils.Resource, bool) {
	if accountID == "" {
		return terraformutils.Resource{}, false
	}
	resource := terraformutils.NewResource(
		accountID,
		cloudflareResourceName(accountID, resourceNamePrefix),
		resourceType,
		"cloudflare",
		map[string]string{"account_id": accountID},
		[]string{},
		map[string]interface{}{},
	)
	setCloudflareImportID(&resource, accountID)
	return resource, true
}

func zeroTrustDeviceDLPName(resource zeroTrustDeviceDLPRawResource, fallbackKeys ...string) []string {
	keys := append([]string{"name", "description"}, fallbackKeys...)
	for _, key := range keys {
		value := zeroTrustDeviceDLPString(resource, key)
		if value != "" {
			return []string{value}
		}
	}
	return nil
}

func zeroTrustDeviceDLPAppendAccountResource(
	resources *[]terraformutils.Resource,
	accountID string,
	resource zeroTrustDeviceDLPRawResource,
	idKeys []string,
	resourceType string,
	resourceNamePrefix string,
	nameFallbackKeys ...string,
) {
	id := zeroTrustDeviceDLPString(resource, idKeys...)
	item, ok := zeroTrustDeviceDLPAccountResource(
		accountID,
		id,
		resourceType,
		resourceNamePrefix,
		zeroTrustDeviceDLPName(resource, nameFallbackKeys...)...,
	)
	if !ok {
		return
	}
	*resources = append(*resources, item)
}

func (g *ZeroTrustDeviceDLPGenerator) appendDeviceDefaultProfileResources(ctx context.Context, api *cf.API, accountID string) error {
	profile, exists, err := readZeroTrustDeviceDLPResource(ctx, api, fmt.Sprintf("/accounts/%s/devices/policy", accountID))
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	profileResource, ok := zeroTrustDeviceDLPSingletonResource(
		accountID,
		"cloudflare_zero_trust_device_default_profile",
		"device_default_profile",
	)
	if ok {
		g.Resources = append(g.Resources, profileResource)
	}
	if zeroTrustDeviceDLPHasListEntries(profile, "fallback_domains") {
		fallbackResource, ok := zeroTrustDeviceDLPSingletonResource(
			accountID,
			"cloudflare_zero_trust_device_default_profile_local_domain_fallback",
			"device_default_profile_local_domain_fallback",
		)
		if ok {
			g.Resources = append(g.Resources, fallbackResource)
		}
	}
	return nil
}

func (g *ZeroTrustDeviceDLPGenerator) appendDeviceCustomProfileResources(ctx context.Context, api *cf.API, accountID string) error {
	profiles, err := listZeroTrustDeviceDLPResources(ctx, api, fmt.Sprintf("/accounts/%s/devices/policies", accountID))
	if err != nil {
		return err
	}
	for _, profile := range profiles {
		if zeroTrustDeviceDLPBool(profile, "default") {
			continue
		}
		id := zeroTrustDeviceDLPString(profile, "policy_id", "id")
		if id == "" {
			continue
		}
		zeroTrustDeviceDLPAppendAccountResource(
			&g.Resources,
			accountID,
			profile,
			[]string{"policy_id", "id"},
			"cloudflare_zero_trust_device_custom_profile",
			"device_custom_profile",
			"policy_id",
		)
		if zeroTrustDeviceDLPHasListEntries(profile, "fallback_domains") {
			resource, ok := zeroTrustDeviceDLPAccountResource(
				accountID,
				id,
				"cloudflare_zero_trust_device_custom_profile_local_domain_fallback",
				"device_custom_profile_local_domain_fallback",
				zeroTrustDeviceDLPName(profile, "policy_id")...,
			)
			if ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *ZeroTrustDeviceDLPGenerator) appendDeviceManagedNetworkResources(ctx context.Context, api *cf.API, accountID string) error {
	networks, err := listZeroTrustDeviceDLPResources(ctx, api, fmt.Sprintf("/accounts/%s/devices/networks", accountID))
	if err != nil {
		return err
	}
	for _, network := range networks {
		zeroTrustDeviceDLPAppendAccountResource(
			&g.Resources,
			accountID,
			network,
			[]string{"network_id", "id"},
			"cloudflare_zero_trust_device_managed_networks",
			"device_managed_network",
			"network_id",
		)
	}
	return nil
}

func (g *ZeroTrustDeviceDLPGenerator) appendDeviceIPProfileResources(ctx context.Context, api *cf.API, accountID string) error {
	profiles, err := listZeroTrustDeviceDLPResources(ctx, api, fmt.Sprintf("/accounts/%s/devices/ip-profiles", accountID))
	if err != nil {
		return err
	}
	for _, profile := range profiles {
		zeroTrustDeviceDLPAppendAccountResource(
			&g.Resources,
			accountID,
			profile,
			[]string{"id"},
			"cloudflare_zero_trust_device_ip_profile",
			"device_ip_profile",
		)
	}
	return nil
}

func (g *ZeroTrustDeviceDLPGenerator) appendDEXRuleResources(ctx context.Context, api *cf.API, accountID string) error {
	rules, err := listZeroTrustDeviceDLPResources(ctx, api, fmt.Sprintf("/accounts/%s/dex/rules", accountID))
	if err != nil {
		return err
	}
	for _, rule := range rules {
		zeroTrustDeviceDLPAppendAccountResource(
			&g.Resources,
			accountID,
			rule,
			[]string{"id"},
			"cloudflare_zero_trust_dex_rule",
			"dex_rule",
		)
	}
	return nil
}

func (g *ZeroTrustDeviceDLPGenerator) appendDEXTestResources(ctx context.Context, api *cf.API, accountID string) error {
	tests, err := listZeroTrustDeviceDLPResources(ctx, api, fmt.Sprintf("/accounts/%s/dex/devices/dex_tests", accountID))
	if err != nil {
		return err
	}
	for _, test := range tests {
		zeroTrustDeviceDLPAppendAccountResource(
			&g.Resources,
			accountID,
			test,
			[]string{"test_id", "id"},
			"cloudflare_zero_trust_dex_test",
			"dex_test",
			"test_id",
		)
	}
	return nil
}

func (g *ZeroTrustDeviceDLPGenerator) appendDLPCustomProfileResources(ctx context.Context, api *cf.API, accountID string) error {
	profiles, err := listZeroTrustDeviceDLPResources(ctx, api, fmt.Sprintf("/accounts/%s/dlp/profiles", accountID))
	if err != nil {
		return err
	}
	for _, profile := range profiles {
		if zeroTrustDeviceDLPResourceType(profile) != "custom" {
			continue
		}
		zeroTrustDeviceDLPAppendAccountResource(
			&g.Resources,
			accountID,
			profile,
			[]string{"id"},
			"cloudflare_zero_trust_dlp_custom_profile",
			"dlp_custom_profile",
		)
	}
	return nil
}

func (g *ZeroTrustDeviceDLPGenerator) appendDLPCustomEntryResources(ctx context.Context, api *cf.API, accountID string) error {
	entries, err := listZeroTrustDeviceDLPResources(ctx, api, fmt.Sprintf("/accounts/%s/dlp/entries", accountID))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if zeroTrustDeviceDLPResourceType(entry) != "custom" {
			continue
		}
		zeroTrustDeviceDLPAppendAccountResource(
			&g.Resources,
			accountID,
			entry,
			[]string{"id"},
			"cloudflare_zero_trust_dlp_custom_entry",
			"dlp_custom_entry",
		)
	}
	return nil
}

func (g *ZeroTrustDeviceDLPGenerator) appendDLPSettingsResource(ctx context.Context, api *cf.API, accountID string) error {
	_, exists, err := readZeroTrustDeviceDLPResource(ctx, api, fmt.Sprintf("/accounts/%s/dlp/settings", accountID))
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	resource, ok := zeroTrustDeviceDLPSingletonResource(
		accountID,
		"cloudflare_zero_trust_dlp_settings",
		"dlp_settings",
	)
	if ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *ZeroTrustDeviceDLPGenerator) appendZeroTrustDeviceDLPResources(ctx context.Context, api *cf.API, accountID string) error {
	return runZeroTrustDeviceDLPDiscoveries([]zeroTrustDeviceDLPDiscovery{
		{
			name:      "device default profile",
			account:   accountID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendDeviceDefaultProfileResources(ctx, api, accountID)
			},
		},
		{
			name:      "device custom profiles",
			account:   accountID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendDeviceCustomProfileResources(ctx, api, accountID)
			},
		},
		{
			name:      "device managed networks",
			account:   accountID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendDeviceManagedNetworkResources(ctx, api, accountID)
			},
		},
		{
			name:      "device IP profiles",
			account:   accountID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendDeviceIPProfileResources(ctx, api, accountID)
			},
		},
		{
			name:      "DEX rules",
			account:   accountID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendDEXRuleResources(ctx, api, accountID)
			},
		},
		{
			name:      "DEX tests",
			account:   accountID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendDEXTestResources(ctx, api, accountID)
			},
		},
		{
			name:      "DLP custom profiles",
			account:   accountID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendDLPCustomProfileResources(ctx, api, accountID)
			},
		},
		{
			name:      "DLP custom entries",
			account:   accountID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendDLPCustomEntryResources(ctx, api, accountID)
			},
		},
		{
			name:      "DLP settings",
			account:   accountID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendDLPSettingsResource(ctx, api, accountID)
			},
		},
	})
}

func (g *ZeroTrustDeviceDLPGenerator) InitResources() error {
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	accountID := g.accountID()
	if accountID == "" {
		return errors.New("set CLOUDFLARE_ACCOUNT_ID env var")
	}
	return g.appendZeroTrustDeviceDLPResources(context.Background(), api, accountID)
}
