// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type MediaPlatformGenerator struct {
	CloudflareService
}

type cloudflareMediaPlatformRawResource map[string]interface{}

type cloudflareMediaPlatformDiscovery struct {
	name      string
	scope     string
	resources *[]terraformutils.Resource
	discover  func() error
}

func cloudflareMediaPlatformString(resource cloudflareMediaPlatformRawResource, keys ...string) string {
	for _, key := range keys {
		value, ok := resource[key].(string)
		if ok && value != "" {
			return value
		}
	}
	return ""
}

func cloudflareMediaPlatformOptionalDiscoveryError(err error) bool {
	var notFoundErr *cf.NotFoundError
	if errors.As(err, &notFoundErr) {
		return cloudflareMediaPlatformOptionalErrorMessage(notFoundErr.Error(), notFoundErr.ErrorMessages())
	}

	var requestErr *cf.RequestError
	if errors.As(err, &requestErr) {
		return cloudflareMediaPlatformOptionalErrorMessage(requestErr.Error(), requestErr.ErrorMessages())
	}

	var authenticationErr *cf.AuthenticationError
	if errors.As(err, &authenticationErr) {
		return cloudflareMediaPlatformOptionalErrorMessage(authenticationErr.Error(), authenticationErr.ErrorMessages())
	}

	var authorizationErr *cf.AuthorizationError
	if errors.As(err, &authorizationErr) {
		return cloudflareMediaPlatformOptionalErrorMessage(authorizationErr.Error(), authorizationErr.ErrorMessages())
	}

	return false
}

func cloudflareMediaPlatformOptionalErrorMessage(message string, errorMessages []string) bool {
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
			"unauthorized",
			"upgrade your plan",
		} {
			if strings.Contains(normalized, marker) {
				return true
			}
		}
	}
	return false
}

func runCloudflareMediaPlatformDiscoveries(discoveries []cloudflareMediaPlatformDiscovery) error {
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
			if cloudflareMediaPlatformOptionalDiscoveryError(err) {
				log.Printf("Skipping Cloudflare media platform %s discovery for %s: %v", discovery.name, discovery.scope, err)
				continue
			}
			return fmt.Errorf("discover Cloudflare media platform %s for %s: %w", discovery.name, discovery.scope, err)
		}
	}
	return nil
}

func cloudflareMediaPlatformPagedPath(path string, page int, cursor string) string {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return fmt.Sprintf("%s%s%s", path, separator, cloudflarePaginationQuery(page, cursor))
}

