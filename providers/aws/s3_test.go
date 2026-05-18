// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestAddS3BucketConfigurationResource(t *testing.T) {
	var resources []terraformutils.Resource
	addS3BucketConfigurationResource(&resources, "example-bucket", "aws_s3_bucket_versioning")
	if len(resources) != 1 {
		t.Fatalf("len(resources) = %d, want 1", len(resources))
	}
	resource := resources[0]
	if resource.InstanceState.ID != "example-bucket" {
		t.Fatalf("InstanceState.ID = %q, want %q", resource.InstanceState.ID, "example-bucket")
	}
	if resource.InstanceInfo.Type != "aws_s3_bucket_versioning" {
		t.Fatalf("InstanceInfo.Type = %q, want %q", resource.InstanceInfo.Type, "aws_s3_bucket_versioning")
	}
	if resource.InstanceState.Attributes["bucket"] != "example-bucket" {
		t.Fatalf("bucket attribute = %q, want %q", resource.InstanceState.Attributes["bucket"], "example-bucket")
	}
}

func TestAddS3BucketNamedConfigurationResource(t *testing.T) {
	var resources []terraformutils.Resource
	addS3BucketNamedConfigurationResource(&resources, "example-bucket", "EntireBucket", s3BucketMetricResourceType)
	addS3BucketNamedConfigurationResource(&resources, "", "skip", s3BucketMetricResourceType)
	addS3BucketNamedConfigurationResource(&resources, "example-bucket", "", s3BucketMetricResourceType)

	if len(resources) != 1 {
		t.Fatalf("len(resources) = %d, want 1", len(resources))
	}
	resource := resources[0]
	if resource.InstanceState.ID != "example-bucket:EntireBucket" {
		t.Fatalf("InstanceState.ID = %q, want %q", resource.InstanceState.ID, "example-bucket:EntireBucket")
	}
	if resource.InstanceInfo.Type != s3BucketMetricResourceType {
		t.Fatalf("InstanceInfo.Type = %q, want %q", resource.InstanceInfo.Type, s3BucketMetricResourceType)
	}
	if resource.InstanceState.Attributes["bucket"] != "example-bucket" {
		t.Fatalf("bucket attribute = %q, want example-bucket", resource.InstanceState.Attributes["bucket"])
	}
	if resource.InstanceState.Attributes["name"] != "EntireBucket" {
		t.Fatalf("name attribute = %q, want EntireBucket", resource.InstanceState.Attributes["name"])
	}
}

func TestAddS3BucketACLResource(t *testing.T) {
	var resources []terraformutils.Resource
	addS3BucketACLResource(&resources, "example-bucket", string(types.BucketCannedACLPublicRead))
	addS3BucketACLResource(&resources, "custom-bucket", "")
	addS3BucketACLResource(&resources, "", string(types.BucketCannedACLPublicRead))

	if len(resources) != 2 {
		t.Fatalf("len(resources) = %d, want 2", len(resources))
	}
	cannedACL := resources[0]
	if cannedACL.InstanceState.ID != "example-bucket,public-read" {
		t.Fatalf("canned ACL InstanceState.ID = %q, want %q", cannedACL.InstanceState.ID, "example-bucket,public-read")
	}
	if cannedACL.InstanceInfo.Type != s3BucketACLResourceType {
		t.Fatalf("canned ACL InstanceInfo.Type = %q, want %q", cannedACL.InstanceInfo.Type, s3BucketACLResourceType)
	}
	if cannedACL.InstanceState.Attributes["bucket"] != "example-bucket" {
		t.Fatalf("bucket attribute = %q, want example-bucket", cannedACL.InstanceState.Attributes["bucket"])
	}
	if cannedACL.InstanceState.Attributes["acl"] != "public-read" {
		t.Fatalf("acl attribute = %q, want public-read", cannedACL.InstanceState.Attributes["acl"])
	}

	customACL := resources[1]
	if customACL.InstanceState.ID != "custom-bucket" {
		t.Fatalf("custom ACL InstanceState.ID = %q, want custom-bucket", customACL.InstanceState.ID)
	}
	if _, ok := customACL.InstanceState.Attributes["acl"]; ok {
		t.Fatalf("custom ACL resource unexpectedly set canned acl attribute: %#v", customACL.InstanceState.Attributes)
	}
}

