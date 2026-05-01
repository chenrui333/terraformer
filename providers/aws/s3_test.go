// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

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

	fieldsByBucket := s3BucketInlineFieldsByBucket([]terraformutils.Resource{versioning, policy})
	if _, ok := fieldsByBucket["versioned-bucket"]["versioning"]; !ok {
		t.Fatalf("versioning inline field was not tracked: %#v", fieldsByBucket)
	}
	if _, ok := fieldsByBucket["policy-bucket"]["policy"]; !ok {
		t.Fatalf("policy inline field was not tracked from resource ID fallback: %#v", fieldsByBucket)
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
