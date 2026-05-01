// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

var S3AllowEmptyValues = []string{"tags."}

var S3AdditionalFields = map[string]interface{}{}

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
	for i, resource := range g.Resources {
		if resource.InstanceInfo.Type == "aws_s3_bucket" {
			if val, ok := g.Resources[i].Item["acl"]; ok && val == "private" {
				delete(g.Resources[i].Item, "acl")
			}
			if val, ok := g.Resources[i].Item["policy"]; ok {
				g.Resources[i].Item["policy"] = fmt.Sprintf(`<<POLICY
%s
POLICY`, g.escapeAwsInterpolation(val.(string)))
			}
		}
	}
	return nil
}