func TestS3BucketMetricConfigurationsPaginate(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/example-bucket" {
			t.Errorf("path = %q, want /example-bucket", r.URL.Path)
		}
		if _, ok := r.URL.Query()["metrics"]; !ok {
			t.Errorf("missing metrics query in %q", r.URL.RawQuery)
		}
		requests++
		switch requests {
		case 1:
			if got := r.URL.Query().Get("continuation-token"); got != "" {
				t.Errorf("first continuation-token = %q, want empty", got)
			}
			writeS3XML(w, http.StatusOK, "<ListMetricsConfigurationsResult xmlns=\"http://s3.amazonaws.com/doc/2006-03-01/\"><IsTruncated>true</IsTruncated><NextContinuationToken>page-2</NextContinuationToken><MetricsConfiguration><Id>EntireBucket</Id></MetricsConfiguration></ListMetricsConfigurationsResult>")
		case 2:
			if got := r.URL.Query().Get("continuation-token"); got != "page-2" {
				t.Errorf("second continuation-token = %q, want page-2", got)
			}
			writeS3XML(w, http.StatusOK, "<ListMetricsConfigurationsResult xmlns=\"http://s3.amazonaws.com/doc/2006-03-01/\"><IsTruncated>false</IsTruncated><MetricsConfiguration><Id>LogsOnly</Id></MetricsConfiguration></ListMetricsConfigurationsResult>")
		default:
			t.Errorf("unexpected metrics request %d", requests)
			writeS3XML(w, http.StatusOK, "<ListMetricsConfigurationsResult xmlns=\"http://s3.amazonaws.com/doc/2006-03-01/\"><IsTruncated>false</IsTruncated></ListMetricsConfigurationsResult>")
		}
	}))
	t.Cleanup(server.Close)

	var resources []terraformutils.Resource
	generator := &S3Generator{}
	generator.addBucketMetricConfigurations(newTestS3Client(server), &resources, "example-bucket")

	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	wantIDs := []string{"example-bucket:EntireBucket", "example-bucket:LogsOnly"}
	if len(resources) != len(wantIDs) {
		t.Fatalf("len(resources) = %d, want %d", len(resources), len(wantIDs))
	}
	for i, wantID := range wantIDs {
		if got := resources[i].InstanceState.ID; got != wantID {
			t.Fatalf("resource[%d] ID = %q, want %q", i, got, wantID)
		}
	}
}