func listCloudflareMediaPlatformResources(
	ctx context.Context,
	api *cf.API,
	path string,
) ([]cloudflareMediaPlatformRawResource, error) {
	var resources []cloudflareMediaPlatformRawResource
	page, cursor := 1, ""
	for {
		response, err := api.Raw(ctx, http.MethodGet, cloudflareMediaPlatformPagedPath(path, page, cursor), nil, nil)
		if err != nil {
			return nil, err
		}
		if len(response.Result) == 0 || string(response.Result) == "null" {
			return resources, nil
		}

		var pageResources []cloudflareMediaPlatformRawResource
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

func listCloudflareImageVariantResources(
	ctx context.Context,
	api *cf.API,
	accountID string,
) ([]cloudflareMediaPlatformRawResource, error) {
	response, err := api.Raw(ctx, http.MethodGet, fmt.Sprintf("/accounts/%s/images/v1/variants", accountID), nil, nil)
	if err != nil {
		return nil, err
	}
	if len(response.Result) == 0 || string(response.Result) == "null" {
		return nil, nil
	}

	var result map[string]map[string]cloudflareMediaPlatformRawResource
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return nil, err
	}
	variantResults := result["variants"]
	variants := make([]cloudflareMediaPlatformRawResource, 0, len(variantResults))
	for variantID, variant := range variantResults {
		if cloudflareMediaPlatformString(variant, "id") == "" {
			variant["id"] = variantID
		}
		variants = append(variants, variant)
	}
	sort.Slice(variants, func(i, j int) bool {
		return cloudflareMediaPlatformString(variants[i], "id") < cloudflareMediaPlatformString(variants[j], "id")
	})
	return variants, nil
}

func cloudflareAccountMediaPlatformResource(
	accountID string,
	id string,
	resourceType string,
	resourceNamePrefix string,
	attributes map[string]string,
	nameParts ...string,
) (terraformutils.Resource, bool) {
	if accountID == "" || id == "" {
		return terraformutils.Resource{}, false
	}
	if attributes == nil {
		attributes = map[string]string{}
	}
	attributes["account_id"] = accountID

	parts := append([]string{accountID, resourceNamePrefix}, nameParts...)
	parts = append(parts, id)
	resource := terraformutils.NewResource(
		id,
		cloudflareResourceName(parts...),
		resourceType,
		"cloudflare",
		attributes,
		[]string{},
		map[string]interface{}{},
	)
	setCloudflareImportID(&resource, accountID+"/"+id)
	return resource, true
}

func cloudflareMediaPlatformBoolAttribute(resource cloudflareMediaPlatformRawResource, key string) string {
	value, ok := resource[key].(bool)
	if !ok {
		return strconv.FormatBool(false)
	}
	return strconv.FormatBool(value)
}

func cloudflareMediaPlatformIntAttribute(resource cloudflareMediaPlatformRawResource, key string) string {
	switch value := resource[key].(type) {
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	case float64:
		return strconv.FormatInt(int64(value), 10)
	case json.Number:
		return value.String()
	case string:
		if value != "" {
			return value
		}
	}
	return "0"
}

func cloudflareMediaPlatformFloatAttribute(resource cloudflareMediaPlatformRawResource, key string) (string, bool) {
	switch value := resource[key].(type) {
	case int:
		return strconv.Itoa(value), true
	case int64:
		return strconv.FormatInt(value, 10), true
	case float64:
		if value == 0 {
			return "", false
		}
		return strconv.FormatFloat(value, 'f', -1, 64), true
	case json.Number:
		if value.String() == "" || value.String() == "0" {
			return "", false
		}
		return value.String(), true
	case string:
		if value != "" && value != "0" {
			return value, true
		}
	}
	return "", false
}

func cloudflareMediaPlatformObject(resource cloudflareMediaPlatformRawResource, key string) (cloudflareMediaPlatformRawResource, bool) {
	object, ok := resource[key].(map[string]interface{})
	if !ok || len(object) == 0 {
		return nil, false
	}
	return cloudflareMediaPlatformRawResource(object), true
}

func cloudflareMediaPlatformAIGatewayImportable(gateway cloudflareMediaPlatformRawResource) bool {
	if value, ok := gateway["stripe"]; ok && value != nil {
		return false
	}
	if otel, ok := gateway["otel"].([]interface{}); ok && len(otel) > 0 {
		return false
	}
	return true
}

func newCloudflareAIGatewayResource(accountID string, gateway cloudflareMediaPlatformRawResource) (terraformutils.Resource, bool) {
	id := cloudflareMediaPlatformString(gateway, "id")
	if id == "" || !cloudflareMediaPlatformAIGatewayImportable(gateway) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"cache_invalidate_on_update": cloudflareMediaPlatformBoolAttribute(gateway, "cache_invalidate_on_update"),
		"cache_ttl":                  cloudflareMediaPlatformIntAttribute(gateway, "cache_ttl"),
		"collect_logs":               cloudflareMediaPlatformBoolAttribute(gateway, "collect_logs"),
		"rate_limiting_interval":     cloudflareMediaPlatformIntAttribute(gateway, "rate_limiting_interval"),
		"rate_limiting_limit":        cloudflareMediaPlatformIntAttribute(gateway, "rate_limiting_limit"),
	}
	resource, ok := cloudflareAccountMediaPlatformResource(accountID, id, "cloudflare_ai_gateway", "ai_gateway", attributes)
	if !ok {
		return resource, false
	}
	resource.AdditionalFields["id"] = id
	return resource, true
}

func newCloudflareImageVariantResource(accountID string, variant cloudflareMediaPlatformRawResource) (terraformutils.Resource, bool) {
	id := cloudflareMediaPlatformString(variant, "id")
	options, ok := cloudflareMediaPlatformObject(variant, "options")
	if id == "" || !ok {
		return terraformutils.Resource{}, false
	}
	height, ok := cloudflareMediaPlatformFloatAttribute(options, "height")
	if !ok {
		return terraformutils.Resource{}, false
	}
	width, ok := cloudflareMediaPlatformFloatAttribute(options, "width")
	if !ok {
		return terraformutils.Resource{}, false
	}
	fit := cloudflareMediaPlatformString(options, "fit")
	metadata := cloudflareMediaPlatformString(options, "metadata")
	if fit == "" || metadata == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"never_require_signed_urls": cloudflareMediaPlatformBoolAttribute(variant, "neverRequireSignedURLs"),
		"options.fit":               fit,
		"options.height":            height,
		"options.metadata":          metadata,
		"options.width":             width,
	}
	resource, ok := cloudflareAccountMediaPlatformResource(accountID, id, "cloudflare_image_variant", "image_variant", attributes)
	if !ok {
		return resource, false
	}
	resource.AdditionalFields["id"] = id
	setCloudflarePreserveIDAfterRefresh(&resource)
	return resource, true
}

