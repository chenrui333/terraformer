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

type NetworkEdgeGenerator struct {
	CloudflareService
}

type cloudflareNetworkEdgeRawResource map[string]interface{}

type cloudflareNetworkEdgeDiscovery struct {
	name      string
	scope     string
	resources *[]terraformutils.Resource
	discover  func() error
}

func cloudflareNetworkEdgeString(resource cloudflareNetworkEdgeRawResource, keys ...string) string {
	for _, key := range keys {
		value, ok := resource[key].(string)
		if ok && value != "" {
			return value
		}
	}
	return ""
}

func cloudflareNetworkEdgeOptionalDiscoveryError(err error) bool {
	var notFoundErr *cf.NotFoundError
	if errors.As(err, &notFoundErr) {
		return cloudflareNetworkEdgeOptionalErrorMessage(notFoundErr.Error(), notFoundErr.ErrorMessages())
	}

	var requestErr *cf.RequestError
	if errors.As(err, &requestErr) {
		return cloudflareNetworkEdgeOptionalErrorMessage(requestErr.Error(), requestErr.ErrorMessages())
	}

	var authenticationErr *cf.AuthenticationError
	if errors.As(err, &authenticationErr) {
		return cloudflareNetworkEdgeOptionalErrorMessage(authenticationErr.Error(), authenticationErr.ErrorMessages())
	}

	var authorizationErr *cf.AuthorizationError
	if errors.As(err, &authorizationErr) {
		return cloudflareNetworkEdgeOptionalErrorMessage(authorizationErr.Error(), authorizationErr.ErrorMessages())
	}

	return false
}

func cloudflareNetworkEdgeOptionalErrorMessage(message string, errorMessages []string) bool {
	messages := append([]string{message}, errorMessages...)
	for _, msg := range messages {
		normalized := strings.ToLower(msg)
		for _, marker := range []string{
			"access denied",
			"feature is not available",
			"missing permission",
			"not authorized",
			"not configured",
			"not enabled",
			"not entitled",
			"permission denied",
			"requires a paid plan",
			"upgrade your plan",
		} {
			if strings.Contains(normalized, marker) {
				return true
			}
		}
	}
	return false
}

func runCloudflareNetworkEdgeDiscoveries(discoveries []cloudflareNetworkEdgeDiscovery) error {
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
			if cloudflareNetworkEdgeOptionalDiscoveryError(err) {
				log.Printf("Skipping Cloudflare network edge %s discovery for %s: %v", discovery.name, discovery.scope, err)
				continue
			}
			return fmt.Errorf("discover Cloudflare network edge %s for %s: %w", discovery.name, discovery.scope, err)
		}
	}
	return nil
}

func cloudflareNetworkEdgePagedPath(path string, page int, cursor string) string {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return fmt.Sprintf("%s%s%s", path, separator, cloudflarePaginationQuery(page, cursor))
}

