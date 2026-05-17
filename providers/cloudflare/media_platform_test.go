// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

func TestCloudflareProviderIncludesMediaPlatformService(t *testing.T) {
	services := (&CloudflareProvider{}).GetSupportedService()
	if _, ok := services["media_platform"]; !ok {
		t.Fatal("media_platform service is not registered")
	}
}

func TestCloudflareMediaPlatformOptionalErrorMessageDoesNotHideGenericNotFound(t *testing.T) {
	if cloudflareMediaPlatformOptionalErrorMessage("not found", nil) {
		t.Fatal("generic not found should not be treated as optional")
	}
	if !cloudflareMediaPlatformOptionalErrorMessage("", []string{"feature is not available on this plan"}) {
		t.Fatal("feature-gated errors should be treated as optional")
	}
	if !cloudflareMediaPlatformOptionalErrorMessage("", []string{"Unauthorized to access requested resource"}) {
		t.Fatal("unauthorized permission errors should be treated as optional")
	}
}

func TestCloudflareMediaPlatformOptionalDiscoveryErrorHandlesAuthErrors(t *testing.T) {
	authenticationErr := cf.NewAuthenticationError(&cf.Error{ErrorMessages: []string{"not authorized"}})
	if !cloudflareMediaPlatformOptionalDiscoveryError(&authenticationErr) {
		t.Fatal("authentication errors should be treated as optional when they match optional markers")
	}

	authenticationErrWithoutMarker := cf.NewAuthenticationError(&cf.Error{ErrorMessages: []string{"invalid token"}})
	if cloudflareMediaPlatformOptionalDiscoveryError(&authenticationErrWithoutMarker) {
		t.Fatal("authentication errors without optional markers should propagate")
	}

	authorizationErr := cf.NewAuthorizationError(&cf.Error{ErrorMessages: []string{"missing permission"}})
	if !cloudflareMediaPlatformOptionalDiscoveryError(&authorizationErr) {
		t.Fatal("authorization errors should be treated as optional when they match optional markers")
	}

	authorizationErrWithoutMarker := cf.NewAuthorizationError(&cf.Error{ErrorMessages: []string{"policy rejected request"}})
	if cloudflareMediaPlatformOptionalDiscoveryError(&authorizationErrWithoutMarker) {
		t.Fatal("authorization errors without optional markers should propagate")
	}
}

func TestCloudflareMediaPlatformOptionalDiscoveryErrorDoesNotHideGenericNotFound(t *testing.T) {
	if cloudflareMediaPlatformOptionalDiscoveryError(testCloudflareNotFoundError("not found")) {
		t.Fatal("generic not found should not be treated as optional")
	}
	if !cloudflareMediaPlatformOptionalDiscoveryError(testCloudflareNotFoundError("feature is not available for this account")) {
		t.Fatal("feature-gated not found errors should be treated as optional")
	}
}

