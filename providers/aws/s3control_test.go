// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	s3controltypes "github.com/aws/aws-sdk-go-v2/service/s3control/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	testS3ControlAccountID      = "123456789012"
	testS3ControlAccessPointARN = "arn:aws:s3:us-east-1:123456789012:accesspoint/core-ap"
)

func TestS3ControlAccessPointImportIDs(t *testing.T) {
	outpostsARN := "arn:aws:s3-outposts:us-east-1:123456789012:outpost/op-1234567890123456/accesspoint/core-ap"
	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "access point",
			got:  s3ControlAccessPointImportID(testS3ControlAccountID, "core-ap", testS3ControlAccessPointARN),
			want: "123456789012:core-ap",
		},
		{
			name: "outposts access point",
			got:  s3ControlAccessPointImportID(testS3ControlAccountID, "core-ap", outpostsARN),
			want: outpostsARN,
		},
		{
			name: "object lambda access point",
			got:  s3ControlObjectLambdaAccessPointImportID(testS3ControlAccountID, "core-olap"),
			want: "123456789012:core-olap",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("import ID = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestS3ControlAccessPointAPIName(t *testing.T) {
	outpostsARN := "arn:aws:s3-outposts:us-east-1:123456789012:outpost/op-1234567890123456/accesspoint/core-ap"
	if got := s3ControlAccessPointAPIName("core-ap", testS3ControlAccessPointARN); got != "core-ap" {
		t.Fatalf("regular access point API name = %q, want %q", got, "core-ap")
	}
	if got := s3ControlAccessPointAPIName("core-ap", outpostsARN); got != outpostsARN {
		t.Fatalf("outposts access point API name = %q, want %q", got, outpostsARN)
	}
}

func TestNewS3ControlAccessPointResource(t *testing.T) {
	resource, ok := newS3ControlAccessPointResource(testS3ControlAccountID, &s3control.GetAccessPointOutput{
		AccessPointArn:  aws.String(testS3ControlAccessPointARN),
		Bucket:          aws.String("core-bucket"),
		BucketAccountId: aws.String(testS3ControlAccountID),
		Name:            aws.String("core-ap"),
	})
	assertS3ControlResourceAttributes(t, resource, ok, s3ControlAccessPointResourceType, "123456789012:core-ap",
		[]string{"access_point", testS3ControlAccountID, "core-ap", testS3ControlAccessPointARN},
		map[string]string{
			"account_id":        testS3ControlAccountID,
			"bucket":            "core-bucket",
			"bucket_account_id": testS3ControlAccountID,
			"name":              "core-ap",
		})

	if _, ok := newS3ControlAccessPointResource("", &s3control.GetAccessPointOutput{
		Bucket: aws.String("core-bucket"),
		Name:   aws.String("core-ap"),
	}); ok {
		t.Fatal("access point with empty account ID should be skipped")
	}
	if _, ok := newS3ControlAccessPointResource(testS3ControlAccountID, &s3control.GetAccessPointOutput{
		Name: aws.String("core-ap"),
	}); ok {
		t.Fatal("access point with empty bucket should be skipped")
	}
	if _, ok := newS3ControlAccessPointResource(testS3ControlAccountID, &s3control.GetAccessPointOutput{
		Bucket: aws.String("core-bucket"),
	}); ok {
		t.Fatal("access point with empty name should be skipped")
	}
	if _, ok := newS3ControlAccessPointResource(testS3ControlAccountID, nil); ok {
		t.Fatal("nil access point should be skipped")
	}
}

func TestNewS3ControlOutpostsAccessPointResource(t *testing.T) {
	outpostsARN := "arn:aws:s3-outposts:us-east-1:123456789012:outpost/op-1234567890123456/accesspoint/core-ap"
	resource, ok := newS3ControlAccessPointResource(testS3ControlAccountID, &s3control.GetAccessPointOutput{
		AccessPointArn: aws.String(outpostsARN),
		Bucket:         aws.String("arn:aws:s3-outposts:us-east-1:123456789012:outpost/op-1234567890123456/bucket/core-bucket"),
		Name:           aws.String("core-ap"),
	})
	assertS3ControlResourceAttributes(t, resource, ok, s3ControlAccessPointResourceType, outpostsARN,
		[]string{"access_point", testS3ControlAccountID, "core-ap", outpostsARN},
		map[string]string{
			"account_id": testS3ControlAccountID,
			"bucket":     "arn:aws:s3-outposts:us-east-1:123456789012:outpost/op-1234567890123456/bucket/core-bucket",
			"name":       "core-ap",
		})
}

func TestNewS3ControlAccessPointPolicyResource(t *testing.T) {
	policy := `{"Version":"2012-10-17"}`
	resource, ok := newS3ControlAccessPointPolicyResource(testS3ControlAccountID, "core-ap", testS3ControlAccessPointARN, policy)
	assertS3ControlResourceAttributes(t, resource, ok, s3ControlAccessPointPolicyResourceType, "123456789012:core-ap",
		[]string{"access_point_policy", testS3ControlAccountID, "core-ap", testS3ControlAccessPointARN},
		map[string]string{
			"access_point_arn": testS3ControlAccessPointARN,
			"policy":           policy,
		})

	if _, ok := newS3ControlAccessPointPolicyResource("", "core-ap", testS3ControlAccessPointARN, policy); ok {
		t.Fatal("policy with empty account ID should be skipped")
	}
	if _, ok := newS3ControlAccessPointPolicyResource(testS3ControlAccountID, "", testS3ControlAccessPointARN, policy); ok {
		t.Fatal("policy with empty access point name should be skipped")
	}
	if _, ok := newS3ControlAccessPointPolicyResource(testS3ControlAccountID, "core-ap", "", policy); ok {
		t.Fatal("policy with empty access point ARN should be skipped")
	}
	if _, ok := newS3ControlAccessPointPolicyResource(testS3ControlAccountID, "core-ap", testS3ControlAccessPointARN, ""); ok {
		t.Fatal("policy with empty policy body should be skipped")
	}
}

func TestNewS3ControlObjectLambdaAccessPointResource(t *testing.T) {
	configuration := testS3ControlObjectLambdaConfiguration()
	objectLambdaARN := "arn:aws:s3-object-lambda:us-east-1:123456789012:accesspoint/core-olap"
	resource, ok := newS3ControlObjectLambdaAccessPointResource(testS3ControlAccountID, s3controltypes.ObjectLambdaAccessPoint{
		Name:                       aws.String("core-olap"),
		ObjectLambdaAccessPointArn: aws.String(objectLambdaARN),
	}, configuration)
	assertS3ControlResourceAttributes(t, resource, ok, s3ControlObjectLambdaAccessPointResourceType, "123456789012:core-olap",
		[]string{"object_lambda_access_point", testS3ControlAccountID, "core-olap", objectLambdaARN},
		map[string]string{
			"account_id": testS3ControlAccountID,
			"name":       "core-olap",
		})

	if _, ok := newS3ControlObjectLambdaAccessPointResource("", s3controltypes.ObjectLambdaAccessPoint{Name: aws.String("core-olap")}, configuration); ok {
		t.Fatal("object lambda access point with empty account ID should be skipped")
	}
	if _, ok := newS3ControlObjectLambdaAccessPointResource(testS3ControlAccountID, s3controltypes.ObjectLambdaAccessPoint{}, configuration); ok {
		t.Fatal("object lambda access point with empty name should be skipped")
	}
	if _, ok := newS3ControlObjectLambdaAccessPointResource(testS3ControlAccountID, s3controltypes.ObjectLambdaAccessPoint{Name: aws.String("core-olap")}, nil); ok {
		t.Fatal("object lambda access point with empty configuration should be skipped")
	}
}

func TestNewS3ControlObjectLambdaAccessPointPolicyResource(t *testing.T) {
	policy := `{"Version":"2012-10-17"}`
	resource, ok := newS3ControlObjectLambdaAccessPointPolicyResource(testS3ControlAccountID, "core-olap", policy)
	assertS3ControlResourceAttributes(t, resource, ok, s3ControlObjectLambdaAccessPointPolicyResourceType, "123456789012:core-olap",
		[]string{"object_lambda_access_point_policy", testS3ControlAccountID, "core-olap"},
		map[string]string{
			"account_id": testS3ControlAccountID,
			"name":       "core-olap",
			"policy":     policy,
		})

	if _, ok := newS3ControlObjectLambdaAccessPointPolicyResource("", "core-olap", policy); ok {
		t.Fatal("object lambda policy with empty account ID should be skipped")
	}
	if _, ok := newS3ControlObjectLambdaAccessPointPolicyResource(testS3ControlAccountID, "", policy); ok {
		t.Fatal("object lambda policy with empty name should be skipped")
	}
	if _, ok := newS3ControlObjectLambdaAccessPointPolicyResource(testS3ControlAccountID, "core-olap", ""); ok {
		t.Fatal("object lambda policy with empty policy body should be skipped")
	}
}

func TestS3ControlObjectLambdaAccessPointImportable(t *testing.T) {
	tests := []struct {
		name          string
		configuration *s3control.GetAccessPointConfigurationForObjectLambdaOutput
		want          bool
	}{
		{name: "valid", configuration: testS3ControlObjectLambdaConfiguration(), want: true},
		{name: "nil output", want: false},
		{name: "nil configuration", configuration: &s3control.GetAccessPointConfigurationForObjectLambdaOutput{}, want: false},
		{name: "empty supporting access point", configuration: &s3control.GetAccessPointConfigurationForObjectLambdaOutput{
			Configuration: &s3controltypes.ObjectLambdaConfiguration{
				TransformationConfigurations: []s3controltypes.ObjectLambdaTransformationConfiguration{{}},
			},
		}, want: false},
		{name: "empty transformations", configuration: &s3control.GetAccessPointConfigurationForObjectLambdaOutput{
			Configuration: &s3controltypes.ObjectLambdaConfiguration{
				SupportingAccessPoint: aws.String(testS3ControlAccessPointARN),
			},
		}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s3ControlObjectLambdaAccessPointImportable(tt.configuration); got != tt.want {
				t.Fatalf("importable = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestS3ControlResourceNameAvoidsSanitizedCollisions(t *testing.T) {
	left := terraformutils.TfSanitize(s3ControlResourceName("access_point", testS3ControlAccountID, "a_b", "c"))
	right := terraformutils.TfSanitize(s3ControlResourceName("access_point", testS3ControlAccountID, "a", "b_c"))
	if left == right {
		t.Fatalf("resource names collide: %q", left)
	}
}

func TestS3ControlResourceNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "typed not found", err: &s3controltypes.NotFoundException{}, want: true},
		{name: "no such access point", err: &smithy.GenericAPIError{Code: "NoSuchAccessPoint"}, want: true},
		{name: "no such policy", err: &smithy.GenericAPIError{Code: "NoSuchAccessPointPolicy"}, want: true},
		{name: "wrapped not found", err: errors.Join(errors.New("lookup failed"), &s3controltypes.NotFoundException{}), want: true},
		{name: "access denied", err: &smithy.GenericAPIError{Code: "AccessDenied"}, want: false},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s3ControlResourceNotFound(tt.err); got != tt.want {
				t.Fatalf("not found = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestS3ControlDirectoryBucketPolicyStatusUnsupported(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "method not allowed", err: &smithy.GenericAPIError{Code: "MethodNotAllowed"}, want: true},
		{name: "unknown error", err: &smithy.GenericAPIError{Code: "UnknownError"}, want: true},
		{name: "access denied", err: &smithy.GenericAPIError{Code: "AccessDenied"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s3ControlDirectoryBucketPolicyStatusUnsupported(tt.err); got != tt.want {
				t.Fatalf("unsupported = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestS3ControlPostConvertHookWrapsSplitPolicies(t *testing.T) {
	accessPoint, ok := newS3ControlAccessPointResource(testS3ControlAccountID, &s3control.GetAccessPointOutput{
		AccessPointArn: aws.String(testS3ControlAccessPointARN),
		Bucket:         aws.String("core-bucket"),
		Name:           aws.String("core-ap"),
	})
	if !ok {
		t.Fatal("access point should be importable")
	}
	accessPoint.Item = map[string]interface{}{
		"name":   "core-ap",
		"policy": `{"Resource":"${aws:username}"}`,
	}
	accessPoint.InstanceState.Attributes["policy"] = `{"Resource":"${aws:username}"}`

	policyResource, ok := newS3ControlAccessPointPolicyResource(testS3ControlAccountID, "core-ap", testS3ControlAccessPointARN, `{"Resource":"${aws:username}"}`)
	if !ok {
		t.Fatal("access point policy should be importable")
	}
	policyResource.Item = map[string]interface{}{
		"policy": `{"Resource":"${aws:username}"}`,
	}

	generator := S3ControlGenerator{
		AWSService: AWSService{
			Service: terraformutils.Service{
				Resources: []terraformutils.Resource{accessPoint, policyResource},
			},
		},
	}
	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}
	if _, ok := generator.Resources[0].Item["policy"]; ok {
		t.Fatal("inline access point policy was not removed when split policy resource exists")
	}
	if _, ok := generator.Resources[0].InstanceState.Attributes["policy"]; ok {
		t.Fatal("inline access point policy flatmap attribute was not removed")
	}
	want := "<<POLICY\n{\"Resource\":\"$" + "$" + "{aws:username}\"}\nPOLICY"
	if got := generator.Resources[1].Item["policy"]; got != want {
		t.Fatalf("split policy = %q, want %q", got, want)
	}
}

func assertS3ControlResourceAttributes(t *testing.T, resource terraformutils.Resource, ok bool, resourceType, resourceID string, nameParts []string, attributes map[string]string) {
	t.Helper()
	if !ok {
		t.Fatal("resource was skipped")
	}
	if resource.InstanceInfo.Type != resourceType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, resourceType)
	}
	if resource.InstanceState.ID != resourceID {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, resourceID)
	}
	for name, want := range attributes {
		if got := resource.InstanceState.Attributes[name]; got != want {
			t.Fatalf("attribute %q = %q, want %q", name, got, want)
		}
	}
	wantName := terraformutils.TfSanitize(s3ControlResourceName(nameParts...))
	if resource.ResourceName != wantName {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, wantName)
	}
}

func testS3ControlObjectLambdaConfiguration() *s3control.GetAccessPointConfigurationForObjectLambdaOutput {
	return &s3control.GetAccessPointConfigurationForObjectLambdaOutput{
		Configuration: &s3controltypes.ObjectLambdaConfiguration{
			SupportingAccessPoint: aws.String(testS3ControlAccessPointARN),
			TransformationConfigurations: []s3controltypes.ObjectLambdaTransformationConfiguration{
				{},
			},
		},
	}
}
