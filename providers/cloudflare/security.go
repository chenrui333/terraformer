// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	cf "github.com/chenrui333/terraformer/providers/cloudflare/internal/cloudflarev7"
	"github.com/chenrui333/terraformer/terraformutils"
)

type SecurityGenerator struct {
	CloudflareService
}

type cloudflareSecurityRawResource map[string]interface{}

type cloudflareSecurityDiscovery struct {
	name     string
	scope    string
	discover func() error
}

func cloudflareSecurityString(resource cloudflareSecurityRawResource, keys ...string) string {
	for _, key := range keys {
		value, ok := resource[key].(string)
		if ok && value != "" {
			return value
		}
	}
	return ""
}

func cloudflareSecurityIDString(resource cloudflareSecurityRawResource, keys ...string) string {
	for _, key := range keys {
		value, ok := resource[key]
		if !ok {
			continue
		}
		switch id := value.(type) {
		case string:
			if id != "" {
				return id
			}
		case float64:
			if id == float64(int64(id)) {
				return strconv.FormatInt(int64(id), 10)
			}
		}
	}
	return ""
}

func cloudflareSecurityOptionalDiscoveryError(err error) bool {
	var notFoundErr *cf.NotFoundError
	if errors.As(err, &notFoundErr) {
		return cloudflareSecurityOptionalErrorMessage(notFoundErr.Error(), notFoundErr.ErrorMessages())
	}

	var requestErr *cf.RequestError
	if errors.As(err, &requestErr) {
		return cloudflareSecurityOptionalErrorMessage(requestErr.Error(), requestErr.ErrorMessages())
	}

	return false
}