func TestNewCloudflareImageVariantResourceSeedsOptions(t *testing.T) {
	resource, ok := newCloudflareImageVariantResource("account-123", cloudflareMediaPlatformRawResource{
		"id":                     "thumb",
		"neverRequireSignedURLs": true,
		"options": map[string]interface{}{
			"fit":      "cover",
			"height":   float64(320),
			"metadata": "none",
			"width":    float64(640),
		},
	})
	if !ok {
		t.Fatal("expected image variant resource")
	}
	if resource.InstanceInfo.Type != "cloudflare_image_variant" {
		t.Fatalf("resource type = %q, want cloudflare_image_variant", resource.InstanceInfo.Type)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "account-123/thumb" {
		t.Fatalf("import_id = %q, want account-123/thumb", got)
	}
	if got := resource.AdditionalFields["id"]; got != "thumb" {
		t.Fatalf("AdditionalFields[id] = %q, want thumb", got)
	}
	assertCloudflarePreservesID(t, resource)
	for key, want := range map[string]string{
		"account_id":                "account-123",
		"never_require_signed_urls": "true",
		"options.fit":               "cover",
		"options.height":            "320",
		"options.metadata":          "none",
		"options.width":             "640",
	} {
		if got := resource.InstanceState.Attributes[key]; got != want {
			t.Fatalf("attribute %s = %q, want %q", key, got, want)
		}
	}
}

func TestNewCloudflarePipelineResource(t *testing.T) {
	resource, ok := newCloudflarePipelineResource("account-123", cloudflareMediaPlatformRawResource{
		"id":   "pipeline-123",
		"name": "events",
		"sql":  "SELECT * FROM stream",
	})
	if !ok {
		t.Fatal("expected pipeline resource")
	}
	if resource.InstanceInfo.Type != "cloudflare_pipeline" {
		t.Fatalf("resource type = %q, want cloudflare_pipeline", resource.InstanceInfo.Type)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "account-123/pipeline-123" {
		t.Fatalf("import_id = %q, want account-123/pipeline-123", got)
	}
	if got := resource.InstanceState.Attributes["name"]; got != "events" {
		t.Fatalf("name = %q, want events", got)
	}
	if got := resource.InstanceState.Attributes["sql"]; got != "SELECT * FROM stream" {
		t.Fatalf("sql = %q, want SELECT * FROM stream", got)
	}
}

func TestNewCloudflarePipelineStreamResource(t *testing.T) {
	resource, ok := newCloudflarePipelineStreamResource("account-123", cloudflareMediaPlatformRawResource{
		"id":   "stream-123",
		"name": "events",
		"schema": map[string]interface{}{
			"fields": []interface{}{
				map[string]interface{}{"name": "payload", "type": "json"},
				map[string]interface{}{"name": "count", "type": "int64"},
			},
		},
	})
	if !ok {
		t.Fatal("expected pipeline stream resource")
	}
	if resource.InstanceInfo.Type != "cloudflare_pipeline_stream" {
		t.Fatalf("resource type = %q, want cloudflare_pipeline_stream", resource.InstanceInfo.Type)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "account-123/stream-123" {
		t.Fatalf("import_id = %q, want account-123/stream-123", got)
	}
	if got := resource.InstanceState.Attributes["name"]; got != "events" {
		t.Fatalf("name = %q, want events", got)
	}
}

func TestNewCloudflarePipelineStreamResourceSkipsUnsupportedSchemaFields(t *testing.T) {
	for _, fieldType := range []string{"struct", "list", "unknown", ""} {
		t.Run(fieldType, func(t *testing.T) {
			if _, ok := newCloudflarePipelineStreamResource("account-123", cloudflareMediaPlatformRawResource{
				"id":   "stream-123",
				"name": "events",
				"schema": map[string]interface{}{
					"fields": []interface{}{
						map[string]interface{}{"name": "payload", "type": fieldType},
					},
				},
			}); ok {
				t.Fatalf("expected pipeline stream with schema field type %q to be skipped", fieldType)
			}
		})
	}
}

func TestCloudflareMediaPlatformResourceConstructorsSkipMissingRequiredFields(t *testing.T) {
	if _, ok := newCloudflareImageVariantResource("account-123", cloudflareMediaPlatformRawResource{"id": "thumb"}); ok {
		t.Fatal("expected image variant without options to be skipped")
	}
	if _, ok := newCloudflarePipelineResource("account-123", cloudflareMediaPlatformRawResource{"id": "pipeline-123", "name": "events"}); ok {
		t.Fatal("expected pipeline without SQL to be skipped")
	}
	if _, ok := newCloudflarePipelineStreamResource("account-123", cloudflareMediaPlatformRawResource{"id": "stream-123"}); ok {
		t.Fatal("expected pipeline stream without name to be skipped")
	}
}

func TestRunCloudflareMediaPlatformDiscoveriesRollsBackFailedDiscovery(t *testing.T) {
	resources := []terraformutils.Resource{}
	err := runCloudflareMediaPlatformDiscoveries([]cloudflareMediaPlatformDiscovery{
		{
			name:      "fails",
			scope:     "account-123",
			resources: &resources,
			discover: func() error {
				resources = append(resources, terraformutils.NewResource("partial", "partial", "cloudflare_pipeline", "cloudflare", nil, nil, nil))
				return errors.New("unexpected response")
			},
		},
	})
	if err == nil {
		t.Fatal("expected non-optional discovery error")
	}
	if len(resources) != 0 {
		t.Fatalf("resources after failed discovery = %#v, want none", resources)
	}
}

func TestRunCloudflareMediaPlatformDiscoveriesContinuesAfterOptionalError(t *testing.T) {
	resources := []terraformutils.Resource{}
	err := runCloudflareMediaPlatformDiscoveries([]cloudflareMediaPlatformDiscovery{
		{
			name:      "permission gated",
			scope:     "account-123",
			resources: &resources,
			discover: func() error {
				resources = append(resources, terraformutils.NewResource("partial", "partial", "cloudflare_pipeline", "cloudflare", nil, nil, nil))
				authorizationErr := cf.NewAuthorizationError(&cf.Error{ErrorMessages: []string{"Unauthorized to access requested resource"}})
				return &authorizationErr
			},
		},
		{
			name:      "succeeds",
			scope:     "account-123",
			resources: &resources,
			discover: func() error {
				resources = append(resources, terraformutils.NewResource("complete", "complete", "cloudflare_pipeline", "cloudflare", nil, nil, nil))
				return nil
			},
		},
	})
	if err != nil {
		t.Fatalf("runCloudflareMediaPlatformDiscoveries() error = %v, want nil", err)
	}
	if len(resources) != 1 || resources[0].InstanceState.ID != "complete" {
		t.Fatalf("resources after optional discovery rollback = %#v, want only complete", resources)
	}
}

func TestListCloudflareMediaPlatformResourcesPaginates(t *testing.T) {
	api := newCloudflareMediaPlatformTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/account-123/pipelines/v1/pipelines" {
			t.Fatalf("path = %q, want /accounts/account-123/pipelines/v1/pipelines", r.URL.Path)
		}
		switch r.URL.Query().Get("cursor") {
		case "":
			if got := r.URL.Query().Get("page"); got != "1" {
				t.Fatalf("page query = %q, want 1", got)
			}
			writeCloudflareMediaPlatformTestResponse(t, w, []map[string]string{{"id": "pipeline-1"}}, map[string]interface{}{
				"cursors": map[string]string{"after": "cursor-2"},
			})
		case "cursor-2":
			writeCloudflareMediaPlatformTestResponse(t, w, []map[string]string{{"id": "pipeline-2"}}, map[string]interface{}{
				"cursors": map[string]string{},
			})
		default:
			t.Fatalf("cursor query = %q, want empty or cursor-2", r.URL.Query().Get("cursor"))
		}
	}))

	resources, err := listCloudflareMediaPlatformResources(context.Background(), api, "/accounts/account-123/pipelines/v1/pipelines")
	if err != nil {
		t.Fatalf("listCloudflareMediaPlatformResources() error = %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("resource count = %d, want 2", len(resources))
	}
	if got := cloudflareMediaPlatformString(resources[0], "id"); got != "pipeline-1" {
		t.Fatalf("first id = %q, want pipeline-1", got)
	}
	if got := cloudflareMediaPlatformString(resources[1], "id"); got != "pipeline-2" {
		t.Fatalf("second id = %q, want pipeline-2", got)
	}
}

