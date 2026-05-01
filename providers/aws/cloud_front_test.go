// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestCloudFrontResourceName(t *testing.T) {
	if got, want := cloudFrontResourceName("resource-id", "friendly-name"), "friendly-name"; got != want {
		t.Fatalf("cloudFrontResourceName() = %q, want %q", got, want)
	}
	if got, want := cloudFrontResourceName("resource-id", ""), "resource-id"; got != want {
		t.Fatalf("cloudFrontResourceName() = %q, want %q", got, want)
	}
}

func TestCloudFrontResourceMissing(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "generic", err: errors.New("boom"), want: false},
		{name: "distribution missing", err: &types.NoSuchDistribution{}, want: true},
		{name: "monitoring subscription missing", err: &types.NoSuchMonitoringSubscription{}, want: true},
		{name: "origin access control missing", err: &types.NoSuchOriginAccessControl{}, want: true},
		{name: "resource missing", err: &types.NoSuchResource{}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cloudFrontResourceMissing(tt.err); got != tt.want {
				t.Fatalf("cloudFrontResourceMissing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCloudFrontPostConvertHookLinksDistributionReferences(t *testing.T) {
	distribution := terraformutils.NewSimpleResource("distribution-1", "distribution-1", "aws_cloudfront_distribution", "aws", cloudFrontAllowEmptyValues)
	distribution.Item = map[string]interface{}{
		"default_cache_behavior": []interface{}{
			map[string]interface{}{
				"cache_policy_id":            "cache-1",
				"origin_request_policy_id":   "origin-request-1",
				"response_headers_policy_id": "response-headers-1",
			},
		},
		"ordered_cache_behavior": []interface{}{
			map[string]interface{}{
				"cache_policy_id":            "cache-1",
				"origin_request_policy_id":   "origin-request-1",
				"response_headers_policy_id": "response-headers-1",
			},
		},
		"origin": []interface{}{
			map[string]interface{}{
				"origin_access_control_id": "origin-access-control-1",
			},
		},
	}

	cachePolicy := terraformutils.NewSimpleResource("cache-1", "cache-1", "aws_cloudfront_cache_policy", "aws", cloudFrontAllowEmptyValues)
	originRequestPolicy := terraformutils.NewSimpleResource("origin-request-1", "origin-request-1", "aws_cloudfront_origin_request_policy", "aws", cloudFrontAllowEmptyValues)
	responseHeadersPolicy := terraformutils.NewSimpleResource("response-headers-1", "response-headers-1", "aws_cloudfront_response_headers_policy", "aws", cloudFrontAllowEmptyValues)
	originAccessControl := terraformutils.NewSimpleResource("origin-access-control-1", "origin-access-control-1", "aws_cloudfront_origin_access_control", "aws", cloudFrontAllowEmptyValues)

	g := &CloudFrontGenerator{}
	g.Resources = []terraformutils.Resource{
		distribution,
		cachePolicy,
		originRequestPolicy,
		responseHeadersPolicy,
		originAccessControl,
	}
	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}

	defaultBehavior := g.Resources[0].Item["default_cache_behavior"].([]interface{})[0].(map[string]interface{})
	orderedBehavior := g.Resources[0].Item["ordered_cache_behavior"].([]interface{})[0].(map[string]interface{})
	origin := g.Resources[0].Item["origin"].([]interface{})[0].(map[string]interface{})

	wantCacheRef := "$" + "{aws_cloudfront_cache_policy." + cachePolicy.ResourceName + ".id}"
	wantOriginRequestRef := "$" + "{aws_cloudfront_origin_request_policy." + originRequestPolicy.ResourceName + ".id}"
	wantResponseHeadersRef := "$" + "{aws_cloudfront_response_headers_policy." + responseHeadersPolicy.ResourceName + ".id}"
	wantOriginAccessControlRef := "$" + "{aws_cloudfront_origin_access_control." + originAccessControl.ResourceName + ".id}"

	if got := defaultBehavior["cache_policy_id"]; got != wantCacheRef {
		t.Fatalf("default cache_policy_id = %q, want %q", got, wantCacheRef)
	}
	if got := defaultBehavior["origin_request_policy_id"]; got != wantOriginRequestRef {
		t.Fatalf("default origin_request_policy_id = %q, want %q", got, wantOriginRequestRef)
	}
	if got := defaultBehavior["response_headers_policy_id"]; got != wantResponseHeadersRef {
		t.Fatalf("default response_headers_policy_id = %q, want %q", got, wantResponseHeadersRef)
	}
	if got := orderedBehavior["cache_policy_id"]; got != wantCacheRef {
		t.Fatalf("ordered cache_policy_id = %q, want %q", got, wantCacheRef)
	}
	if got := origin["origin_access_control_id"]; got != wantOriginAccessControlRef {
		t.Fatalf("origin_access_control_id = %q, want %q", got, wantOriginAccessControlRef)
	}
}