func cloudflareSecurityOptionalErrorMessage(message string, errorMessages []string) bool {
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

func runCloudflareSecurityDiscoveries(discoveries []cloudflareSecurityDiscovery) error {
	for _, discovery := range discoveries {
		if discovery.discover == nil {
			continue
		}
		if err := discovery.discover(); err != nil {
			if cloudflareSecurityOptionalDiscoveryError(err) {
				log.Printf("Skipping Cloudflare security %s discovery for %s: %v", discovery.name, discovery.scope, err)
				continue
			}
			return fmt.Errorf("discover Cloudflare security %s for %s: %w", discovery.name, discovery.scope, err)
		}
	}
	return nil
}

func cloudflareSecurityPagedPath(path string, page int, cursor string) string {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return fmt.Sprintf("%s%s%s", path, separator, cloudflarePaginationQuery(page, cursor))
}

func listCloudflareSecurityResources(ctx context.Context, api *cf.API, path string) ([]cloudflareSecurityRawResource, error) {
	var resources []cloudflareSecurityRawResource
	page, cursor := 1, ""
	for {
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			cloudflareSecurityPagedPath(path, page, cursor),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		if len(response.Result) == 0 || string(response.Result) == "null" {
			return resources, nil
		}

		var pageResources []cloudflareSecurityRawResource
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

func getCloudflareSecurityResource(ctx context.Context, api *cf.API, path string) (cloudflareSecurityRawResource, bool, error) {
	response, err := api.Raw(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, false, err
	}
	if len(response.Result) == 0 || string(response.Result) == "null" {
		return nil, false, nil
	}
	var resource cloudflareSecurityRawResource
	if err := json.Unmarshal(response.Result, &resource); err != nil {
		return nil, false, err
	}
	return resource, len(resource) > 0, nil
}

func cloudflareZoneSecurityResource(
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

func cloudflareZoneSecuritySingletonResource(
	zone cf.Zone,
	resourceType string,
	resourceNamePrefix string,
) (terraformutils.Resource, bool) {
	if zone.ID == "" {
		return terraformutils.Resource{}, false
	}
	resource := terraformutils.NewResource(
		zone.ID,
		cloudflareResourceName(zone.Name, resourceNamePrefix),
		resourceType,
		"cloudflare",
		map[string]string{"zone_id": zone.ID},
		[]string{},
		map[string]interface{}{},
	)
	setCloudflareImportID(&resource, zone.ID)
	return resource, true
}

func cloudflareAccountSecurityResource(
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

func cloudflareScopedSecurityResource(
	scopeType string,
	scopeID string,
	id string,
	resourceType string,
	resourceNamePrefix string,
	nameParts ...string,
) (terraformutils.Resource, bool) {
	if scopeID == "" || id == "" {
		return terraformutils.Resource{}, false
	}
	parts := append([]string{scopeType, scopeID, resourceNamePrefix}, nameParts...)
	parts = append(parts, id)
	resource := terraformutils.NewResource(
		id,
		cloudflareResourceName(parts...),
		resourceType,
		"cloudflare",
		accessScopeAttributes(scopeType, scopeID),
		[]string{},
		map[string]interface{}{},
	)
	setCloudflareImportID(&resource, fmt.Sprintf("%s/%s/%s", scopeType, scopeID, id))
	return resource, true
}

func apiShieldConfigImportable(config cloudflareSecurityRawResource) bool {
	characteristics, ok := config["auth_id_characteristics"].([]interface{})
	return ok && len(characteristics) > 0
}

func (g *SecurityGenerator) appendAPIShieldResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	config, exists, err := getCloudflareSecurityResource(ctx, api, fmt.Sprintf("/zones/%s/api_gateway/configuration", zone.ID))
	if err != nil {
		return err
	}
	if !exists || !apiShieldConfigImportable(config) {
		return nil
	}
	resource, ok := cloudflareZoneSecuritySingletonResource(zone, "cloudflare_api_shield", "api_shield")
	if ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *SecurityGenerator) appendAPIShieldOperationResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	operations, err := listCloudflareSecurityResources(ctx, api, fmt.Sprintf("/zones/%s/api_gateway/operations", zone.ID))
	if err != nil {
		return err
	}
	for _, operation := range operations {
		id := cloudflareSecurityString(operation, "operation_id", "id")
		resource, ok := cloudflareZoneSecurityResource(
			zone,
			id,
			"cloudflare_api_shield_operation",
			"api_shield_operation",
			cloudflareSecurityString(operation, "host"),
			cloudflareSecurityString(operation, "method"),
			cloudflareSecurityString(operation, "endpoint"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SecurityGenerator) appendSchemaValidationSchemaResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	schemas, err := listCloudflareSecurityResources(ctx, api, fmt.Sprintf("/zones/%s/schema_validation/schemas", zone.ID))
	if err != nil {
		return err
	}
	for _, schema := range schemas {
		id := cloudflareSecurityString(schema, "schema_id", "id")
		resource, ok := cloudflareZoneSecurityResource(
			zone,
			id,
			"cloudflare_schema_validation_schemas",
			"schema_validation_schema",
			cloudflareSecurityString(schema, "name"),
			cloudflareSecurityString(schema, "kind"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SecurityGenerator) appendPageShieldPolicyResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	policies, err := listCloudflareSecurityResources(ctx, api, fmt.Sprintf("/zones/%s/page_shield/policies", zone.ID))
	if err != nil {
		return err
	}
	for _, policy := range policies {
		id := cloudflareSecurityString(policy, "id")
		resource, ok := cloudflareZoneSecurityResource(
			zone,
			id,
			"cloudflare_page_shield_policy",
			"page_shield_policy",
			cloudflareSecurityString(policy, "description"),
			cloudflareSecurityString(policy, "action"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SecurityGenerator) appendCloudConnectorRulesResource(ctx context.Context, api *cf.API, zone cf.Zone) error {
	rules, err := listCloudflareSecurityResources(ctx, api, fmt.Sprintf("/zones/%s/cloud_connector/rules", zone.ID))
	if err != nil {
		return err
	}
	if len(rules) == 0 {
		return nil
	}
	resource, ok := cloudflareZoneSecuritySingletonResource(zone, "cloudflare_cloud_connector_rules", "cloud_connector_rules")
	if ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *SecurityGenerator) appendCustomPageResources(
	ctx context.Context,
	api *cf.API,
	scopeType string,
	scopeID string,
	scopeName string,
) error {
	pages, err := listCloudflareSecurityResources(ctx, api, fmt.Sprintf("/%s/%s/custom_pages", scopeType, scopeID))
	if err != nil {
		return err
	}
	for _, page := range pages {
		if cloudflareSecurityString(page, "state") != "customized" {
			continue
		}
		id := cloudflareSecurityIDString(page, "identifier", "id")
		resource, ok := cloudflareScopedSecurityResource(
			scopeType,
			scopeID,
			id,
			"cloudflare_custom_pages",
			"custom_page",
			scopeName,
			cloudflareSecurityString(page, "description", "state"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SecurityGenerator) appendCustomPageAssetResources(
	ctx context.Context,
	api *cf.API,
	scopeType string,
	scopeID string,
	scopeName string,
) error {
	assets, err := listCloudflareSecurityResources(ctx, api, fmt.Sprintf("/%s/%s/custom_pages/assets", scopeType, scopeID))
	if err != nil {
		return err
	}
	for _, asset := range assets {
		id := cloudflareSecurityIDString(asset, "name", "id")
		resource, ok := cloudflareScopedSecurityResource(
			scopeType,
			scopeID,
			id,
			"cloudflare_custom_page_asset",
			"custom_page_asset",
			scopeName,
			cloudflareSecurityString(asset, "description"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SecurityGenerator) appendLeakedCredentialCheckRuleResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	detections, err := listCloudflareSecurityResources(ctx, api, fmt.Sprintf("/zones/%s/leaked-credential-checks/detections", zone.ID))
	if err != nil {
		return err
	}
	for _, detection := range detections {
		id := cloudflareSecurityIDString(detection, "id")
		resource, ok := cloudflareZoneSecurityResource(
			zone,
			id,
			"cloudflare_leaked_credential_check_rule",
			"leaked_credential_check_rule",
			cloudflareSecurityString(detection, "username"),
			cloudflareSecurityString(detection, "password"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SecurityGenerator) appendTokenValidationConfigResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	configs, err := listCloudflareSecurityResources(ctx, api, fmt.Sprintf("/zones/%s/token_validation/config", zone.ID))
	if err != nil {
		return err
	}
	for _, config := range configs {
		id := cloudflareSecurityIDString(config, "id")
		resource, ok := cloudflareZoneSecurityResource(
			zone,
			id,
			"cloudflare_token_validation_config",
			"token_validation_config",
			cloudflareSecurityString(config, "title"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SecurityGenerator) appendTokenValidationRuleResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	rules, err := listCloudflareSecurityResources(ctx, api, fmt.Sprintf("/zones/%s/token_validation/rules", zone.ID))
	if err != nil {
		return err
	}
	for _, rule := range rules {
		id := cloudflareSecurityIDString(rule, "id")
		resource, ok := cloudflareZoneSecurityResource(
			zone,
			id,
			"cloudflare_token_validation_rules",
			"token_validation_rule",
			cloudflareSecurityString(rule, "title", "action"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SecurityGenerator) appendUserAgentBlockingRuleResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	rules, err := listCloudflareSecurityResources(ctx, api, fmt.Sprintf("/zones/%s/firewall/ua_rules", zone.ID))
	if err != nil {
		return err
	}
	for _, rule := range rules {
		id := cloudflareSecurityIDString(rule, "id")
		resource, ok := cloudflareZoneSecurityResource(
			zone,
			id,
			"cloudflare_user_agent_blocking_rule",
			"user_agent_blocking_rule",
			cloudflareSecurityString(rule, "description", "mode"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SecurityGenerator) appendVulnerabilityScannerCredentialSetResources(
	ctx context.Context,
	api *cf.API,
	accountID string,
) error {
	credentialSets, err := listCloudflareSecurityResources(ctx, api, fmt.Sprintf("/accounts/%s/vuln_scanner/credential_sets", accountID))
	if err != nil {
		return err
	}
	for _, credentialSet := range credentialSets {
		id := cloudflareSecurityString(credentialSet, "id")
		resource, ok := cloudflareAccountSecurityResource(
			accountID,
			id,
			"cloudflare_vulnerability_scanner_credential_set",
			"vulnerability_scanner_credential_set",
			cloudflareSecurityString(credentialSet, "name"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SecurityGenerator) appendVulnerabilityScannerTargetEnvironmentResources(
	ctx context.Context,
	api *cf.API,
	accountID string,
) error {
	targetEnvironments, err := listCloudflareSecurityResources(ctx, api, fmt.Sprintf("/accounts/%s/vuln_scanner/target_environments", accountID))
	if err != nil {
		return err
	}
	for _, targetEnvironment := range targetEnvironments {
		id := cloudflareSecurityString(targetEnvironment, "id")
		resource, ok := cloudflareAccountSecurityResource(
			accountID,
			id,
			"cloudflare_vulnerability_scanner_target_environment",
			"vulnerability_scanner_target_environment",
			cloudflareSecurityString(targetEnvironment, "name"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SecurityGenerator) appendEmailSecurityBlockSenderResources(ctx context.Context, api *cf.API, accountID string) error {
	senders, err := listCloudflareSecurityResources(ctx, api, fmt.Sprintf("/accounts/%s/email-security/settings/block_senders", accountID))
	if err != nil {
		return err
	}
	for _, sender := range senders {
		id := cloudflareSecurityIDString(sender, "id")
		resource, ok := cloudflareAccountSecurityResource(
			accountID,
			id,
			"cloudflare_email_security_block_sender",
			"email_security_block_sender",
			cloudflareSecurityString(sender, "pattern", "pattern_type"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SecurityGenerator) appendEmailSecurityImpersonationRegistryResources(ctx context.Context, api *cf.API, accountID string) error {
	registries, err := listCloudflareSecurityResources(ctx, api, fmt.Sprintf("/accounts/%s/email-security/settings/impersonation_registry", accountID))
	if err != nil {
		return err
	}
	for _, registry := range registries {
		id := cloudflareSecurityIDString(registry, "id")
		resource, ok := cloudflareAccountSecurityResource(
			accountID,
			id,
			"cloudflare_email_security_impersonation_registry",
			"email_security_impersonation_registry",
			cloudflareSecurityString(registry, "name", "email"),
		)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SecurityGenerator) appendZoneSecurityResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	return runCloudflareSecurityDiscoveries([]cloudflareSecurityDiscovery{
		{
			name:  "API Shield configuration",
			scope: zone.ID,
			discover: func() error {
				return g.appendAPIShieldResource(ctx, api, zone)
			},
		},
		{
			name:  "API Shield operations",
			scope: zone.ID,
			discover: func() error {
				return g.appendAPIShieldOperationResources(ctx, api, zone)
			},
		},
		{
			name:  "schema validation schemas",
			scope: zone.ID,
			discover: func() error {
				return g.appendSchemaValidationSchemaResources(ctx, api, zone)
			},
		},
		{
			name:  "Page Shield policies",
			scope: zone.ID,
			discover: func() error {
				return g.appendPageShieldPolicyResources(ctx, api, zone)
			},
		},
		{
			name:  "Cloud Connector rules",
			scope: zone.ID,
			discover: func() error {
				return g.appendCloudConnectorRulesResource(ctx, api, zone)
			},
		},
		{
			name:  "zone custom pages",
			scope: zone.ID,
			discover: func() error {
				return g.appendCustomPageResources(ctx, api, "zones", zone.ID, zone.Name)
			},
		},
		{
			name:  "zone custom page assets",
			scope: zone.ID,
			discover: func() error {
				return g.appendCustomPageAssetResources(ctx, api, "zones", zone.ID, zone.Name)
			},
		},
		{
			name:  "leaked credential check rules",
			scope: zone.ID,
			discover: func() error {
				return g.appendLeakedCredentialCheckRuleResources(ctx, api, zone)
			},
		},
		{
			name:  "token validation configs",
			scope: zone.ID,
			discover: func() error {
				return g.appendTokenValidationConfigResources(ctx, api, zone)
			},
		},
		{
			name:  "token validation rules",
			scope: zone.ID,
			discover: func() error {
				return g.appendTokenValidationRuleResources(ctx, api, zone)
			},
		},
		{
			name:  "user-agent blocking rules",
			scope: zone.ID,
			discover: func() error {
				return g.appendUserAgentBlockingRuleResources(ctx, api, zone)
			},
		},
	})
}

func (g *SecurityGenerator) appendAccountSecurityResources(ctx context.Context, api *cf.API, accountID string) error {
	return runCloudflareSecurityDiscoveries([]cloudflareSecurityDiscovery{
		{
			name:  "vulnerability scanner credential sets",
			scope: accountID,
			discover: func() error {
				return g.appendVulnerabilityScannerCredentialSetResources(ctx, api, accountID)
			},
		},
		{
			name:  "vulnerability scanner target environments",
			scope: accountID,
			discover: func() error {
				return g.appendVulnerabilityScannerTargetEnvironmentResources(ctx, api, accountID)
			},
		},
		{
			name:  "account custom pages",
			scope: accountID,
			discover: func() error {
				return g.appendCustomPageResources(ctx, api, "accounts", accountID, accountID)
			},
		},
		{
			name:  "account custom page assets",
			scope: accountID,
			discover: func() error {
				return g.appendCustomPageAssetResources(ctx, api, "accounts", accountID, accountID)
			},
		},
		{
			name:  "Email Security block senders",
			scope: accountID,
			discover: func() error {
				return g.appendEmailSecurityBlockSenderResources(ctx, api, accountID)
			},
		},
		{
			name:  "Email Security impersonation registry",
			scope: accountID,
			discover: func() error {
				return g.appendEmailSecurityImpersonationRegistryResources(ctx, api, accountID)
			},
		},
	})
}

func (g *SecurityGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}

	zones, err := cloudflareZones(ctx, api)
	if err != nil {
		return err
	}
	for _, zone := range zones {
		if err := g.appendZoneSecurityResources(ctx, api, zone); err != nil {
			return err
		}
	}

	accountID := g.accountID()
	if accountID == "" {
		return nil
	}
	return g.appendAccountSecurityResources(ctx, api, accountID)
}
