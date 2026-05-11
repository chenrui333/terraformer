// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	testRoute53ZoneID      = "Z1234567890"
	testRoute53QueryLogID  = "qlc-1234567890"
	testRoute53KMSKeyARN   = "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"
	testRoute53LogGroupARN = "arn:aws:logs:us-east-1:123456789012:log-group:/aws/route53/example"
)

func TestWildcardUnescape(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"escaped wildcard", "\\052.example.com", "*.example.com"},
		{"no wildcard", "www.example.com", "www.example.com"},
		{"empty string", "", ""},
		{"only wildcard", "\\052", "*"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := wildcardUnescape(tc.input); got != tc.want {
				t.Errorf("wildcardUnescape(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCleanZoneID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"with prefix", "/hostedzone/Z1234ABC", "Z1234ABC"},
		{"without prefix", "Z1234ABC", "Z1234ABC"},
		{"empty string", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := cleanZoneID(tc.input); got != tc.want {
				t.Errorf("cleanZoneID(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCleanPrefix(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		prefix string
		want   string
	}{
		{"matching prefix", "/hostedzone/Z123", "/hostedzone/", "Z123"},
		{"no match", "Z123", "/hostedzone/", "Z123"},
		{"empty id", "", "/hostedzone/", ""},
		{"empty prefix", "/hostedzone/Z123", "", "/hostedzone/Z123"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := cleanPrefix(tc.id, tc.prefix); got != tc.want {
				t.Errorf("cleanPrefix(%q, %q) = %q, want %q", tc.id, tc.prefix, got, tc.want)
			}
		})
	}
}

func TestRoute53ResourceIDs(t *testing.T) {
	if got, want := route53KeySigningKeyImportID(testRoute53ZoneID, "core-key"), testRoute53ZoneID+",core-key"; got != want {
		t.Fatalf("key signing key import ID = %q, want %q", got, want)
	}
	if got, want := cleanDelegationSetID("/delegationset/N1234567890"), "N1234567890"; got != want {
		t.Fatalf("delegation set ID = %q, want %q", got, want)
	}
}

func TestNewRoute53QueryLogResource(t *testing.T) {
	resource, ok := newRoute53QueryLogResource(route53types.QueryLoggingConfig{
		CloudWatchLogsLogGroupArn: aws.String(testRoute53LogGroupARN),
		HostedZoneId:              aws.String("/hostedzone/" + testRoute53ZoneID),
		Id:                        aws.String(testRoute53QueryLogID),
	})
	assertRoute53ResourceAttributes(t, resource, ok, route53QueryLogResourceType, testRoute53QueryLogID,
		[]string{"query_log", testRoute53ZoneID, testRoute53QueryLogID},
		map[string]string{
			"cloudwatch_log_group_arn": testRoute53LogGroupARN,
			"zone_id":                  testRoute53ZoneID,
		})

	if _, ok := newRoute53QueryLogResource(route53types.QueryLoggingConfig{HostedZoneId: aws.String(testRoute53ZoneID), Id: aws.String(testRoute53QueryLogID)}); ok {
		t.Fatal("query log with empty CloudWatch log group ARN should be skipped")
	}
}

func TestNewRoute53DelegationSetResource(t *testing.T) {
	resource, ok := newRoute53DelegationSetResource(route53types.DelegationSet{Id: aws.String("/delegationset/N1234567890")})
	assertRoute53ResourceAttributes(t, resource, ok, route53DelegationSetResourceType, "N1234567890",
		[]string{"delegation_set", "N1234567890"},
		map[string]string{})

	if _, ok := newRoute53DelegationSetResource(route53types.DelegationSet{}); ok {
		t.Fatal("delegation set with empty ID should be skipped")
	}
}

func TestNewRoute53HostedZoneDNSSECResource(t *testing.T) {
	resource, ok := newRoute53HostedZoneDNSSECResource(testRoute53ZoneID, &route53.GetDNSSECOutput{
		Status: &route53types.DNSSECStatus{ServeSignature: aws.String("SIGNING")},
	})
	assertRoute53ResourceAttributes(t, resource, ok, route53HostedZoneDNSSECResourceType, testRoute53ZoneID,
		[]string{"hosted_zone_dnssec", testRoute53ZoneID},
		map[string]string{
			"hosted_zone_id": testRoute53ZoneID,
			"signing_status": "SIGNING",
		})

	if _, ok := newRoute53HostedZoneDNSSECResource(testRoute53ZoneID, &route53.GetDNSSECOutput{Status: &route53types.DNSSECStatus{ServeSignature: aws.String("NOT_SIGNING")}}); ok {
		t.Fatal("DNSSEC resource should skip non-signing hosted zones")
	}
	if _, ok := newRoute53HostedZoneDNSSECResource("", &route53.GetDNSSECOutput{Status: &route53types.DNSSECStatus{ServeSignature: aws.String("SIGNING")}}); ok {
		t.Fatal("DNSSEC resource with empty zone ID should be skipped")
	}
}

func TestNewRoute53KeySigningKeyResource(t *testing.T) {
	resource, ok := newRoute53KeySigningKeyResource(testRoute53ZoneID, route53types.KeySigningKey{
		KmsArn: aws.String(testRoute53KMSKeyARN),
		Name:   aws.String("core-key"),
		Status: aws.String("ACTIVE"),
	})
	assertRoute53ResourceAttributes(t, resource, ok, route53KeySigningKeyResourceType, testRoute53ZoneID+",core-key",
		[]string{"key_signing_key", testRoute53ZoneID, "core-key"},
		map[string]string{
			"hosted_zone_id":             testRoute53ZoneID,
			"key_management_service_arn": testRoute53KMSKeyARN,
			"name":                       "core-key",
			"status":                     "ACTIVE",
		})

	if _, ok := newRoute53KeySigningKeyResource(testRoute53ZoneID, route53types.KeySigningKey{KmsArn: aws.String(testRoute53KMSKeyARN), Name: aws.String("core-key"), Status: aws.String("ACTION_NEEDED")}); ok {
		t.Fatal("key signing key with unsupported status should be skipped")
	}
	if _, ok := newRoute53KeySigningKeyResource(testRoute53ZoneID, route53types.KeySigningKey{Name: aws.String("core-key"), Status: aws.String("ACTIVE")}); ok {
		t.Fatal("key signing key with empty KMS ARN should be skipped")
	}
}

func TestRoute53PostConvertHookReplacesHostedZoneReferences(t *testing.T) {
	zone := terraformutils.NewResource(
		testRoute53ZoneID,
		"example_com",
		route53ZoneResourceType,
		"aws",
		map[string]string{"name": "example.com."},
		route53AllowEmptyValues,
		route53AdditionalFields,
	)
	queryLog, _ := newRoute53QueryLogResource(route53types.QueryLoggingConfig{
		CloudWatchLogsLogGroupArn: aws.String(testRoute53LogGroupARN),
		HostedZoneId:              aws.String(testRoute53ZoneID),
		Id:                        aws.String(testRoute53QueryLogID),
	})
	queryLog.Item = map[string]interface{}{"zone_id": testRoute53ZoneID}
	dnssec, _ := newRoute53HostedZoneDNSSECResource(testRoute53ZoneID, &route53.GetDNSSECOutput{
		Status: &route53types.DNSSECStatus{ServeSignature: aws.String("SIGNING")},
	})
	dnssec.Item = map[string]interface{}{"hosted_zone_id": testRoute53ZoneID}

	generator := Route53Generator{AWSService: AWSService{Service: terraformutils.Service{Resources: []terraformutils.Resource{zone, queryLog, dnssec}}}}
	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}
	want := "${aws_route53_zone." + zone.ResourceName + ".zone_id}"
	if got := generator.Resources[1].Item["zone_id"]; got != want {
		t.Fatalf("query log zone_id = %q, want %q", got, want)
	}
	if got := generator.Resources[2].Item["hosted_zone_id"]; got != want {
		t.Fatalf("DNSSEC hosted_zone_id = %q, want %q", got, want)
	}
}

func TestRoute53ResourceNameAvoidsSanitizedCollisions(t *testing.T) {
	left := terraformutils.TfSanitize(route53ResourceName("query_log", "a_b", "c"))
	right := terraformutils.TfSanitize(route53ResourceName("query_log", "a", "b_c"))
	if left == right {
		t.Fatalf("resource names collide: %q", left)
	}
}

func TestRoute53ResourceNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "typed DNSSEC not found", err: &route53types.DNSSECNotFound{}, want: true},
		{name: "typed hosted zone not found", err: &route53types.NoSuchHostedZone{}, want: true},
		{name: "generic query log not found", err: &smithy.GenericAPIError{Code: "NoSuchQueryLoggingConfig"}, want: true},
		{name: "wrapped delegation set not found", err: errors.Join(errors.New("lookup failed"), &route53types.NoSuchDelegationSet{}), want: true},
		{name: "access denied", err: &smithy.GenericAPIError{Code: "AccessDenied"}, want: false},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := route53ResourceNotFound(tt.err); got != tt.want {
				t.Fatalf("not found = %t, want %t", got, tt.want)
			}
		})
	}
}

func assertRoute53ResourceAttributes(t *testing.T, resource terraformutils.Resource, ok bool, resourceType, resourceID string, nameParts []string, attributes map[string]string) {
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
	for key, want := range attributes {
		if got := resource.InstanceState.Attributes[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
	wantName := terraformutils.TfSanitize(route53ResourceName(nameParts...))
	if resource.ResourceName != wantName {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, wantName)
	}
}