func TestListCloudflareMediaPlatformResourcesPaginatesFullV4PagesWithoutTotalPages(t *testing.T) {
	api := newCloudflareMediaPlatformTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/account-123/pipelines/v1/pipelines" {
			t.Fatalf("path = %q, want /accounts/account-123/pipelines/v1/pipelines", r.URL.Path)
		}
		if cursor := r.URL.Query().Get("cursor"); cursor != "" {
			t.Fatalf("cursor query = %q, want empty", cursor)
		}

		switch r.URL.Query().Get("page") {
		case "1":
			writeCloudflareMediaPlatformTestResponse(t, w, cloudflareMediaPlatformTestResources("pipeline", 50), map[string]interface{}{
				"page":     1,
				"per_page": 50,
			})
		case "2":
			writeCloudflareMediaPlatformTestResponse(t, w, cloudflareMediaPlatformTestResources("pipeline-final", 1), map[string]interface{}{
				"page":     2,
				"per_page": 50,
			})
		default:
			t.Fatalf("page query = %q, want 1 or 2", r.URL.Query().Get("page"))
		}
	}))

	resources, err := listCloudflareMediaPlatformResources(context.Background(), api, "/accounts/account-123/pipelines/v1/pipelines")
	if err != nil {
		t.Fatalf("listCloudflareMediaPlatformResources() error = %v", err)
	}
	if len(resources) != 51 {
		t.Fatalf("resource count = %d, want 51", len(resources))
	}
	if got := cloudflareMediaPlatformString(resources[0], "id"); got != "pipeline-1" {
		t.Fatalf("first id = %q, want pipeline-1", got)
	}
	if got := cloudflareMediaPlatformString(resources[50], "id"); got != "pipeline-final-1" {
		t.Fatalf("last id = %q, want pipeline-final-1", got)
	}
}

func TestListCloudflareMediaPlatformResourcesHandlesEmptyResult(t *testing.T) {
	api := newCloudflareMediaPlatformTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeCloudflareMediaPlatformTestResponse(t, w, nil, nil)
	}))

	resources, err := listCloudflareMediaPlatformResources(context.Background(), api, "/accounts/account-123/pipelines/v1/pipelines")
	if err != nil {
		t.Fatalf("listCloudflareMediaPlatformResources() error = %v", err)
	}
	if len(resources) != 0 {
		t.Fatalf("resource count = %d, want 0", len(resources))
	}
}

