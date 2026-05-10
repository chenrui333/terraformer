// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestLambdaAliasImportID(t *testing.T) {
	if got, want := lambdaAliasImportID("my-function", "prod"), "my-function/prod"; got != want {
		t.Fatalf("lambdaAliasImportID() = %q, want %q", got, want)
	}
}

func TestLambdaFunctionURLImportID(t *testing.T) {
	tests := []struct {
		name         string
		functionName string
		qualifier    string
		want         string
	}{
		{name: "unqualified", functionName: "my-function", want: "my-function"},
		{name: "qualified", functionName: "my-function", qualifier: "prod", want: "my-function/prod"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lambdaFunctionURLImportID(tt.functionName, tt.qualifier); got != tt.want {
				t.Fatalf("lambdaFunctionURLImportID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLambdaProvisionedConcurrencyConfigImportID(t *testing.T) {
	if got, want := lambdaProvisionedConcurrencyConfigImportID("my-function", "prod"), "my-function,prod"; got != want {
		t.Fatalf("lambdaProvisionedConcurrencyConfigImportID() = %q, want %q", got, want)
	}
}

func TestLambdaFunctionRecursionConfigImportID(t *testing.T) {
	if got, want := lambdaFunctionRecursionConfigImportID("my-function"), "my-function"; got != want {
		t.Fatalf("lambdaFunctionRecursionConfigImportID() = %q, want %q", got, want)
	}
}

func TestLambdaRuntimeManagementConfigImportID(t *testing.T) {
	tests := []struct {
		name         string
		functionName string
		qualifier    string
		want         string
	}{
		{name: "unqualified", functionName: "my-function", want: "my-function,"},
		{name: "qualified", functionName: "my-function", qualifier: "prod", want: "my-function,prod"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lambdaRuntimeManagementConfigImportID(tt.functionName, tt.qualifier); got != tt.want {
				t.Fatalf("lambdaRuntimeManagementConfigImportID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLambdaEventSourceMappingAttributes(t *testing.T) {
	tests := []struct {
		name           string
		functionARN    string
		eventSourceARN string
		want           map[string]string
	}{
		{
			name:           "managed event source",
			functionARN:    "arn:aws:lambda:us-east-1:123456789012:function:consumer",
			eventSourceARN: "arn:aws:sqs:us-east-1:123456789012:queue",
			want: map[string]string{
				"event_source_arn": "arn:aws:sqs:us-east-1:123456789012:queue",
				"function_name":    "arn:aws:lambda:us-east-1:123456789012:function:consumer",
			},
		},
		{
			name:        "self managed event source",
			functionARN: "arn:aws:lambda:us-east-1:123456789012:function:consumer",
			want: map[string]string{
				"function_name": "arn:aws:lambda:us-east-1:123456789012:function:consumer",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lambdaEventSourceMappingAttributes(tt.functionARN, tt.eventSourceARN)
			if len(got) != len(tt.want) {
				t.Fatalf("lambdaEventSourceMappingAttributes() = %#v, want %#v", got, tt.want)
			}
			for key, want := range tt.want {
				if got[key] != want {
					t.Fatalf("lambdaEventSourceMappingAttributes()[%q] = %q, want %q", key, got[key], want)
				}
			}
		})
	}
}

func TestLambdaQualifierFromFunctionARN(t *testing.T) {
	tests := []struct {
		name         string
		functionARN  string
		functionName string
		want         string
	}{
		{
			name:         "unqualified arn",
			functionARN:  "arn:aws:lambda:us-east-1:123456789012:function:my-function",
			functionName: "my-function",
		},
		{
			name:         "alias qualifier",
			functionARN:  "arn:aws:lambda:us-east-1:123456789012:function:my-function:prod",
			functionName: "my-function",
			want:         "prod",
		},
		{
			name:         "version qualifier",
			functionARN:  "arn:aws:lambda:us-east-1:123456789012:function:my-function:42",
			functionName: "my-function",
			want:         "42",
		},
		{
			name:         "malformed arn",
			functionARN:  "my-function:prod",
			functionName: "my-function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lambdaQualifierFromFunctionARN(tt.functionARN, tt.functionName); got != tt.want {
				t.Fatalf("lambdaQualifierFromFunctionARN() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLambdaResourceName(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "joins parts", parts: []string{"function", "alias"}, want: "function_alias"},
		{name: "omits empty parts", parts: []string{"", "function", "", "url"}, want: "function_url"},
		{name: "empty", parts: []string{"", ""}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lambdaResourceName(tt.parts...); got != tt.want {
				t.Fatalf("lambdaResourceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewLambdaFunctionRecursionConfigResource(t *testing.T) {
	resource, ok := newLambdaFunctionRecursionConfigResource("my-function", &lambda.GetFunctionRecursionConfigOutput{
		RecursiveLoop: lambdatypes.RecursiveLoopTerminate,
	})
	if !ok {
		t.Fatal("newLambdaFunctionRecursionConfigResource() ok = false, want true")
	}
	if resource.InstanceInfo.Type != lambdaFunctionRecursionConfigResourceType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, lambdaFunctionRecursionConfigResourceType)
	}
	if resource.InstanceState.ID != "my-function" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "my-function")
	}
	wantName := terraformutils.TfSanitize(lambdaResourceNameWithLengths("function_recursion_config", "my-function"))
	if resource.ResourceName != wantName {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, wantName)
	}
	wantAttributes := map[string]string{
		"function_name":  "my-function",
		"recursive_loop": "Terminate",
	}
	for key, want := range wantAttributes {
		if got := resource.InstanceState.Attributes[key]; got != want {
			t.Fatalf("attribute %q = %q, want %q", key, got, want)
		}
	}
	if _, ok := newLambdaFunctionRecursionConfigResource("", &lambda.GetFunctionRecursionConfigOutput{
		RecursiveLoop: lambdatypes.RecursiveLoopTerminate,
	}); ok {
		t.Fatal("newLambdaFunctionRecursionConfigResource() ok = true for empty function name, want false")
	}
	if _, ok := newLambdaFunctionRecursionConfigResource("my-function", nil); ok {
		t.Fatal("newLambdaFunctionRecursionConfigResource() ok = true for nil config, want false")
	}
	if _, ok := newLambdaFunctionRecursionConfigResource("my-function", &lambda.GetFunctionRecursionConfigOutput{}); ok {
		t.Fatal("newLambdaFunctionRecursionConfigResource() ok = true for empty recursive_loop, want false")
	}
}

func TestNewLambdaRuntimeManagementConfigResource(t *testing.T) {
	resource, ok := newLambdaRuntimeManagementConfigResource("my-function", "", &lambda.GetRuntimeManagementConfigOutput{
		RuntimeVersionArn: aws.String("arn:aws:lambda:us-east-1::runtime:abcd"),
		UpdateRuntimeOn:   lambdatypes.UpdateRuntimeOnManual,
	})
	if !ok {
		t.Fatal("newLambdaRuntimeManagementConfigResource() ok = false, want true")
	}
	if resource.InstanceInfo.Type != lambdaRuntimeManagementConfigResourceType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, lambdaRuntimeManagementConfigResourceType)
	}
	if resource.InstanceState.ID != "my-function," {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "my-function,")
	}
	wantName := terraformutils.TfSanitize(lambdaResourceNameWithLengths("runtime_management_config", "my-function", ""))
	if resource.ResourceName != wantName {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, wantName)
	}
	wantAttributes := map[string]string{
		"function_name":       "my-function",
		"qualifier":           "",
		"runtime_version_arn": "arn:aws:lambda:us-east-1::runtime:abcd",
		"update_runtime_on":   "Manual",
	}
	for key, want := range wantAttributes {
		if got := resource.InstanceState.Attributes[key]; got != want {
			t.Fatalf("attribute %q = %q, want %q", key, got, want)
		}
	}
	if _, ok := newLambdaRuntimeManagementConfigResource("", "", &lambda.GetRuntimeManagementConfigOutput{
		UpdateRuntimeOn: lambdatypes.UpdateRuntimeOnAuto,
	}); ok {
		t.Fatal("newLambdaRuntimeManagementConfigResource() ok = true for empty function name, want false")
	}
	if _, ok := newLambdaRuntimeManagementConfigResource("my-function", "", nil); ok {
		t.Fatal("newLambdaRuntimeManagementConfigResource() ok = true for nil config, want false")
	}
}

func TestLambdaResourceNameWithLengthsAvoidsSanitizedCollisions(t *testing.T) {
	tests := []struct {
		name   string
		first  []string
		second []string
	}{
		{name: "separator boundary", first: []string{"runtime_management_config", "a_b", "c"}, second: []string{"runtime_management_config", "a", "b_c"}},
		{name: "slash encoding", first: []string{"runtime_management_config", "a/b", "c"}, second: []string{"runtime_management_config", "a-002F-b", "c"}},
		{name: "qualifier separator encoding", first: []string{"runtime_management_config", "function:prod"}, second: []string{"runtime_management_config", "function-003A-prod"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := terraformutils.TfSanitize(lambdaResourceNameWithLengths(tt.first...))
			second := terraformutils.TfSanitize(lambdaResourceNameWithLengths(tt.second...))
			if first == second {
				t.Fatalf("lambdaResourceNameWithLengths() generated duplicate sanitized names %q", first)
			}
		})
	}
}

func TestLambdaResourceNotFound(t *testing.T) {
	if !lambdaResourceNotFound(&lambdatypes.ResourceNotFoundException{}) {
		t.Fatal("lambdaResourceNotFound() = false for ResourceNotFoundException, want true")
	}
	if lambdaResourceNotFound(errors.New("boom")) {
		t.Fatal("lambdaResourceNotFound() = true for generic error, want false")
	}
	if lambdaResourceNotFound(nil) {
		t.Fatal("lambdaResourceNotFound() = true for nil, want false")
	}
}