func TestS3BucketACLImportable(t *testing.T) {
	ownerID := "owner-canonical-id"
	tests := []struct {
		name   string
		output *s3.GetBucketAclOutput
		want   bool
	}{
		{name: "nil", want: false},
		{
			name: "default private",
			output: &s3.GetBucketAclOutput{
				Owner: &types.Owner{ID: aws.String(ownerID)},
				Grants: []types.Grant{{
					Grantee:    &types.Grantee{ID: aws.String(ownerID), Type: types.TypeCanonicalUser},
					Permission: types.PermissionFullControl,
				}},
			},
			want: false,
		},
		{
			name: "non-owner grant",
			output: &s3.GetBucketAclOutput{
				Owner: &types.Owner{ID: aws.String(ownerID)},
				Grants: []types.Grant{{
					Grantee:    &types.Grantee{ID: aws.String("other-canonical-id"), Type: types.TypeCanonicalUser},
					Permission: types.PermissionRead,
				}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s3BucketACLImportable(tt.output); got != tt.want {
				t.Fatalf("s3BucketACLImportable() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestS3BucketCannedACL(t *testing.T) {
	ownerID := "owner-canonical-id"
	tests := []struct {
		name   string
		grants []types.Grant
		want   string
	}{
		{
			name: "public read",
			grants: []types.Grant{
				s3OwnerGrant(ownerID),
				s3GroupGrant("/groups/global/AllUsers", types.PermissionRead),
			},
			want: string(types.BucketCannedACLPublicRead),
		},
		{
			name: "public read write",
			grants: []types.Grant{
				s3GroupGrant("/groups/global/AllUsers", types.PermissionWrite),
				s3OwnerGrant(ownerID),
				s3GroupGrant("/groups/global/AllUsers", types.PermissionRead),
			},
			want: string(types.BucketCannedACLPublicReadWrite),
		},
		{
			name: "authenticated read",
			grants: []types.Grant{
				s3OwnerGrant(ownerID),
				s3GroupGrant("/groups/global/AuthenticatedUsers", types.PermissionRead),
			},
			want: string(types.BucketCannedACLAuthenticatedRead),
		},
		{
			name: "log delivery write",
			grants: []types.Grant{
				s3OwnerGrant(ownerID),
				s3GroupGrant("/groups/s3/LogDelivery", types.PermissionWrite),
				s3GroupGrant("/groups/s3/LogDelivery", types.PermissionReadAcp),
			},
			want: s3BucketCannedACLLogDeliveryWrite,
		},
		{
			name: "custom grant",
			grants: []types.Grant{
				s3OwnerGrant(ownerID),
				{
					Grantee:    &types.Grantee{ID: aws.String("other-canonical-id"), Type: types.TypeCanonicalUser},
					Permission: types.PermissionRead,
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &s3.GetBucketAclOutput{
				Owner:  &types.Owner{ID: aws.String(ownerID)},
				Grants: tt.grants,
			}
			if got := s3BucketCannedACL(output); got != tt.want {
				t.Fatalf("s3BucketCannedACL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestS3BucketNotificationConfigured(t *testing.T) {
	tests := []struct {
		name   string
		output *s3.GetBucketNotificationConfigurationOutput
		want   bool
	}{
		{name: "nil", want: false},
		{name: "empty", output: &s3.GetBucketNotificationConfigurationOutput{}, want: false},
		{name: "eventbridge", output: &s3.GetBucketNotificationConfigurationOutput{EventBridgeConfiguration: &types.EventBridgeConfiguration{}}, want: true},
		{name: "lambda", output: &s3.GetBucketNotificationConfigurationOutput{LambdaFunctionConfigurations: []types.LambdaFunctionConfiguration{{}}}, want: true},
		{name: "queue", output: &s3.GetBucketNotificationConfigurationOutput{QueueConfigurations: []types.QueueConfiguration{{}}}, want: true},
		{name: "topic", output: &s3.GetBucketNotificationConfigurationOutput{TopicConfigurations: []types.TopicConfiguration{{}}}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s3BucketNotificationConfigured(tt.output); got != tt.want {
				t.Fatalf("s3BucketNotificationConfigured() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestS3ObjectLockConfigured(t *testing.T) {
	tests := []struct {
		name   string
		output *s3.GetObjectLockConfigurationOutput
		want   bool
	}{
		{name: "nil", want: false},
		{name: "empty", output: &s3.GetObjectLockConfigurationOutput{}, want: false},
		{name: "empty config", output: &s3.GetObjectLockConfigurationOutput{ObjectLockConfiguration: &types.ObjectLockConfiguration{}}, want: false},
		{name: "enabled", output: &s3.GetObjectLockConfigurationOutput{ObjectLockConfiguration: &types.ObjectLockConfiguration{ObjectLockEnabled: types.ObjectLockEnabledEnabled}}, want: true},
		{name: "rule", output: &s3.GetObjectLockConfigurationOutput{ObjectLockConfiguration: &types.ObjectLockConfiguration{Rule: &types.ObjectLockRule{}}}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s3ObjectLockConfigured(tt.output); got != tt.want {
				t.Fatalf("s3ObjectLockConfigured() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestS3BucketWebsiteConfigured(t *testing.T) {
	tests := []struct {
		name   string
		output *s3.GetBucketWebsiteOutput
		want   bool
	}{
		{name: "nil", want: false},
		{name: "empty", output: &s3.GetBucketWebsiteOutput{}, want: false},
		{name: "index", output: &s3.GetBucketWebsiteOutput{IndexDocument: &types.IndexDocument{}}, want: true},
		{name: "error", output: &s3.GetBucketWebsiteOutput{ErrorDocument: &types.ErrorDocument{}}, want: true},
		{name: "redirect", output: &s3.GetBucketWebsiteOutput{RedirectAllRequestsTo: &types.RedirectAllRequestsTo{}}, want: true},
		{name: "routing rule", output: &s3.GetBucketWebsiteOutput{RoutingRules: []types.RoutingRule{{}}}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s3BucketWebsiteConfigured(tt.output); got != tt.want {
				t.Fatalf("s3BucketWebsiteConfigured() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestS3BucketConfigurationMissing(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "typed no such bucket", err: &types.NoSuchBucket{}, want: true},
		{name: "typed not found", err: &types.NotFound{}, want: true},
		{name: "cors missing", err: &smithy.GenericAPIError{Code: "NoSuchCORSConfiguration"}, want: true},
		{name: "lifecycle missing", err: &smithy.GenericAPIError{Code: "NoSuchLifecycleConfiguration"}, want: true},
		{name: "encryption missing", err: &smithy.GenericAPIError{Code: "ServerSideEncryptionConfigurationNotFoundError"}, want: true},
		{name: "access denied", err: &smithy.GenericAPIError{Code: "AccessDenied"}, want: false},
		{name: "generic error", err: errors.New("boom"), want: false},
		{name: "nil", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s3BucketConfigurationMissing(tt.err); got != tt.want {
				t.Fatalf("s3BucketConfigurationMissing() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestS3BucketInlineFieldsByBucket(t *testing.T) {
	versioning := terraformutils.NewResource(
		"versioned-bucket",
		"versioned-bucket",
		"aws_s3_bucket_versioning",
		"aws",
		map[string]string{"bucket": "versioned-bucket"},
		S3AllowEmptyValues,
		S3AdditionalFields,
	)
	policy := terraformutils.NewResource(
		"policy-bucket",
		"policy-bucket",
		"aws_s3_bucket_policy",
		"aws",
		nil,
		S3AllowEmptyValues,
		S3AdditionalFields,
	)
	acl := terraformutils.NewResource(
		"acl-bucket",
		"acl-bucket",
		s3BucketACLResourceType,
		"aws",
		map[string]string{"bucket": "acl-bucket"},
		S3AllowEmptyValues,
		S3AdditionalFields,
	)

	fieldsByBucket := s3BucketInlineFieldsByBucket([]terraformutils.Resource{versioning, policy, acl})
	if _, ok := fieldsByBucket["versioned-bucket"]["versioning"]; !ok {
		t.Fatalf("versioning inline field was not tracked: %#v", fieldsByBucket)
	}
	if _, ok := fieldsByBucket["policy-bucket"]["policy"]; !ok {
		t.Fatalf("policy inline field was not tracked from resource ID fallback: %#v", fieldsByBucket)
	}
	if _, ok := fieldsByBucket["acl-bucket"]["acl"]; !ok {
		t.Fatalf("acl inline field was not tracked: %#v", fieldsByBucket)
	}
}

func TestRemoveS3BucketInlineFields(t *testing.T) {
	resource := terraformutils.NewResource(
		"example-bucket",
		"example-bucket",
		"aws_s3_bucket",
		"aws",
		map[string]string{
			"bucket":                                 "example-bucket",
			"id":                                     "example-bucket",
			"versioning.#":                           "1",
			"versioning.0.enabled":                   "true",
			"lifecycle_rule.#":                       "1",
			"lifecycle_rule.0.id":                    "expire",
			"server_side_encryption_configuration.#": "1",
		},
		S3AllowEmptyValues,
		S3AdditionalFields,
	)
	resource.Item = map[string]interface{}{
		"bucket":                               "example-bucket",
		"versioning":                           []interface{}{map[string]interface{}{"enabled": true}},
		"lifecycle_rule":                       []interface{}{map[string]interface{}{"id": "expire"}},
		"server_side_encryption_configuration": []interface{}{map[string]interface{}{}},
	}

	removeS3BucketInlineFields(&resource, map[string]struct{}{
		"versioning":                           {},
		"server_side_encryption_configuration": {},
	})

	if _, ok := resource.Item["versioning"]; ok {
		t.Fatal("versioning was not removed from rendered item")
	}
	if _, ok := resource.Item["server_side_encryption_configuration"]; ok {
		t.Fatal("server_side_encryption_configuration was not removed from rendered item")
	}
	if _, ok := resource.Item["lifecycle_rule"]; !ok {
		t.Fatal("unselected lifecycle_rule was removed from rendered item")
	}
	if _, ok := resource.InstanceState.Attributes["versioning.#"]; ok {
		t.Fatal("versioning flatmap prefix was not removed")
	}
	if _, ok := resource.InstanceState.Attributes["server_side_encryption_configuration.#"]; ok {
		t.Fatal("server_side_encryption_configuration flatmap prefix was not removed")
	}
	if _, ok := resource.InstanceState.Attributes["lifecycle_rule.#"]; !ok {
		t.Fatal("unselected lifecycle_rule flatmap prefix was removed")
	}
}

func TestS3PostConvertHookRemovesComputedACLPolicyForCannedACL(t *testing.T) {
	cannedACL := terraformutils.NewResource(
		"example-bucket,public-read",
		"example-bucket",
		s3BucketACLResourceType,
		"aws",
		map[string]string{
			"bucket":                                     "example-bucket",
			"acl":                                        "public-read",
			"access_control_policy.#":                    "1",
			"access_control_policy.0.grant.#":            "1",
			"access_control_policy.0.grant.0.type":       "Group",
			"access_control_policy.0.grant.0.uri":        "http://acs.amazonaws.com/groups/global/AllUsers",
			"access_control_policy.0.grant.0.permission": "READ",
		},
		S3AllowEmptyValues,
		S3AdditionalFields,
	)
	cannedACL.Item = map[string]interface{}{
		"bucket": "example-bucket",
		"acl":    "public-read",
		"access_control_policy": []interface{}{
			map[string]interface{}{
				"grant": []interface{}{
					map[string]interface{}{
						"type":       "Group",
						"uri":        "http://acs.amazonaws.com/groups/global/AllUsers",
						"permission": "READ",
					},
				},
			},
		},
	}
	customACL := terraformutils.NewResource(
		"custom-bucket",
		"custom-bucket",
		s3BucketACLResourceType,
		"aws",
		map[string]string{
			"bucket":                                     "custom-bucket",
			"access_control_policy.#":                    "1",
			"access_control_policy.0.grant.#":            "1",
			"access_control_policy.0.grant.0.id":         "canonical-user-id",
			"access_control_policy.0.grant.0.permission": "READ",
		},
		S3AllowEmptyValues,
		S3AdditionalFields,
	)
	customACL.Item = map[string]interface{}{
		"bucket": "custom-bucket",
		"access_control_policy": []interface{}{
			map[string]interface{}{
				"grant": []interface{}{
					map[string]interface{}{
						"id":         "canonical-user-id",
						"permission": "READ",
					},
				},
			},
		},
	}
	generator := &S3Generator{}
	generator.Resources = []terraformutils.Resource{cannedACL, customACL}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}

	if got := generator.Resources[0].Item["acl"]; got != "public-read" {
		t.Fatalf("canned ACL acl = %v, want public-read", got)
	}
	if _, ok := generator.Resources[0].Item["access_control_policy"]; ok {
		t.Fatal("canned ACL access_control_policy was not removed from rendered item")
	}
	if _, ok := generator.Resources[0].InstanceState.Attributes["access_control_policy.#"]; ok {
		t.Fatal("canned ACL access_control_policy flatmap prefix was not removed")
	}
	if _, ok := generator.Resources[0].InstanceState.Attributes["access_control_policy.0.grant.0.uri"]; ok {
		t.Fatal("canned ACL nested access_control_policy flatmap field was not removed")
	}
	if _, ok := generator.Resources[1].Item["access_control_policy"]; !ok {
		t.Fatal("custom ACL access_control_policy was removed from rendered item")
	}
	if _, ok := generator.Resources[1].InstanceState.Attributes["access_control_policy.#"]; !ok {
		t.Fatal("custom ACL access_control_policy flatmap prefix was removed")
	}
}

func TestS3NamedConfigurationFilterBehavior(t *testing.T) {
	var resources []terraformutils.Resource
	addS3BucketNamedConfigurationResource(&resources, "example-bucket", "prod", s3BucketMetricResourceType)
	resource := resources[0]

	if !(&terraformutils.ResourceFilter{ServiceName: "s3_bucket_metric", FieldPath: "name", AcceptableValues: []string{"prod"}}).Filter(resource) {
		t.Fatal("expected typed name filter to keep S3 metric configuration")
	}
	if (&terraformutils.ResourceFilter{ServiceName: "s3_bucket_metric", FieldPath: "name", AcceptableValues: []string{"dev"}}).Filter(resource) {
		t.Fatal("expected typed name filter to drop non-matching S3 metric configuration")
	}
}

func TestS3UnsupportedResourceEntries(t *testing.T) {
	data, err := os.ReadFile("unsupported_resources.json")
	if err != nil {
		t.Fatalf("read unsupported resources: %v", err)
	}
	var unsupported map[string]interface{}
	if err := json.Unmarshal(data, &unsupported); err != nil {
		t.Fatalf("decode unsupported resources: %v", err)
	}
	entries, ok := unsupported["resources"].([]interface{})
	if !ok {
		t.Fatal("unsupported resources file is missing resources list")
	}

	found := map[string]bool{
		"aws_s3_bucket_object": false,
		"aws_s3_object":        false,
		"aws_s3_object_copy":   false,
	}
	for _, rawEntry := range entries {
		entry, ok := rawEntry.(map[string]interface{})
		if !ok {
			t.Fatalf("unsupported resource entry has unexpected type %T", rawEntry)
		}
		resource, _ := entry["resource"].(string)
		if _, ok := found[resource]; !ok {
			continue
		}
		found[resource] = true
		if serviceFamily, _ := entry["service_family"].(string); serviceFamily != "s3" {
			t.Fatalf("%s service family = %q, want s3", resource, serviceFamily)
		}
		if status, _ := entry["status"].(string); status != "unsupported" {
			t.Fatalf("%s status = %q, want unsupported", resource, status)
		}
		references, _ := entry["references"].([]interface{})
		reason, _ := entry["reason"].(string)
		evidence, _ := entry["evidence"].(string)
		if reason == "" || evidence == "" || len(references) == 0 {
			t.Fatalf("%s unsupported entry is missing reason, evidence, or references", resource)
		}
	}
	for resource, ok := range found {
		if !ok {
			t.Fatalf("%s unsupported entry was not found", resource)
		}
	}
}

func newTestS3Client(server *httptest.Server) *s3.Client {
	config := aws.Config{
		Region:           "us-east-1",
		Credentials:      credentials.NewStaticCredentialsProvider("test", "test", ""),
		HTTPClient:       server.Client(),
		RetryMaxAttempts: 1,
	}
	return s3.NewFromConfig(config, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(server.URL)
		options.UsePathStyle = true
	})
}

func writeS3XML(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func s3OwnerGrant(ownerID string) types.Grant {
	return types.Grant{
		Grantee:    &types.Grantee{ID: aws.String(ownerID), Type: types.TypeCanonicalUser},
		Permission: types.PermissionFullControl,
	}
}

func s3GroupGrant(uriSuffix string, permission types.Permission) types.Grant {
	return types.Grant{
		Grantee:    &types.Grantee{Type: types.TypeGroup, URI: aws.String("http://acs.amazonaws.com" + uriSuffix)},
		Permission: permission,
	}
}