func TestListCloudflareImageVariantResourcesHandlesMapResponse(t *testing.T) {
	api := newCloudflareMediaPlatformTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/account-123/images/v1/variants" {
			t.Fatalf("path = %q, want /accounts/account-123/images/v1/variants", r.URL.Path)
		}
		writeCloudflareMediaPlatformTestResponse(t, w, map[string]interface{}{
			"variants": map[string]interface{}{
				"thumb": map[string]interface{}{
					"options": map[string]interface{}{
						"fit":      "cover",
						"height":   320,
						"metadata": "none",
						"width":    640,
					},
				},
			},
		}, nil)
	}))

	variants, err := listCloudflareImageVariantResources(context.Background(), api, "account-123")
	if err != nil {
		t.Fatalf("listCloudflareImageVariantResources() error = %v", err)
	}
	if len(variants) != 1 {
		t.Fatalf("variant count = %d, want 1", len(variants))
	}
	if got := cloudflareMediaPlatformString(variants[0], "id"); got != "thumb" {
		t.Fatalf("variant id = %q, want thumb", got)
	}
}

func TestAppendMediaPlatformResourcesDiscoversSupportedResources(t *testing.T) {
	api := newCloudflareMediaPlatformTestAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/accounts/account-123/images/v1/variants":
			writeCloudflareMediaPlatformTestResponse(t, w, map[string]interface{}{
				"variants": map[string]interface{}{
					"thumb": map[string]interface{}{
						"options": map[string]interface{}{
							"fit":      "cover",
							"height":   320,
							"metadata": "none",
							"width":    640,
						},
					},
				},
			}, nil)
		case "/accounts/account-123/pipelines/v1/pipelines":
			writeCloudflareMediaPlatformTestResponse(t, w, []map[string]string{{
				"id":   "pipeline-123",
				"name": "events",
				"sql":  "SELECT * FROM stream",
			}}, nil)
		case "/accounts/account-123/pipelines/v1/streams":
			writeCloudflareMediaPlatformTestResponse(t, w, []map[string]string{{
				"id":   "stream-123",
				"name": "events",
			}}, nil)
		default:
			t.Fatalf("unexpected discovery path %q", r.URL.Path)
		}
	}))

	g := &MediaPlatformGenerator{}
	if err := g.appendMediaPlatformResources(context.Background(), api, "account-123"); err != nil {
		t.Fatalf("appendMediaPlatformResources() error = %v", err)
	}
	if len(g.Resources) != 3 {
		t.Fatalf("resource count = %d, want 3", len(g.Resources))
	}
	got := map[string]string{}
	for _, resource := range g.Resources {
		got[resource.InstanceInfo.Type] = resource.InstanceState.Meta["import_id"].(string)
	}
	for resourceType, wantImportID := range map[string]string{
		"cloudflare_image_variant":   "account-123/thumb",
		"cloudflare_pipeline":        "account-123/pipeline-123",
		"cloudflare_pipeline_stream": "account-123/stream-123",
	} {
		if got[resourceType] != wantImportID {
			t.Fatalf("%s import ID = %q, want %q", resourceType, got[resourceType], wantImportID)
		}
	}
}

func TestMediaPlatformUnsupportedResourcesMetadata(t *testing.T) {
	data, err := os.ReadFile("unsupported_resources.json")
	if err != nil {
		t.Fatalf("read unsupported resources: %v", err)
	}
	var metadata cloudflareUnsupportedResourcesFile
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("decode unsupported resources: %v", err)
	}
	seen := map[string]bool{}
	for _, resource := range metadata.Resources {
		seen[resource.Resource] = true
	}
	for _, resource := range []string{
		"cloudflare_ai_gateway",
		"cloudflare_ai_search_instance",
		"cloudflare_calls_sfu_app",
		"cloudflare_calls_turn_app",
		"cloudflare_image",
		"cloudflare_stream",
		"cloudflare_stream_audio_track",
		"cloudflare_stream_caption_language",
		"cloudflare_stream_watermark",
	} {
		if !seen[resource] {
			t.Fatalf("unsupported metadata is missing %s", resource)
		}
	}
}

func newCloudflareMediaPlatformTestAPI(t *testing.T, handler http.Handler) *cf.API {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	api, err := cf.NewWithAPIToken(
		"test-token",
		cf.BaseURL(server.URL),
		cf.UsingRateLimit(100000),
		cf.UsingRetryPolicy(0, 0, 0),
	)
	if err != nil {
		t.Fatalf("create Cloudflare test API: %v", err)
	}
	return api
}

func cloudflareMediaPlatformTestResources(prefix string, count int) []map[string]string {
	resources := make([]map[string]string, count)
	for i := range resources {
		resources[i] = map[string]string{"id": fmt.Sprintf("%s-%d", prefix, i+1)}
	}
	return resources
}

func writeCloudflareMediaPlatformTestResponse(t *testing.T, w http.ResponseWriter, result interface{}, resultInfo interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")

	payload := map[string]interface{}{
		"success": true,
		"result":  result,
	}
	if resultInfo != nil {
		payload["result_info"] = resultInfo
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("write response: %v", err)
	}
}