func newCloudflarePipelineResource(accountID string, pipeline cloudflareMediaPlatformRawResource) (terraformutils.Resource, bool) {
	id := cloudflareMediaPlatformString(pipeline, "id")
	name := cloudflareMediaPlatformString(pipeline, "name")
	sql := cloudflareMediaPlatformString(pipeline, "sql")
	if id == "" || name == "" || sql == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"name": name,
		"sql":  sql,
	}
	return cloudflareAccountMediaPlatformResource(accountID, id, "cloudflare_pipeline", "pipeline", attributes, name)
}

func newCloudflarePipelineStreamResource(accountID string, stream cloudflareMediaPlatformRawResource) (terraformutils.Resource, bool) {
	id := cloudflareMediaPlatformString(stream, "id")
	name := cloudflareMediaPlatformString(stream, "name")
	if id == "" || name == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{"name": name}
	return cloudflareAccountMediaPlatformResource(accountID, id, "cloudflare_pipeline_stream", "pipeline_stream", attributes, name)
}

func (g *MediaPlatformGenerator) appendAIGatewayResources(ctx context.Context, api *cf.API, accountID string) error {
	gateways, err := listCloudflareMediaPlatformResources(ctx, api, fmt.Sprintf("/accounts/%s/ai-gateway/gateways", accountID))
	if err != nil {
		return err
	}
	for _, gateway := range gateways {
		resource, ok := newCloudflareAIGatewayResource(accountID, gateway)
		if !ok {
			if cloudflareMediaPlatformString(gateway, "id") != "" {
				log.Printf("Skipping Cloudflare AI Gateway %s because it has credential-backed OTEL or Stripe configuration", cloudflareMediaPlatformString(gateway, "id"))
			}
			continue
		}
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *MediaPlatformGenerator) appendImageVariantResources(ctx context.Context, api *cf.API, accountID string) error {
	variants, err := listCloudflareImageVariantResources(ctx, api, accountID)
	if err != nil {
		return err
	}
	for _, variant := range variants {
		resource, ok := newCloudflareImageVariantResource(accountID, variant)
		if !ok {
			continue
		}
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *MediaPlatformGenerator) appendPipelineResources(ctx context.Context, api *cf.API, accountID string) error {
	pipelines, err := listCloudflareMediaPlatformResources(ctx, api, fmt.Sprintf("/accounts/%s/pipelines/v1/pipelines", accountID))
	if err != nil {
		return err
	}
	for _, pipeline := range pipelines {
		resource, ok := newCloudflarePipelineResource(accountID, pipeline)
		if !ok {
			continue
		}
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *MediaPlatformGenerator) appendPipelineStreamResources(ctx context.Context, api *cf.API, accountID string) error {
	streams, err := listCloudflareMediaPlatformResources(ctx, api, fmt.Sprintf("/accounts/%s/pipelines/v1/streams", accountID))
	if err != nil {
		return err
	}
	for _, stream := range streams {
		resource, ok := newCloudflarePipelineStreamResource(accountID, stream)
		if !ok {
			continue
		}
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *MediaPlatformGenerator) appendMediaPlatformResources(ctx context.Context, api *cf.API, accountID string) error {
	return runCloudflareMediaPlatformDiscoveries([]cloudflareMediaPlatformDiscovery{
		{
			name:      "AI Gateway",
			scope:     accountID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendAIGatewayResources(ctx, api, accountID)
			},
		},
		{
			name:      "Images variants",
			scope:     accountID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendImageVariantResources(ctx, api, accountID)
			},
		},
		{
			name:      "Pipelines",
			scope:     accountID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendPipelineResources(ctx, api, accountID)
			},
		},
		{
			name:      "Pipeline streams",
			scope:     accountID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendPipelineStreamResources(ctx, api, accountID)
			},
		},
	})
}

func (g *MediaPlatformGenerator) InitResources() error {
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	accountID := g.accountID()
	if accountID == "" {
		return errors.New("set CLOUDFLARE_ACCOUNT_ID env var")
	}
	return g.appendMediaPlatformResources(context.Background(), api, accountID)
}