func listCloudflareNetworkEdgeResources(
	ctx context.Context,
	api *cf.API,
	path string,
) ([]cloudflareNetworkEdgeRawResource, error) {
	var resources []cloudflareNetworkEdgeRawResource
	page, cursor := 1, ""
	for {
		response, err := api.Raw(ctx, http.MethodGet, cloudflareNetworkEdgePagedPath(path, page, cursor), nil, nil)
		if err != nil {
			return nil, err
		}
		if len(response.Result) == 0 || string(response.Result) == "null" {
			return resources, nil
		}

		var pageResources []cloudflareNetworkEdgeRawResource
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

func cloudflareZoneNetworkEdgeResource(
	zone cf.Zone,
	id string,
	resourceType string,
	resourceNamePrefix string,
	nameParts ...string,
) (terraformutils.Resource, bool) {
	if zone.ID == "" || id == "" {
		return terraformutils.Resource{}, false
	}
	parts := append([]string{zone.Name, resourceNamePrefix}, nameParts...)
	parts = append(parts, id)
	resource := terraformutils.NewResource(
		id,
		cloudflareResourceName(parts...),
		resourceType,
		"cloudflare",
		map[string]string{"zone_id": zone.ID},
		[]string{},
		map[string]interface{}{},
	)
	setCloudflareImportID(&resource, zone.ID+"/"+id)
	return resource, true
}

func cloudflareRegionalHostnameResource(zone cf.Zone, hostname string) (terraformutils.Resource, bool) {
	if zone.ID == "" || hostname == "" {
		return terraformutils.Resource{}, false
	}
	resource := terraformutils.NewResource(
		hostname,
		cloudflareResourceName(zone.Name, "regional_hostname", hostname),
		"cloudflare_regional_hostname",
		"cloudflare",
		map[string]string{
			"hostname": hostname,
			"zone_id":  zone.ID,
		},
		[]string{},
		map[string]interface{}{},
	)
	setCloudflareImportID(&resource, zone.ID+"/"+hostname)
	setCloudflarePreserveIDAfterRefresh(&resource)
	return resource, true
}

func cloudflareAccountNetworkEdgeResource(
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

func cloudflareMagicTransitSiteChildResource(
	accountID string,
	siteID string,
	id string,
	resourceType string,
	resourceNamePrefix string,
	nameParts ...string,
) (terraformutils.Resource, bool) {
	if accountID == "" || siteID == "" || id == "" {
		return terraformutils.Resource{}, false
	}
	parts := append([]string{accountID, resourceNamePrefix, siteID}, nameParts...)
	parts = append(parts, id)
	resource := terraformutils.NewResource(
		id,
		cloudflareResourceName(parts...),
		resourceType,
		"cloudflare",
		map[string]string{
			"account_id": accountID,
			"site_id":    siteID,
		},
		[]string{},
		map[string]interface{}{},
	)
	setCloudflareImportID(&resource, accountID+"/"+siteID+"/"+id)
	return resource, true
}

func cloudflareAddressMapImportable(addressMap cloudflareNetworkEdgeRawResource) bool {
	for _, key := range []string{"can_delete", "can_modify_ips"} {
		if value, ok := addressMap[key].(bool); ok && !value {
			return false
		}
	}
	return true
}

func (g *NetworkEdgeGenerator) appendSpectrumApplicationResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	applications, err := listCloudflareNetworkEdgeResources(ctx, api, fmt.Sprintf("/zones/%s/spectrum/apps", zone.ID))
	if err != nil {
		return err
	}
	for _, application := range applications {
		id := cloudflareNetworkEdgeString(application, "id")
		dnsName := ""
		if dns, ok := application["dns"].(map[string]interface{}); ok {
			dnsName = cloudflareNetworkEdgeString(cloudflareNetworkEdgeRawResource(dns), "name")
		}
		resource, ok := cloudflareZoneNetworkEdgeResource(
			zone,
			id,
			"cloudflare_spectrum_application",
			"spectrum_application",
			dnsName,
			cloudflareNetworkEdgeString(application, "protocol"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *NetworkEdgeGenerator) appendRegionalHostnameResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	hostnames, err := listCloudflareNetworkEdgeResources(ctx, api, fmt.Sprintf("/zones/%s/addressing/regional_hostnames", zone.ID))
	if err != nil {
		return err
	}
	for _, hostname := range hostnames {
		resource, ok := cloudflareRegionalHostnameResource(zone, cloudflareNetworkEdgeString(hostname, "hostname"))
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *NetworkEdgeGenerator) appendWeb3HostnameResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	hostnames, err := listCloudflareNetworkEdgeResources(ctx, api, fmt.Sprintf("/zones/%s/web3/hostnames", zone.ID))
	if err != nil {
		return err
	}
	for _, hostname := range hostnames {
		id := cloudflareNetworkEdgeString(hostname, "id")
		resource, ok := cloudflareZoneNetworkEdgeResource(
			zone,
			id,
			"cloudflare_web3_hostname",
			"web3_hostname",
			cloudflareNetworkEdgeString(hostname, "name"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *NetworkEdgeGenerator) appendAddressMapResources(ctx context.Context, api *cf.API, accountID string) error {
	addressMaps, err := listCloudflareNetworkEdgeResources(ctx, api, fmt.Sprintf("/accounts/%s/addressing/address_maps", accountID))
	if err != nil {
		return err
	}
	for _, addressMap := range addressMaps {
		if !cloudflareAddressMapImportable(addressMap) {
			continue
		}
		resource, ok := cloudflareAccountNetworkEdgeResource(
			accountID,
			cloudflareNetworkEdgeString(addressMap, "id"),
			"cloudflare_address_map",
			"address_map",
			cloudflareNetworkEdgeString(addressMap, "description", "default_sni"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *NetworkEdgeGenerator) appendMagicNetworkMonitoringRuleResources(ctx context.Context, api *cf.API, accountID string) error {
	rules, err := listCloudflareNetworkEdgeResources(ctx, api, fmt.Sprintf("/accounts/%s/mnm/rules", accountID))
	if err != nil {
		return err
	}
	for _, rule := range rules {
		resource, ok := cloudflareAccountNetworkEdgeResource(
			accountID,
			cloudflareNetworkEdgeString(rule, "id"),
			"cloudflare_magic_network_monitoring_rule",
			"magic_network_monitoring_rule",
			cloudflareNetworkEdgeString(rule, "name"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *NetworkEdgeGenerator) appendMagicTransitSiteACLResources(
	ctx context.Context,
	api *cf.API,
	accountID string,
	siteID string,
) error {
	acls, err := listCloudflareNetworkEdgeResources(ctx, api, fmt.Sprintf("/accounts/%s/magic/sites/%s/acls", accountID, siteID))
	if err != nil {
		return err
	}
	for _, acl := range acls {
		resource, ok := cloudflareMagicTransitSiteChildResource(
			accountID,
			siteID,
			cloudflareNetworkEdgeString(acl, "id"),
			"cloudflare_magic_transit_site_acl",
			"magic_transit_site_acl",
			cloudflareNetworkEdgeString(acl, "name"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *NetworkEdgeGenerator) appendMagicTransitSiteLANResources(
	ctx context.Context,
	api *cf.API,
	accountID string,
	siteID string,
) error {
	lans, err := listCloudflareNetworkEdgeResources(ctx, api, fmt.Sprintf("/accounts/%s/magic/sites/%s/lans", accountID, siteID))
	if err != nil {
		return err
	}
	for _, lan := range lans {
		resource, ok := cloudflareMagicTransitSiteChildResource(
			accountID,
			siteID,
			cloudflareNetworkEdgeString(lan, "id"),
			"cloudflare_magic_transit_site_lan",
			"magic_transit_site_lan",
			cloudflareNetworkEdgeString(lan, "name"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *NetworkEdgeGenerator) appendMagicTransitSiteWANResources(
	ctx context.Context,
	api *cf.API,
	accountID string,
	siteID string,
) error {
	wans, err := listCloudflareNetworkEdgeResources(ctx, api, fmt.Sprintf("/accounts/%s/magic/sites/%s/wans", accountID, siteID))
	if err != nil {
		return err
	}
	for _, wan := range wans {
		resource, ok := cloudflareMagicTransitSiteChildResource(
			accountID,
			siteID,
			cloudflareNetworkEdgeString(wan, "id"),
			"cloudflare_magic_transit_site_wan",
			"magic_transit_site_wan",
			cloudflareNetworkEdgeString(wan, "name"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *NetworkEdgeGenerator) appendMagicTransitSiteResources(ctx context.Context, api *cf.API, accountID string) error {
	sites, err := listCloudflareNetworkEdgeResources(ctx, api, fmt.Sprintf("/accounts/%s/magic/sites", accountID))
	if err != nil {
		return err
	}
	for _, site := range sites {
		siteID := cloudflareNetworkEdgeString(site, "id")
		resource, ok := cloudflareAccountNetworkEdgeResource(
			accountID,
			siteID,
			"cloudflare_magic_transit_site",
			"magic_transit_site",
			cloudflareNetworkEdgeString(site, "name"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
		if siteID == "" {
			continue
		}
		if err := runCloudflareNetworkEdgeDiscoveries([]cloudflareNetworkEdgeDiscovery{
			{
				name:      "Magic Transit site ACLs",
				scope:     siteID,
				resources: &g.Resources,
				discover: func() error {
					return g.appendMagicTransitSiteACLResources(ctx, api, accountID, siteID)
				},
			},
			{
				name:      "Magic Transit site LANs",
				scope:     siteID,
				resources: &g.Resources,
				discover: func() error {
					return g.appendMagicTransitSiteLANResources(ctx, api, accountID, siteID)
				},
			},
			{
				name:      "Magic Transit site WANs",
				scope:     siteID,
				resources: &g.Resources,
				discover: func() error {
					return g.appendMagicTransitSiteWANResources(ctx, api, accountID, siteID)
				},
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (g *NetworkEdgeGenerator) appendZoneNetworkEdgeResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	return runCloudflareNetworkEdgeDiscoveries([]cloudflareNetworkEdgeDiscovery{
		{
			name:      "Spectrum applications",
			scope:     zone.ID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendSpectrumApplicationResources(ctx, api, zone)
			},
		},
		{
			name:      "regional hostnames",
			scope:     zone.ID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendRegionalHostnameResources(ctx, api, zone)
			},
		},
		{
			name:      "Web3 hostnames",
			scope:     zone.ID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendWeb3HostnameResources(ctx, api, zone)
			},
		},
	})
}

func (g *NetworkEdgeGenerator) appendAccountNetworkEdgeResources(ctx context.Context, api *cf.API, accountID string) error {
	return runCloudflareNetworkEdgeDiscoveries([]cloudflareNetworkEdgeDiscovery{
		{
			name:      "address maps",
			scope:     accountID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendAddressMapResources(ctx, api, accountID)
			},
		},
		{
			name:      "Magic Network Monitoring rules",
			scope:     accountID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendMagicNetworkMonitoringRuleResources(ctx, api, accountID)
			},
		},
		{
			name:      "Magic Transit sites",
			scope:     accountID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendMagicTransitSiteResources(ctx, api, accountID)
			},
		},
	})
}

func (g *NetworkEdgeGenerator) appendNetworkEdgeResources(
	ctx context.Context,
	api *cf.API,
	zones []cf.Zone,
	zonesErr error,
	accountID string,
) error {
	if zonesErr != nil {
		if accountID == "" || !cloudflareNetworkEdgeOptionalDiscoveryError(zonesErr) {
			return fmt.Errorf("discover Cloudflare network edge zones: %w", zonesErr)
		}
		log.Printf("Skipping Cloudflare network edge zone discovery: %v", zonesErr)
	} else {
		for _, zone := range zones {
			if err := g.appendZoneNetworkEdgeResources(ctx, api, zone); err != nil {
				return err
			}
		}
	}

	if accountID == "" {
		return nil
	}
	return g.appendAccountNetworkEdgeResources(ctx, api, accountID)
}

func (g *NetworkEdgeGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}

	zones, zonesErr := cloudflareZones(ctx, api)
	return g.appendNetworkEdgeResources(ctx, api, zones, zonesErr, g.accountID())
}
