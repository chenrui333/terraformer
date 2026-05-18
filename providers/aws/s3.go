// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

var S3AllowEmptyValues = []string{"tags."}

var S3AdditionalFields = map[string]interface{}{}

const (
	s3BucketACLResourceType                             = "aws_s3_bucket_acl"
	s3BucketAnalyticsConfigurationResourceType          = "aws_s3_bucket_analytics_configuration"
	s3BucketIntelligentTieringConfigurationResourceType = "aws_s3_bucket_intelligent_tiering_configuration"
	s3BucketInventoryResourceType                       = "aws_s3_bucket_inventory"
	s3BucketMetricResourceType                          = "aws_s3_bucket_metric"
	s3BucketNamedConfigurationIDSeparator               = ":"
)

var s3BucketSplitResourceInlineFields = map[string][]string{
	s3BucketACLResourceType:                              {"acl", "grant"},
	"aws_s3_bucket_accelerate_configuration":             {"acceleration_status"},
	"aws_s3_bucket_cors_configuration":                   {"cors_rule"},
	"aws_s3_bucket_lifecycle_configuration":              {"lifecycle_rule"},
	"aws_s3_bucket_logging":                              {"logging"},
	"aws_s3_bucket_object_lock_configuration":            {"object_lock_configuration", "object_lock_enabled"},
	"aws_s3_bucket_policy":                               {"policy"},
	"aws_s3_bucket_replication_configuration":            {"replication_configuration"},
	"aws_s3_bucket_request_payment_configuration":        {"request_payer"},
	"aws_s3_bucket_server_side_encryption_configuration": {"server_side_encryption_configuration"},
	"aws_s3_bucket_versioning":                           {"versioning"},
	"aws_s3_bucket_website_configuration":                {"website"},
}

type S3Generator struct {
	AWSService
}

// createResources iterate on all buckets
// for each bucket we check region and choose only bucket from set region
// for each bucket try get bucket policy, if policy exist create additional NewTerraformResource for policy
func (g *S3Generator) createResources(config aws.Config, buckets *s3.ListBucketsOutput, region string) []terraformutils.Resource {
	var resources []terraformutils.Resource
	svc := s3.NewFromConfig(config)
	for _, bucket := range buckets.Buckets {
		resourceName := StringValue(bucket.Name)
		location, err := svc.GetBucketLocation(context.TODO(), &s3.GetBucketLocationInput{Bucket: bucket.Name})
		if err != nil {
			log.Println(err)
			continue
		}
		// check if bucket in region
		constraintString := string(location.LocationConstraint)
		if constraintString == region || (constraintString == "" && region == "us-east-1") {
			attributes := map[string]string{
				"force_destroy": "false",
				"acl":           "private",
			}
			g.addBucketConfigurationResources(svc, &resources, resourceName)
			// try get policy
			var policy *s3.GetBucketPolicyOutput
			policy, err = svc.GetBucketPolicy(context.TODO(), &s3.GetBucketPolicyInput{
				Bucket: bucket.Name,
			})

			if err == nil && policy.Policy != nil {
				attributes["policy"] = *policy.Policy
				resources = append(resources, terraformutils.NewResource(
					resourceName,
					resourceName,
					"aws_s3_bucket_policy",
					"aws",
					nil,
					S3AllowEmptyValues,
					S3AdditionalFields))
			}
			resources = append(resources, terraformutils.NewResource(
				resourceName,
				resourceName,
				"aws_s3_bucket",
				"aws",
				attributes,
				S3AllowEmptyValues,
				S3AdditionalFields))
		}
	}
	return resources
}

