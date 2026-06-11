// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	cf "github.com/chenrui333/terraformer/providers/cloudflare/internal/cloudflarev7"
	"github.com/chenrui333/terraformer/terraformutils"
)

type ZeroTrustGatewayGenerator struct {
	CloudflareService
}

type zeroTrustGatewayRawResource map[string]interface{}

type zeroTrustGatewayDiscovery struct {
	name     string
	account  string
	discover func() error
}

func zeroTrustGatewayString(resource zeroTrustGatewayRawResource, key string) string {
	value, ok := resource[key].(string)
	if !ok {
		return ""
	}
	return value
}

func zeroTrustGatewayBool(resource zeroTrustGatewayRawResource, key string) bool {
	value, ok := resource[key].(bool)
	return ok && value
}

func zeroTrustGatewayNonEmptyStringPointer(resource zeroTrustGatewayRawResource, key string) bool {
	value, ok := resource[key].(string)
	return ok && value != ""
}

func zeroTrustGatewayAccountResource(accountID, id, resourceName, resourceType string) terraformutils.Resource {
	resource := terraformutils.NewResource(
		id,
		resourceName,
		resourceType,
		"cloudflare",
		map[string]string{"account_id": accountID},
		[]string{},
		map[string]interface{}{},
	)
	setCloudflareImportID(&resource, accountID+"/"+id)
	return resource
}

func zeroTrustGatewaySingletonResource(accountID, resourceName, resourceType string) terraformutils.Resource {
	resource := terraformutils.NewResource(
		accountID,
		resourceName,
		resourceType,
		"cloudflare",
		map[string]string{"account_id": accountID},
		[]string{},
		map[string]interface{}{},
	)
	setCloudflareImportID(&resource, accountID)
	return resource
}

func zeroTrustGatewayOptionalUnavailableError(err error) bool {
	var notFoundErr *cf.NotFoundError
	if errors.As(err, &notFoundErr) {
		return zeroTrustGatewayUnavailableMessage(notFoundErr.Error(), notFoundErr.ErrorMessages())
	}

	var requestErr *cf.RequestError
	if errors.As(err, &requestErr) {
		return zeroTrustGatewayUnavailableMessage(requestErr.Error(), requestErr.ErrorMessages())
	}

	return false
}

func runZeroTrustGatewayDiscoveries(discoveries []zeroTrustGatewayDiscovery) error {
	for _, discovery := range discoveries {
		if discovery.discover == nil {
			continue
		}
		if err := discovery.discover(); err != nil {
			return fmt.Errorf("discover Cloudflare Zero Trust Gateway %s for %s: %w", discovery.name, discovery.account, err)
		}
	}
	return nil
}

func zeroTrustGatewayUnavailableMessage(message string, errorMessages []string) bool {
	messages := append([]string{message}, errorMessages...)
	for _, msg := range messages {
		normalized := strings.ToLower(msg)
		for _, marker := range []string{
			"not enabled",
			"not configured",
			"feature is not available",
			"zero trust account",
		} {
			if strings.Contains(normalized, marker) {
				return true
			}
		}
	}
	return false
}

