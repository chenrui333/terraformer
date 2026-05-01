// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
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