func (g *S3Generator) addBucketConfigurationResources(svc *s3.Client, resources *[]terraformutils.Resource, bucketName string) {
	if bucketName == "" {
		return
	}
	g.addBucketACLResource(svc, resources, bucketName)
	if output, err := svc.GetBucketAccelerateConfiguration(context.TODO(), &s3.GetBucketAccelerateConfigurationInput{Bucket: &bucketName}); err == nil {
		if output.Status == types.BucketAccelerateStatusEnabled {
			addS3BucketConfigurationResource(resources, bucketName, "aws_s3_bucket_accelerate_configuration")
		}
	} else {
		logS3OptionalBucketConfigurationError(bucketName, "aws_s3_bucket_accelerate_configuration", err)
	}
	if output, err := svc.GetBucketCors(context.TODO(), &s3.GetBucketCorsInput{Bucket: &bucketName}); err == nil {
		if len(output.CORSRules) > 0 {
			addS3BucketConfigurationResource(resources, bucketName, "aws_s3_bucket_cors_configuration")
		}
	} else {
		logS3OptionalBucketConfigurationError(bucketName, "aws_s3_bucket_cors_configuration", err)
	}
	if output, err := svc.GetBucketLifecycleConfiguration(context.TODO(), &s3.GetBucketLifecycleConfigurationInput{Bucket: &bucketName}); err == nil {
		if len(output.Rules) > 0 {
			addS3BucketConfigurationResource(resources, bucketName, "aws_s3_bucket_lifecycle_configuration")
		}
	} else {
		logS3OptionalBucketConfigurationError(bucketName, "aws_s3_bucket_lifecycle_configuration", err)
	}
	if output, err := svc.GetBucketLogging(context.TODO(), &s3.GetBucketLoggingInput{Bucket: &bucketName}); err == nil {
		if output.LoggingEnabled != nil {
			addS3BucketConfigurationResource(resources, bucketName, "aws_s3_bucket_logging")
		}
	} else {
		logS3OptionalBucketConfigurationError(bucketName, "aws_s3_bucket_logging", err)
	}
	if output, err := svc.GetBucketNotificationConfiguration(context.TODO(), &s3.GetBucketNotificationConfigurationInput{Bucket: &bucketName}); err == nil {
		if s3BucketNotificationConfigured(output) {
			addS3BucketConfigurationResource(resources, bucketName, "aws_s3_bucket_notification")
		}
	} else {
		logS3OptionalBucketConfigurationError(bucketName, "aws_s3_bucket_notification", err)
	}
	if output, err := svc.GetObjectLockConfiguration(context.TODO(), &s3.GetObjectLockConfigurationInput{Bucket: &bucketName}); err == nil {
		if s3ObjectLockConfigured(output) {
			addS3BucketConfigurationResource(resources, bucketName, "aws_s3_bucket_object_lock_configuration")
		}
	} else {
		logS3OptionalBucketConfigurationError(bucketName, "aws_s3_bucket_object_lock_configuration", err)
	}
	if output, err := svc.GetBucketOwnershipControls(context.TODO(), &s3.GetBucketOwnershipControlsInput{Bucket: &bucketName}); err == nil {
		if output.OwnershipControls != nil && len(output.OwnershipControls.Rules) > 0 {
			addS3BucketConfigurationResource(resources, bucketName, "aws_s3_bucket_ownership_controls")
		}
	} else {
		logS3OptionalBucketConfigurationError(bucketName, "aws_s3_bucket_ownership_controls", err)
	}
	if output, err := svc.GetPublicAccessBlock(context.TODO(), &s3.GetPublicAccessBlockInput{Bucket: &bucketName}); err == nil {
		if output.PublicAccessBlockConfiguration != nil {
			addS3BucketConfigurationResource(resources, bucketName, "aws_s3_bucket_public_access_block")
		}
	} else {
		logS3OptionalBucketConfigurationError(bucketName, "aws_s3_bucket_public_access_block", err)
	}
	if output, err := svc.GetBucketReplication(context.TODO(), &s3.GetBucketReplicationInput{Bucket: &bucketName}); err == nil {
		if output.ReplicationConfiguration != nil && len(output.ReplicationConfiguration.Rules) > 0 {
			addS3BucketConfigurationResource(resources, bucketName, "aws_s3_bucket_replication_configuration")
		}
	} else {
		logS3OptionalBucketConfigurationError(bucketName, "aws_s3_bucket_replication_configuration", err)
	}
	if output, err := svc.GetBucketRequestPayment(context.TODO(), &s3.GetBucketRequestPaymentInput{Bucket: &bucketName}); err == nil {
		if output.Payer == types.PayerRequester {
			addS3BucketConfigurationResource(resources, bucketName, "aws_s3_bucket_request_payment_configuration")
		}
	} else {
		logS3OptionalBucketConfigurationError(bucketName, "aws_s3_bucket_request_payment_configuration", err)
	}
	if output, err := svc.GetBucketEncryption(context.TODO(), &s3.GetBucketEncryptionInput{Bucket: &bucketName}); err == nil {
		if output.ServerSideEncryptionConfiguration != nil && len(output.ServerSideEncryptionConfiguration.Rules) > 0 {
			addS3BucketConfigurationResource(resources, bucketName, "aws_s3_bucket_server_side_encryption_configuration")
		}
	} else {
		logS3OptionalBucketConfigurationError(bucketName, "aws_s3_bucket_server_side_encryption_configuration", err)
	}
	if output, err := svc.GetBucketVersioning(context.TODO(), &s3.GetBucketVersioningInput{Bucket: &bucketName}); err == nil {
		if output.Status != "" || output.MFADelete != "" {
			addS3BucketConfigurationResource(resources, bucketName, "aws_s3_bucket_versioning")
		}
	} else {
		logS3OptionalBucketConfigurationError(bucketName, "aws_s3_bucket_versioning", err)
	}
	if output, err := svc.GetBucketWebsite(context.TODO(), &s3.GetBucketWebsiteInput{Bucket: &bucketName}); err == nil {
		if s3BucketWebsiteConfigured(output) {
			addS3BucketConfigurationResource(resources, bucketName, "aws_s3_bucket_website_configuration")
		}
	} else {
		logS3OptionalBucketConfigurationError(bucketName, "aws_s3_bucket_website_configuration", err)
	}
	g.addBucketAnalyticsConfigurations(svc, resources, bucketName)
	g.addBucketIntelligentTieringConfigurations(svc, resources, bucketName)
	g.addBucketInventoryConfigurations(svc, resources, bucketName)
	g.addBucketMetricConfigurations(svc, resources, bucketName)
}

