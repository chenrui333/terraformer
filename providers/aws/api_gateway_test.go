// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestAPIGatewayImportIDs(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "account", got: apiGatewayAccountImportID("123456789012"), want: "123456789012"},
		{name: "base path mapping", got: apiGatewayBasePathMappingImportID("api.example.com", "v1", ""), want: "api.example.com/v1"},
		{name: "root base path mapping", got: apiGatewayBasePathMappingImportID("api.example.com", "", ""), want: "api.example.com/"},
		{name: "private domain base path mapping", got: apiGatewayBasePathMappingImportID("api.example.com", "v1", "domain-id"), want: "api.example.com/v1/domain-id"},
		{name: "client certificate", got: apiGatewayClientCertificateImportID("cert-123"), want: "cert-123"},
		{name: "documentation version", got: apiGatewayDocumentationVersionImportID("api-123", "v1"), want: "api-123/v1"},
		{name: "request validator", got: apiGatewayRequestValidatorImportID("api-123", "validator-456"), want: "api-123/validator-456"},
		{name: "usage plan key", got: apiGatewayUsagePlanKeyImportID("plan-123", "key-456"), want: "plan-123/key-456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("import ID = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestNewAPIGatewayAccountResource(t *testing.T) {
	resource, ok := newAPIGatewayAccountResource("123456789012", &apigateway.GetAccountOutput{
		CloudwatchRoleArn: aws.String("arn:aws:iam::123456789012:role/apigw-cloudwatch"),
	})
	if !ok {
		t.Fatal("newAPIGatewayAccountResource() ok = false, want true")
	}
	assertAPIGatewayResourceAttributes(t, resource, apiGatewayAccountResourceType, "123456789012",
		[]string{"account", "123456789012"},
		map[string]string{
			"cloudwatch_role_arn": "arn:aws:iam::123456789012:role/apigw-cloudwatch",
		})

	if _, ok := newAPIGatewayAccountResource("", &apigateway.GetAccountOutput{CloudwatchRoleArn: aws.String("arn")}); ok {
		t.Fatal("newAPIGatewayAccountResource() ok = true for empty account ID, want false")
	}
	if _, ok := newAPIGatewayAccountResource("123456789012", nil); ok {
		t.Fatal("newAPIGatewayAccountResource() ok = true for nil output, want false")
	}
	if _, ok := newAPIGatewayAccountResource("123456789012", &apigateway.GetAccountOutput{}); ok {
		t.Fatal("newAPIGatewayAccountResource() ok = true for unconfigured account, want false")
	}
}

func TestNewAPIGatewayBasePathMappingResource(t *testing.T) {
	resource, ok := newAPIGatewayBasePathMappingResource("api.example.com", "domain-id", types.BasePathMapping{
		BasePath:  aws.String("v1"),
		RestApiId: aws.String("api-123"),
		Stage:     aws.String("prod"),
	})
	if !ok {
		t.Fatal("newAPIGatewayBasePathMappingResource() ok = false, want true")
	}
	assertAPIGatewayResourceAttributes(t, resource, apiGatewayBasePathMappingResourceType, "api.example.com/v1/domain-id",
		[]string{"base_path_mapping", "api.example.com", "v1", "domain-id"},
		map[string]string{
			"api_id":         "api-123",
			"base_path":      "v1",
			"domain_name":    "api.example.com",
			"domain_name_id": "domain-id",
			"stage_name":     "prod",
		})

	resource, ok = newAPIGatewayBasePathMappingResource("api.example.com", "", types.BasePathMapping{
		BasePath:  aws.String(apiGatewayEmptyBasePathMappingValue),
		RestApiId: aws.String("api-123"),
	})
	if !ok {
		t.Fatal("newAPIGatewayBasePathMappingResource() ok = false for root mapping, want true")
	}
	assertAPIGatewayResourceAttributes(t, resource, apiGatewayBasePathMappingResourceType, "api.example.com/",
		[]string{"base_path_mapping", "api.example.com"},
		map[string]string{
			"api_id":      "api-123",
			"base_path":   "",
			"domain_name": "api.example.com",
		})
	if _, ok := resource.InstanceState.Attributes["stage_name"]; ok {
		t.Fatal("newAPIGatewayBasePathMappingResource() seeded empty stage_name, want omitted")
	}

	if _, ok := newAPIGatewayBasePathMappingResource("", "", types.BasePathMapping{RestApiId: aws.String("api-123")}); ok {
		t.Fatal("newAPIGatewayBasePathMappingResource() ok = true for empty domain name, want false")
	}
	if _, ok := newAPIGatewayBasePathMappingResource("api.example.com", "", types.BasePathMapping{}); ok {
		t.Fatal("newAPIGatewayBasePathMappingResource() ok = true without rest API ID, want false")
	}
	if _, ok := newAPIGatewayBasePathMappingResource("api.example.com", "", types.BasePathMapping{
		BasePath:  aws.String("v1/orders"),
		RestApiId: aws.String("api-123"),
	}); ok {
		t.Fatal("newAPIGatewayBasePathMappingResource() ok = true for slash-delimited base path, want false")
	}
}

func TestAPIGatewayBasePathMappingsRespectRestAPITagFilters(t *testing.T) {
	noFilter := &APIGatewayGenerator{}
	if noFilter.shouldFilterBasePathMapping(types.BasePathMapping{RestApiId: aws.String("api-excluded")}) {
		t.Fatal("shouldFilterBasePathMapping() = true without REST API tag filter, want false")
	}

	filtered := &APIGatewayGenerator{}
	filtered.Filter = []terraformutils.ResourceFilter{{
		ServiceName:      "api_gateway_rest_api",
		FieldPath:        "tags.Environment",
		AcceptableValues: []string{"prod"},
	}}
	filtered.rememberAcceptedRestAPIID("api-allowed")

	if filtered.shouldFilterBasePathMapping(types.BasePathMapping{RestApiId: aws.String("api-allowed")}) {
		t.Fatal("shouldFilterBasePathMapping() = true for accepted REST API, want false")
	}
	if !filtered.shouldFilterBasePathMapping(types.BasePathMapping{RestApiId: aws.String("api-excluded")}) {
		t.Fatal("shouldFilterBasePathMapping() = false for filtered REST API, want true")
	}
	if !filtered.shouldFilterBasePathMapping(types.BasePathMapping{}) {
		t.Fatal("shouldFilterBasePathMapping() = false for empty REST API ID under tag filter, want true")
	}
}

func TestNewAPIGatewayClientCertificateResource(t *testing.T) {
	resource, ok := newAPIGatewayClientCertificateResource(types.ClientCertificate{
		ClientCertificateId: aws.String("cert-123"),
		Description:         aws.String("backend mTLS"),
	})
	if !ok {
		t.Fatal("newAPIGatewayClientCertificateResource() ok = false, want true")
	}
	assertAPIGatewayResourceAttributes(t, resource, apiGatewayClientCertificateResourceType, "cert-123",
		[]string{"client_certificate", "cert-123"},
		map[string]string{
			"description": "backend mTLS",
		})

	if _, ok := newAPIGatewayClientCertificateResource(types.ClientCertificate{}); ok {
		t.Fatal("newAPIGatewayClientCertificateResource() ok = true for empty certificate ID, want false")
	}
}

func TestNewAPIGatewayDocumentationVersionResource(t *testing.T) {
	resource, ok := newAPIGatewayDocumentationVersionResource("api-123", types.DocumentationVersion{
		Description: aws.String("public docs"),
		Version:     aws.String("v1"),
	})
	if !ok {
		t.Fatal("newAPIGatewayDocumentationVersionResource() ok = false, want true")
	}
	assertAPIGatewayResourceAttributes(t, resource, apiGatewayDocumentationVersionResourceType, "api-123/v1",
		[]string{"documentation_version", "api-123", "v1"},
		map[string]string{
			"description": "public docs",
			"rest_api_id": "api-123",
			"version":     "v1",
		})

	if _, ok := newAPIGatewayDocumentationVersionResource("", types.DocumentationVersion{Version: aws.String("v1")}); ok {
		t.Fatal("newAPIGatewayDocumentationVersionResource() ok = true for empty rest API ID, want false")
	}
	if _, ok := newAPIGatewayDocumentationVersionResource("api-123", types.DocumentationVersion{}); ok {
		t.Fatal("newAPIGatewayDocumentationVersionResource() ok = true for empty version, want false")
	}
}

func TestNewAPIGatewayRequestValidatorResource(t *testing.T) {
	resource, ok := newAPIGatewayRequestValidatorResource("api-123", types.RequestValidator{
		Id:                        aws.String("validator-456"),
		Name:                      aws.String("body-and-params"),
		ValidateRequestBody:       true,
		ValidateRequestParameters: true,
	})
	if !ok {
		t.Fatal("newAPIGatewayRequestValidatorResource() ok = false, want true")
	}
	assertAPIGatewayResourceAttributes(t, resource, apiGatewayRequestValidatorResourceType, "validator-456",
		[]string{"request_validator", "api-123", "validator-456"},
		map[string]string{
			"name":                        "body-and-params",
			"rest_api_id":                 "api-123",
			"validate_request_body":       "true",
			"validate_request_parameters": "true",
		})

	if _, ok := newAPIGatewayRequestValidatorResource("", types.RequestValidator{Id: aws.String("validator-456"), Name: aws.String("validator")}); ok {
		t.Fatal("newAPIGatewayRequestValidatorResource() ok = true for empty rest API ID, want false")
	}
	if _, ok := newAPIGatewayRequestValidatorResource("api-123", types.RequestValidator{Name: aws.String("validator")}); ok {
		t.Fatal("newAPIGatewayRequestValidatorResource() ok = true for empty validator ID, want false")
	}
	if _, ok := newAPIGatewayRequestValidatorResource("api-123", types.RequestValidator{Id: aws.String("validator-456")}); ok {
		t.Fatal("newAPIGatewayRequestValidatorResource() ok = true for empty validator name, want false")
	}
}

func TestNewAPIGatewayResourceResource(t *testing.T) {
	resource, ok := newAPIGatewayResourceResource("api-123", types.Resource{
		Id:       aws.String("resource-456"),
		ParentId: aws.String("parent-123"),
		Path:     aws.String("/pets/{petId}"),
		PathPart: aws.String("{petId}"),
	})
	if !ok {
		t.Fatal("newAPIGatewayResourceResource() ok = false, want true")
	}
	assertAPIGatewayResourceAttributes(t, resource, apiGatewayResourceResourceType, "resource-456",
		[]string{"resource", "api-123", "resource-456"},
		map[string]string{
			"parent_id":   "parent-123",
			"path":        "/pets/{petId}",
			"path_part":   "{petId}",
			"rest_api_id": "api-123",
		})
	if _, ok := resource.InstanceState.Attributes["partent_id"]; ok {
		t.Fatal("newAPIGatewayResourceResource() seeded misspelled partent_id, want parent_id only")
	}

	if _, ok := newAPIGatewayResourceResource("", types.Resource{Id: aws.String("resource-456")}); ok {
		t.Fatal("newAPIGatewayResourceResource() ok = true for empty rest API ID, want false")
	}
	if _, ok := newAPIGatewayResourceResource("api-123", types.Resource{
		ParentId: aws.String("parent-123"),
		PathPart: aws.String("pets"),
	}); ok {
		t.Fatal("newAPIGatewayResourceResource() ok = true for empty resource ID, want false")
	}
	if _, ok := newAPIGatewayResourceResource("api-123", types.Resource{
		Id:       aws.String("resource-456"),
		PathPart: aws.String("pets"),
	}); ok {
		t.Fatal("newAPIGatewayResourceResource() ok = true for empty parent ID, want false")
	}
	if _, ok := newAPIGatewayResourceResource("api-123", types.Resource{
		Id:       aws.String("resource-456"),
		ParentId: aws.String("parent-123"),
	}); ok {
		t.Fatal("newAPIGatewayResourceResource() ok = true for empty path part, want false")
	}
}

func TestNewAPIGatewayUsagePlanKeyResource(t *testing.T) {
	resource, ok := newAPIGatewayUsagePlanKeyResource("plan-123", types.UsagePlanKey{
		Id:   aws.String("key-456"),
		Type: aws.String("API_KEY"),
	})
	if !ok {
		t.Fatal("newAPIGatewayUsagePlanKeyResource() ok = false, want true")
	}
	assertAPIGatewayResourceAttributes(t, resource, apiGatewayUsagePlanKeyResourceType, "key-456",
		[]string{"usage_plan_key", "plan-123", "key-456"},
		map[string]string{
			"key_id":        "key-456",
			"key_type":      "API_KEY",
			"usage_plan_id": "plan-123",
		})

	if _, ok := newAPIGatewayUsagePlanKeyResource("", types.UsagePlanKey{Id: aws.String("key-456"), Type: aws.String("API_KEY")}); ok {
		t.Fatal("newAPIGatewayUsagePlanKeyResource() ok = true for empty usage plan ID, want false")
	}
	if _, ok := newAPIGatewayUsagePlanKeyResource("plan-123", types.UsagePlanKey{Type: aws.String("API_KEY")}); ok {
		t.Fatal("newAPIGatewayUsagePlanKeyResource() ok = true for empty key ID, want false")
	}
	if _, ok := newAPIGatewayUsagePlanKeyResource("plan-123", types.UsagePlanKey{Id: aws.String("key-456")}); ok {
		t.Fatal("newAPIGatewayUsagePlanKeyResource() ok = true for empty key type, want false")
	}
}

func TestAPIGatewayResourceNameIsCollisionResistant(t *testing.T) {
	first := terraformutils.TfSanitize(apiGatewayResourceName("base_path_mapping", "a/b", "c"))
	second := terraformutils.TfSanitize(apiGatewayResourceName("base_path_mapping", "a", "b/c"))
	if first == second {
		t.Fatalf("apiGatewayResourceName() generated duplicate sanitized names %q", first)
	}
}

func TestAPIGatewayResourceMissing(t *testing.T) {
	if !apiGatewayResourceMissing(&types.NotFoundException{}) {
		t.Fatal("apiGatewayResourceMissing() = false for NotFoundException, want true")
	}
	if apiGatewayResourceMissing(errors.New("boom")) {
		t.Fatal("apiGatewayResourceMissing() = true for generic error, want false")
	}
	if apiGatewayResourceMissing(nil) {
		t.Fatal("apiGatewayResourceMissing() = true for nil error, want false")
	}
}

func assertAPIGatewayResourceAttributes(t *testing.T, resource terraformutils.Resource, resourceType, resourceID string, nameParts []string, attributes map[string]string) {
	t.Helper()

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
	wantName := terraformutils.TfSanitize(apiGatewayResourceName(nameParts...))
	if resource.ResourceName != wantName {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, wantName)
	}
}
