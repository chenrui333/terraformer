// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3tables"
	s3tablestypes "github.com/aws/aws-sdk-go-v2/service/s3tables/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
)

const (
	testS3TablesTableBucketARN = "arn:aws:s3tables:us-east-1:123456789012:bucket/core-bucket"
	testS3TablesTableARN       = "arn:aws:s3tables:us-east-1:123456789012:bucket/core-bucket/table/12345678-1234-1234-1234-123456789012"
)

func TestS3TablesImportIDs(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "namespace",
			got:  s3TablesNamespaceImportID(testS3TablesTableBucketARN, "core_namespace"),
			want: testS3TablesTableBucketARN + ";core_namespace",
		},
		{
			name: "table",
			got:  s3TablesTableImportID(testS3TablesTableBucketARN, "core_namespace", "core_table"),
			want: testS3TablesTableBucketARN + ";core_namespace;core_table",
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

func TestNewS3TablesTableBucketResource(t *testing.T) {
	resource, ok := newS3TablesTableBucketResource(&s3tables.GetTableBucketOutput{
		Arn:  aws.String(testS3TablesTableBucketARN),
		Name: aws.String("core-bucket"),
		Type: s3tablestypes.TableBucketTypeCustomer,
	})
	assertS3TablesResourceAttributes(t, resource, ok, s3TablesTableBucketResourceType, testS3TablesTableBucketARN,
		[]string{"table_bucket", "core-bucket", testS3TablesTableBucketARN},
		map[string]string{
			"arn":           testS3TablesTableBucketARN,
			"force_destroy": "false",
			"name":          "core-bucket",
		})
	assertS3TablesPreservesID(t, resource)

	if _, ok := newS3TablesTableBucketResource(nil); ok {
		t.Fatal("nil table bucket should be skipped")
	}
	if _, ok := newS3TablesTableBucketResource(&s3tables.GetTableBucketOutput{Name: aws.String("core-bucket")}); ok {
		t.Fatal("table bucket with empty ARN should be skipped")
	}
	if _, ok := newS3TablesTableBucketResource(&s3tables.GetTableBucketOutput{Arn: aws.String(testS3TablesTableBucketARN)}); ok {
		t.Fatal("table bucket with empty name should be skipped")
	}
	if _, ok := newS3TablesTableBucketResource(&s3tables.GetTableBucketOutput{
		Arn:  aws.String(testS3TablesTableBucketARN),
		Name: aws.String("core-bucket"),
		Type: s3tablestypes.TableBucketTypeAws,
	}); ok {
		t.Fatal("AWS-managed table bucket should be skipped")
	}
}

func TestNewS3TablesNamespaceResource(t *testing.T) {
	resource, ok := newS3TablesNamespaceResource(testS3TablesTableBucketARN, &s3tables.GetNamespaceOutput{
		Namespace: []string{"core_namespace"},
	})
	assertS3TablesResourceAttributes(t, resource, ok, s3TablesNamespaceResourceType, testS3TablesTableBucketARN+";core_namespace",
		[]string{"namespace", testS3TablesTableBucketARN, "core_namespace"},
		map[string]string{
			"namespace":        "core_namespace",
			"table_bucket_arn": testS3TablesTableBucketARN,
		})
	assertS3TablesPreservesID(t, resource)

	if _, ok := newS3TablesNamespaceResource("", &s3tables.GetNamespaceOutput{Namespace: []string{"core_namespace"}}); ok {
		t.Fatal("namespace with empty table bucket ARN should be skipped")
	}
	if _, ok := newS3TablesNamespaceResource(testS3TablesTableBucketARN, nil); ok {
		t.Fatal("nil namespace output should be skipped")
	}
	if _, ok := newS3TablesNamespaceResource(testS3TablesTableBucketARN, &s3tables.GetNamespaceOutput{}); ok {
		t.Fatal("namespace with empty name should be skipped")
	}
	if _, ok := newS3TablesNamespaceResource(testS3TablesTableBucketARN, &s3tables.GetNamespaceOutput{Namespace: []string{"core", "nested"}}); ok {
		t.Fatal("multi-part namespace should be skipped")
	}
}

func TestNewS3TablesTableResource(t *testing.T) {
	resource, ok := newS3TablesTableResource(testS3TablesTableBucketARN, &s3tables.GetTableOutput{
		Format:    s3tablestypes.OpenTableFormatIceberg,
		Name:      aws.String("core_table"),
		Namespace: []string{"core_namespace"},
		TableARN:  aws.String(testS3TablesTableARN),
		Type:      s3tablestypes.TableTypeCustomer,
	})
	assertS3TablesResourceAttributes(t, resource, ok, s3TablesTableResourceType, testS3TablesTableBucketARN+";core_namespace;core_table",
		[]string{"table", testS3TablesTableBucketARN, "core_namespace", "core_table", testS3TablesTableARN},
		map[string]string{
			"arn":              testS3TablesTableARN,
			"format":           "ICEBERG",
			"name":             "core_table",
			"namespace":        "core_namespace",
			"table_bucket_arn": testS3TablesTableBucketARN,
		})
	assertS3TablesPreservesID(t, resource)

	if _, ok := newS3TablesTableResource("", &s3tables.GetTableOutput{
		Format:    s3tablestypes.OpenTableFormatIceberg,
		Name:      aws.String("core_table"),
		Namespace: []string{"core_namespace"},
		TableARN:  aws.String(testS3TablesTableARN),
	}); ok {
		t.Fatal("table with empty table bucket ARN should be skipped")
	}
	if _, ok := newS3TablesTableResource(testS3TablesTableBucketARN, nil); ok {
		t.Fatal("nil table should be skipped")
	}
	if _, ok := newS3TablesTableResource(testS3TablesTableBucketARN, &s3tables.GetTableOutput{
		Name:      aws.String("core_table"),
		Namespace: []string{"core_namespace"},
		TableARN:  aws.String(testS3TablesTableARN),
	}); ok {
		t.Fatal("table with empty format should be skipped")
	}
	if _, ok := newS3TablesTableResource(testS3TablesTableBucketARN, &s3tables.GetTableOutput{
		Format:    s3tablestypes.OpenTableFormatIceberg,
		Namespace: []string{"core_namespace"},
		TableARN:  aws.String(testS3TablesTableARN),
	}); ok {
		t.Fatal("table with empty name should be skipped")
	}
	if _, ok := newS3TablesTableResource(testS3TablesTableBucketARN, &s3tables.GetTableOutput{
		Format:    s3tablestypes.OpenTableFormatIceberg,
		Name:      aws.String("core_table"),
		Namespace: []string{"core", "nested"},
		TableARN:  aws.String(testS3TablesTableARN),
	}); ok {
		t.Fatal("table with multi-part namespace should be skipped")
	}
	if _, ok := newS3TablesTableResource(testS3TablesTableBucketARN, &s3tables.GetTableOutput{
		Format:    s3tablestypes.OpenTableFormatIceberg,
		Name:      aws.String("core_table"),
		Namespace: []string{"core_namespace"},
	}); ok {
		t.Fatal("table with empty ARN should be skipped")
	}
	if _, ok := newS3TablesTableResource(testS3TablesTableBucketARN, &s3tables.GetTableOutput{
		Format:    s3tablestypes.OpenTableFormatIceberg,
		Name:      aws.String("core_table"),
		Namespace: []string{"core_namespace"},
		TableARN:  aws.String(testS3TablesTableARN),
		Type:      s3tablestypes.TableTypeAws,
	}); ok {
		t.Fatal("AWS-managed table should be skipped")
	}
	if _, ok := newS3TablesTableResource(testS3TablesTableBucketARN, &s3tables.GetTableOutput{
		Format:           s3tablestypes.OpenTableFormatIceberg,
		ManagedByService: aws.String("s3.amazonaws.com"),
		Name:             aws.String("core_table"),
		Namespace:        []string{"core_namespace"},
		TableARN:         aws.String(testS3TablesTableARN),
		Type:             s3tablestypes.TableTypeCustomer,
	}); ok {
		t.Fatal("service-managed table should be skipped")
	}
}

func TestNewS3TablesTableBucketPolicyResource(t *testing.T) {
	policy := `{"Version":"2012-10-17"}`
	resource, ok := newS3TablesTableBucketPolicyResource(testS3TablesTableBucketARN, policy)
	assertS3TablesResourceAttributes(t, resource, ok, s3TablesTableBucketPolicyResourceType, testS3TablesTableBucketARN,
		[]string{"table_bucket_policy", testS3TablesTableBucketARN},
		map[string]string{
			"resource_policy":  policy,
			"table_bucket_arn": testS3TablesTableBucketARN,
		})
	assertS3TablesPreservesID(t, resource)

	if _, ok := newS3TablesTableBucketPolicyResource("", policy); ok {
		t.Fatal("policy with empty table bucket ARN should be skipped")
	}
	if _, ok := newS3TablesTableBucketPolicyResource(testS3TablesTableBucketARN, ""); ok {
		t.Fatal("policy with empty policy body should be skipped")
	}
}

func TestS3TablesNamespaceName(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
		ok    bool
	}{
		{name: "single", parts: []string{"core_namespace"}, want: "core_namespace", ok: true},
		{name: "empty slice", ok: false},
		{name: "empty value", parts: []string{""}, ok: false},
		{name: "multi part", parts: []string{"core", "nested"}, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := s3TablesNamespaceName(tt.parts)
			if ok != tt.ok {
				t.Fatalf("ok = %t, want %t", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("namespace = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestS3TablesImportablePredicates(t *testing.T) {
	if !s3TablesTableBucketImportable(&s3tables.GetTableBucketOutput{Type: s3tablestypes.TableBucketTypeCustomer}) {
		t.Fatal("customer table bucket should be importable")
	}
	if s3TablesTableBucketImportable(&s3tables.GetTableBucketOutput{Type: s3tablestypes.TableBucketTypeAws}) {
		t.Fatal("AWS-managed table bucket should not be importable")
	}
	if !s3TablesTableImportable(&s3tables.GetTableOutput{Type: s3tablestypes.TableTypeCustomer}) {
		t.Fatal("customer table should be importable")
	}
	if s3TablesTableImportable(&s3tables.GetTableOutput{Type: s3tablestypes.TableTypeAws}) {
		t.Fatal("AWS-managed table should not be importable")
	}
	if s3TablesTableImportable(&s3tables.GetTableOutput{
		ManagedByService: aws.String("s3.amazonaws.com"),
		Type:             s3tablestypes.TableTypeCustomer,
	}) {
		t.Fatal("service-managed table should not be importable")
	}
}

func TestS3TablesResourceNameAvoidsSanitizedCollisions(t *testing.T) {
	left := terraformutils.TfSanitize(s3TablesResourceName("table", "a_b", "c"))
	right := terraformutils.TfSanitize(s3TablesResourceName("table", "a", "b_c"))
	if left == right {
		t.Fatalf("resource names collide: %q", left)
	}
}

func TestS3TablesResourceNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "typed not found", err: &s3tablestypes.NotFoundException{}, want: true},
		{name: "generic not found", err: &smithy.GenericAPIError{Code: "NotFoundException"}, want: true},
		{name: "resource not found", err: &smithy.GenericAPIError{Code: "ResourceNotFoundException"}, want: true},
		{name: "wrapped not found", err: errors.Join(errors.New("lookup failed"), &s3tablestypes.NotFoundException{}), want: true},
		{name: "access denied", err: &smithy.GenericAPIError{Code: "AccessDenied"}, want: false},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s3TablesResourceNotFound(tt.err); got != tt.want {
				t.Fatalf("not found = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestS3TablesPostConvertHookWrapsPolicy(t *testing.T) {
	policyResource, ok := newS3TablesTableBucketPolicyResource(testS3TablesTableBucketARN, `{"Resource":"${aws:username}"}`)
	if !ok {
		t.Fatal("table bucket policy should be importable")
	}
	policyResource.Item = map[string]interface{}{
		"resource_policy": `{"Resource":"${aws:username}"}`,
	}

	generator := S3TablesGenerator{
		AWSService: AWSService{
			Service: terraformutils.Service{
				Resources: []terraformutils.Resource{policyResource},
			},
		},
	}
	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}
	want := "<<POLICY\n{\"Resource\":\"$" + "$" + "{aws:username}\"}\nPOLICY"
	if got := generator.Resources[0].Item["resource_policy"]; got != want {
		t.Fatalf("resource_policy = %q, want %q", got, want)
	}
}

func assertS3TablesResourceAttributes(t *testing.T, resource terraformutils.Resource, ok bool, resourceType, resourceID string, nameParts []string, attributes map[string]string) {
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
	wantName := terraformutils.TfSanitize(s3TablesResourceName(nameParts...))
	if resource.ResourceName != wantName {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, wantName)
	}
}

func assertS3TablesPreservesID(t *testing.T, resource terraformutils.Resource) {
	t.Helper()
	preserveID, ok := resource.InstanceState.Meta[tfcompat.MetaKeyPreserveIDAfterRefresh].(bool)
	if !ok || !preserveID {
		t.Fatalf("preserve ID metadata = %#v, want true", resource.InstanceState.Meta[tfcompat.MetaKeyPreserveIDAfterRefresh])
	}
}