func (g *S3Generator) addBucketACLResource(svc *s3.Client, resources *[]terraformutils.Resource, bucketName string) {
	output, err := svc.GetBucketAcl(context.TODO(), &s3.GetBucketAclInput{Bucket: &bucketName})
	if err != nil {
		logS3OptionalBucketConfigurationError(bucketName, s3BucketACLResourceType, err)
		return
	}
	if s3BucketACLImportable(output) {
		addS3BucketConfigurationResource(resources, bucketName, s3BucketACLResourceType)
	}
}

func (g *S3Generator) addBucketAnalyticsConfigurations(svc *s3.Client, resources *[]terraformutils.Resource, bucketName string) {
	var continuationToken *string
	for {
		page, err := svc.ListBucketAnalyticsConfigurations(context.TODO(), &s3.ListBucketAnalyticsConfigurationsInput{
			Bucket:            &bucketName,
			ContinuationToken: continuationToken,
		})
		if err != nil {
			logS3OptionalBucketConfigurationError(bucketName, s3BucketAnalyticsConfigurationResourceType, err)
			return
		}
		for _, configuration := range page.AnalyticsConfigurationList {
			addS3BucketNamedConfigurationResource(resources, bucketName, StringValue(configuration.Id), s3BucketAnalyticsConfigurationResourceType)
		}
		if !aws.ToBool(page.IsTruncated) || StringValue(page.NextContinuationToken) == "" {
			return
		}
		continuationToken = page.NextContinuationToken
	}
}

func (g *S3Generator) addBucketIntelligentTieringConfigurations(svc *s3.Client, resources *[]terraformutils.Resource, bucketName string) {
	var continuationToken *string
	for {
		page, err := svc.ListBucketIntelligentTieringConfigurations(context.TODO(), &s3.ListBucketIntelligentTieringConfigurationsInput{
			Bucket:            &bucketName,
			ContinuationToken: continuationToken,
		})
		if err != nil {
			logS3OptionalBucketConfigurationError(bucketName, s3BucketIntelligentTieringConfigurationResourceType, err)
			return
		}
		for _, configuration := range page.IntelligentTieringConfigurationList {
			addS3BucketNamedConfigurationResource(resources, bucketName, StringValue(configuration.Id), s3BucketIntelligentTieringConfigurationResourceType)
		}
		if !aws.ToBool(page.IsTruncated) || StringValue(page.NextContinuationToken) == "" {
			return
		}
		continuationToken = page.NextContinuationToken
	}
}