func listZeroTrustGatewayResources(ctx context.Context, api *cf.API, path string) ([]zeroTrustGatewayRawResource, error) {
	var resources []zeroTrustGatewayRawResource
	page, cursor := 1, ""
	for {
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("%s?%s", path, cloudflarePaginationQuery(page, cursor)),
			nil,
			nil,
		)
		if err != nil {
			if zeroTrustGatewayOptionalUnavailableError(err) {
				return resources, nil
			}
			return nil, err
		}

		var pageResources []zeroTrustGatewayRawResource
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

func zeroTrustGatewaySingletonExists(ctx context.Context, api *cf.API, path string) (bool, error) {
	_, err := api.Raw(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		if zeroTrustGatewayOptionalUnavailableError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func appendZeroTrustGatewayNamedResources(
	resources *[]terraformutils.Resource,
	accountID string,
	resourceType string,
	resourceNamePrefix string,
	items []zeroTrustGatewayRawResource,
) {
	for _, item := range items {
		id := zeroTrustGatewayString(item, "id")
		if id == "" {
			continue
		}
		*resources = append(*resources, zeroTrustGatewayAccountResource(
			accountID,
			id,
			cloudflareResourceName(accountID, resourceNamePrefix, zeroTrustGatewayString(item, "name"), id),
			resourceType,
		))
	}
}

func (g *ZeroTrustGatewayGenerator) appendDNSLocationResources(ctx context.Context, api *cf.API, accountID string) error {
	locations, err := listZeroTrustGatewayResources(ctx, api, fmt.Sprintf("/accounts/%s/gateway/locations", accountID))
	if err != nil {
		return err
	}
	appendZeroTrustGatewayNamedResources(
		&g.Resources,
		accountID,
		"cloudflare_zero_trust_dns_location",
		"dns_location",
		locations,
	)
	return nil
}

func (g *ZeroTrustGatewayGenerator) appendGatewayCertificateResources(ctx context.Context, api *cf.API, accountID string) error {
	certificates, err := listZeroTrustGatewayResources(ctx, api, fmt.Sprintf("/accounts/%s/gateway/certificates", accountID))
	if err != nil {
		return err
	}
	for _, certificate := range certificates {
		id := zeroTrustGatewayString(certificate, "id")
		if id == "" || zeroTrustGatewayString(certificate, "binding_status") == "pending_deletion" {
			continue
		}
		g.Resources = append(g.Resources, zeroTrustGatewayAccountResource(
			accountID,
			id,
			cloudflareResourceName(
				accountID,
				"gateway_certificate",
				zeroTrustGatewayString(certificate, "fingerprint"),
				zeroTrustGatewayString(certificate, "issuer_org"),
				zeroTrustGatewayString(certificate, "type"),
				id,
			),
			"cloudflare_zero_trust_gateway_certificate",
		))
	}
	return nil
}

func (g *ZeroTrustGatewayGenerator) appendGatewayLoggingResource(ctx context.Context, api *cf.API, accountID string) error {
	exists, err := zeroTrustGatewaySingletonExists(ctx, api, fmt.Sprintf("/accounts/%s/gateway/logging", accountID))
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	g.Resources = append(g.Resources, zeroTrustGatewaySingletonResource(
		accountID,
		cloudflareResourceName(accountID, "gateway_logging"),
		"cloudflare_zero_trust_gateway_logging",
	))
	return nil
}

func (g *ZeroTrustGatewayGenerator) appendGatewayPacfileResources(ctx context.Context, api *cf.API, accountID string) error {
	pacfiles, err := listZeroTrustGatewayResources(ctx, api, fmt.Sprintf("/accounts/%s/gateway/pacfiles", accountID))
	if err != nil {
		return err
	}
	appendZeroTrustGatewayNamedResources(
		&g.Resources,
		accountID,
		"cloudflare_zero_trust_gateway_pacfile",
		"gateway_pacfile",
		pacfiles,
	)
	return nil
}

func zeroTrustGatewayPolicyImportable(policy zeroTrustGatewayRawResource) bool {
	if zeroTrustGatewayString(policy, "id") == "" {
		return false
	}
	if zeroTrustGatewayNonEmptyStringPointer(policy, "deleted_at") {
		return false
	}
	return !zeroTrustGatewayBool(policy, "read_only")
}

func (g *ZeroTrustGatewayGenerator) appendGatewayPolicyResources(ctx context.Context, api *cf.API, accountID string) error {
	policies, err := listZeroTrustGatewayResources(ctx, api, fmt.Sprintf("/accounts/%s/gateway/rules", accountID))
	if err != nil {
		return err
	}
	for _, policy := range policies {
		if !zeroTrustGatewayPolicyImportable(policy) {
			continue
		}
		id := zeroTrustGatewayString(policy, "id")
		g.Resources = append(g.Resources, zeroTrustGatewayAccountResource(
			accountID,
			id,
			cloudflareResourceName(accountID, "gateway_policy", zeroTrustGatewayString(policy, "name"), id),
			"cloudflare_zero_trust_gateway_policy",
		))
	}
	return nil
}

func (g *ZeroTrustGatewayGenerator) appendGatewayProxyEndpointResources(ctx context.Context, api *cf.API, accountID string) error {
	endpoints, err := listZeroTrustGatewayResources(ctx, api, fmt.Sprintf("/accounts/%s/gateway/proxy_endpoints", accountID))
	if err != nil {
		return err
	}
	appendZeroTrustGatewayNamedResources(
		&g.Resources,
		accountID,
		"cloudflare_zero_trust_gateway_proxy_endpoint",
		"gateway_proxy_endpoint",
		endpoints,
	)
	return nil
}

func (g *ZeroTrustGatewayGenerator) appendGatewaySettingsResource(ctx context.Context, api *cf.API, accountID string) error {
	exists, err := zeroTrustGatewaySingletonExists(ctx, api, fmt.Sprintf("/accounts/%s/gateway/configuration", accountID))
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	g.Resources = append(g.Resources, zeroTrustGatewaySingletonResource(
		accountID,
		cloudflareResourceName(accountID, "gateway_settings"),
		"cloudflare_zero_trust_gateway_settings",
	))
	return nil
}

func (g *ZeroTrustGatewayGenerator) appendZeroTrustListResources(ctx context.Context, api *cf.API, accountID string) error {
	lists, err := listZeroTrustGatewayResources(ctx, api, fmt.Sprintf("/accounts/%s/gateway/lists", accountID))
	if err != nil {
		return err
	}
	appendZeroTrustGatewayNamedResources(
		&g.Resources,
		accountID,
		"cloudflare_zero_trust_list",
		"zero_trust_list",
		lists,
	)
	return nil
}

func zeroTrustNetworkHostnameRouteImportable(route zeroTrustGatewayRawResource) bool {
	if zeroTrustGatewayString(route, "id") == "" {
		return false
	}
	return !zeroTrustGatewayNonEmptyStringPointer(route, "deleted_at")
}

func (g *ZeroTrustGatewayGenerator) appendNetworkHostnameRouteResources(ctx context.Context, api *cf.API, accountID string) error {
	routes, err := listZeroTrustGatewayResources(ctx, api, fmt.Sprintf("/accounts/%s/zerotrust/routes/hostname", accountID))
	if err != nil {
		return err
	}
	for _, route := range routes {
		if !zeroTrustNetworkHostnameRouteImportable(route) {
			continue
		}
		id := zeroTrustGatewayString(route, "id")
		g.Resources = append(g.Resources, zeroTrustGatewayAccountResource(
			accountID,
			id,
			cloudflareResourceName(
				accountID,
				"network_hostname_route",
				zeroTrustGatewayString(route, "hostname"),
				zeroTrustGatewayString(route, "tunnel_name"),
				zeroTrustGatewayString(route, "tunnel_id"),
				id,
			),
			"cloudflare_zero_trust_network_hostname_route",
		))
	}
	return nil
}

func (g *ZeroTrustGatewayGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	account, err := g.accountResourceContainer()
	if err != nil {
		return err
	}
	return runZeroTrustGatewayDiscoveries([]zeroTrustGatewayDiscovery{
		{
			name:    "DNS locations",
			account: account.Identifier,
			discover: func() error {
				return g.appendDNSLocationResources(ctx, api, account.Identifier)
			},
		},
		{
			name:    "Gateway certificates",
			account: account.Identifier,
			discover: func() error {
				return g.appendGatewayCertificateResources(ctx, api, account.Identifier)
			},
		},
		{
			name:    "Gateway logging",
			account: account.Identifier,
			discover: func() error {
				return g.appendGatewayLoggingResource(ctx, api, account.Identifier)
			},
		},
		{
			name:    "Gateway PAC files",
			account: account.Identifier,
			discover: func() error {
				return g.appendGatewayPacfileResources(ctx, api, account.Identifier)
			},
		},
		{
			name:    "Gateway policies",
			account: account.Identifier,
			discover: func() error {
				return g.appendGatewayPolicyResources(ctx, api, account.Identifier)
			},
		},
		{
			name:    "Gateway proxy endpoints",
			account: account.Identifier,
			discover: func() error {
				return g.appendGatewayProxyEndpointResources(ctx, api, account.Identifier)
			},
		},
		{
			name:    "Gateway settings",
			account: account.Identifier,
			discover: func() error {
				return g.appendGatewaySettingsResource(ctx, api, account.Identifier)
			},
		},
		{
			name:    "Zero Trust lists",
			account: account.Identifier,
			discover: func() error {
				return g.appendZeroTrustListResources(ctx, api, account.Identifier)
			},
		},
		{
			name:    "hostname routes",
			account: account.Identifier,
			discover: func() error {
				return g.appendNetworkHostnameRouteResources(ctx, api, account.Identifier)
			},
		},
	})
}