func (g *S3Generator) addBucketInventoryConfigurations(svc *s3.Client, resources *[]terraformutils.Resource, bucketName string) {
	var continuationToken *string
	for {
		page, err := svc.ListBucketInventoryConfigurations(context.TODO(), &s3.ListBucketInventoryConfigurationsInput{
			Bucket:            &bucketName,
			ContinuationToken: continuationToken,
		})
		if err != nil {
			logS3OptionalBucketConfigurationError(bucketName, s3BucketInventoryResourceType, err)
			return
		}
		for _, configuration := range page.InventoryConfigurationList {
			addS3BucketNamedConfigurationResource(resources, bucketName, StringValue(configuration.Id), s3BucketInventoryResourceType)
		}
		if !aws.ToBool(page.IsTruncated) || StringValue(page.NextContinuationToken) == "" {
			return
		}
		continuationToken = page.NextContinuationToken
	}
}

func (g *S3Generator) addBucketMetricConfigurations(svc *s3.Client, resources *[]terraformutils.Resource, bucketName string) {
	var continuationToken *string
	for {
		page, err := svc.ListBucketMetricsConfigurations(context.TODO(), &s3.ListBucketMetricsConfigurationsInput{
			Bucket:            &bucketName,
			ContinuationToken: continuationToken,
		})
		if err != nil {
			logS3OptionalBucketConfigurationError(bucketName, s3BucketMetricResourceType, err)
			return
		}
		for _, configuration := range page.MetricsConfigurationList {
			addS3BucketNamedConfigurationResource(resources, bucketName, StringValue(configuration.Id), s3BucketMetricResourceType)
		}
		if !aws.ToBool(page.IsTruncated) || StringValue(page.NextContinuationToken) == "" {
			return
		}
		continuationToken = page.NextContinuationToken
	}
}

func addS3BucketConfigurationResource(resources *[]terraformutils.Resource, bucketName, resourceType string) {
	*resources = append(*resources, terraformutils.NewResource(
		bucketName,
		bucketName,
		resourceType,
		"aws",
		map[string]string{
			"bucket": bucketName,
		},
		S3AllowEmptyValues,
		S3AdditionalFields,
	))
}

func addS3BucketNamedConfigurationResource(resources *[]terraformutils.Resource, bucketName, configurationName, resourceType string) {
	if bucketName == "" || configurationName == "" {
		return
	}
	*resources = append(*resources, terraformutils.NewResource(
		s3BucketNamedConfigurationImportID(bucketName, configurationName),
		s3BucketNamedConfigurationResourceName(resourceType, bucketName, configurationName),
		resourceType,
		"aws",
		map[string]string{
			"bucket": bucketName,
			"name":   configurationName,
		},
		S3AllowEmptyValues,
		S3AdditionalFields,
	))
}

func s3BucketNamedConfigurationImportID(bucketName, configurationName string) string {
	return strings.Join([]string{bucketName, configurationName}, s3BucketNamedConfigurationIDSeparator)
}

func s3BucketNamedConfigurationResourceName(resourceType, bucketName, configurationName string) string {
	return strings.Join([]string{strings.TrimPrefix(resourceType, s3BucketResourceTypePrefix()), bucketName, configurationName}, "_")
}

func s3BucketResourceTypePrefix() string {
	return "aws" + "_s3_bucket_"
}

func s3BucketACLImportable(output *s3.GetBucketAclOutput) bool {
	return output != nil && !s3BucketACLIsDefaultPrivate(output)
}

func s3BucketACLIsDefaultPrivate(output *s3.GetBucketAclOutput) bool {
	if output == nil || len(output.Grants) != 1 {
		return false
	}
	ownerID := StringValue(output.Owner.ID)
	grant := output.Grants[0]
	if grant.Grantee == nil || ownerID == "" {
		return false
	}
	return grant.Permission == types.PermissionFullControl &&
		grant.Grantee.Type == types.TypeCanonicalUser &&
		StringValue(grant.Grantee.ID) == ownerID
}

func s3BucketNotificationConfigured(output *s3.GetBucketNotificationConfigurationOutput) bool {
	return output != nil &&
		(output.EventBridgeConfiguration != nil ||
			len(output.LambdaFunctionConfigurations) > 0 ||
			len(output.QueueConfigurations) > 0 ||
			len(output.TopicConfigurations) > 0)
}

func s3ObjectLockConfigured(output *s3.GetObjectLockConfigurationOutput) bool {
	return output != nil &&
		output.ObjectLockConfiguration != nil &&
		(output.ObjectLockConfiguration.ObjectLockEnabled != "" || output.ObjectLockConfiguration.Rule != nil)
}

func s3BucketWebsiteConfigured(output *s3.GetBucketWebsiteOutput) bool {
	return output != nil &&
		(output.ErrorDocument != nil ||
			output.IndexDocument != nil ||
			output.RedirectAllRequestsTo != nil ||
			len(output.RoutingRules) > 0)
}

func logS3OptionalBucketConfigurationError(bucketName, resourceType string, err error) {
	if s3BucketConfigurationMissing(err) {
		return
	}
	log.Printf("skipping %s discovery for S3 bucket %s: %v", resourceType, bucketName, err)
}

func s3BucketConfigurationMissing(err error) bool {
	var noSuchBucket *types.NoSuchBucket
	if errors.As(err, &noSuchBucket) {
		return true
	}
	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return true
	}
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.ErrorCode() {
	case "NoSuchBucket",
		"NoSuchConfiguration",
		"NoSuchBucketPolicy",
		"NoSuchCORSConfiguration",
		"NoSuchLifecycleConfiguration",
		"NoSuchPublicAccessBlockConfiguration",
		"NoSuchWebsiteConfiguration",
		"ObjectLockConfigurationNotFoundError",
		"OwnershipControlsNotFoundError",
		"ReplicationConfigurationNotFoundError",
		"ServerSideEncryptionConfigurationNotFoundError":
		return true
	default:
		return false
	}
}

// Generate TerraformResources from AWS API,
// Need bucket name as ID for terraform resource
func (g *S3Generator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := s3.NewFromConfig(config)

	buckets, err := svc.ListBuckets(context.TODO(), nil)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(config, buckets, g.GetArgs()["region"].(string))
	return nil
}

// PostGenerateHook for add bucket policy json as heredoc
// support only bucket with policy
func (g *S3Generator) PostConvertHook() error {
	inlineFieldsByBucket := s3BucketInlineFieldsByBucket(g.Resources)
	for i, resource := range g.Resources {
		switch resource.InstanceInfo.Type {
		case "aws_s3_bucket":
			removeS3BucketInlineFields(&g.Resources[i], inlineFieldsByBucket[resource.InstanceState.ID])
			if val, ok := g.Resources[i].Item["acl"]; ok && val == "private" {
				delete(g.Resources[i].Item, "acl")
			}
			if val, ok := g.Resources[i].Item["policy"]; ok {
				g.Resources[i].Item["policy"] = fmt.Sprintf(`<<POLICY
%s
POLICY`, g.escapeAwsInterpolation(val.(string)))
			}
		case "aws_s3_bucket_policy":
			if val, ok := g.Resources[i].Item["policy"]; ok {
				g.Resources[i].Item["policy"] = fmt.Sprintf(`<<POLICY
%s
POLICY`, g.escapeAwsInterpolation(val.(string)))
			}
		}
	}
	return nil
}

func s3BucketInlineFieldsByBucket(resources []terraformutils.Resource) map[string]map[string]struct{} {
	fieldsByBucket := map[string]map[string]struct{}{}
	for _, resource := range resources {
		fields, ok := s3BucketSplitResourceInlineFields[resource.InstanceInfo.Type]
		if !ok {
			continue
		}
		bucketName := s3BucketNameForSplitResource(resource)
		if bucketName == "" {
			continue
		}
		if fieldsByBucket[bucketName] == nil {
			fieldsByBucket[bucketName] = map[string]struct{}{}
		}
		for _, field := range fields {
			fieldsByBucket[bucketName][field] = struct{}{}
		}
	}
	return fieldsByBucket
}

func s3BucketNameForSplitResource(resource terraformutils.Resource) string {
	if resource.InstanceState != nil && resource.InstanceState.Attributes != nil {
		if bucketName := resource.InstanceState.Attributes["bucket"]; bucketName != "" {
			return bucketName
		}
	}
	if resource.InstanceState != nil {
		return resource.InstanceState.ID
	}
	return ""
}

func removeS3BucketInlineFields(resource *terraformutils.Resource, fields map[string]struct{}) {
	if resource == nil {
		return
	}
	for field := range fields {
		if resource.Item != nil {
			delete(resource.Item, field)
		}
		if resource.InstanceState != nil {
			deleteFlatmapAttribute(resource.InstanceState.Attributes, field)
		}
	}
}

func deleteFlatmapAttribute(attributes map[string]string, field string) {
	delete(attributes, field)
	prefix := field + "."
	for key := range attributes {
		if strings.HasPrefix(key, prefix) {
			delete(attributes, key)
		}
	}
}
