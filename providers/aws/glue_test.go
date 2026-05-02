// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	gluetypes "github.com/aws/aws-sdk-go-v2/service/glue/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestGlueClassifierName(t *testing.T) {
	tests := []struct {
		name       string
		classifier gluetypes.Classifier
		want       string
	}{
		{name: "csv", classifier: gluetypes.Classifier{CsvClassifier: &gluetypes.CsvClassifier{Name: aws.String("csv-classifier")}}, want: "csv-classifier"},
		{name: "grok", classifier: gluetypes.Classifier{GrokClassifier: &gluetypes.GrokClassifier{Name: aws.String("grok-classifier")}}, want: "grok-classifier"},
		{name: "json", classifier: gluetypes.Classifier{JsonClassifier: &gluetypes.JsonClassifier{Name: aws.String("json-classifier")}}, want: "json-classifier"},
		{name: "xml", classifier: gluetypes.Classifier{XMLClassifier: &gluetypes.XMLClassifier{Name: aws.String("xml-classifier")}}, want: "xml-classifier"},
		{name: "empty", classifier: gluetypes.Classifier{}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := glueClassifierName(tt.classifier)
			if got != tt.want {
				t.Fatalf("glueClassifierName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGlueResourceName(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "filters empty parts", parts: []string{"", "database", "", "function"}, want: "database/function"},
		{name: "preserves segment boundaries", parts: []string{"orders", "stream", "policy"}, want: "orders/stream/policy"},
		{name: "fallback", parts: nil, want: "glue_resource"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := glueResourceName(tt.parts...)
			if got != tt.want {
				t.Fatalf("glueResourceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGlueUserDefinedFunctionImportID(t *testing.T) {
	got := glueUserDefinedFunctionImportID("123456789012", "analytics", "normalize_json")
	want := "123456789012:analytics:normalize_json"
	if got != want {
		t.Fatalf("glueUserDefinedFunctionImportID() = %q, want %q", got, want)
	}
}

func TestGlueDevEndpointImportable(t *testing.T) {
	tests := []struct {
		name     string
		endpoint gluetypes.DevEndpoint
		want     bool
	}{
		{name: "empty status", endpoint: gluetypes.DevEndpoint{}, want: true},
		{name: "ready", endpoint: gluetypes.DevEndpoint{Status: aws.String("READY")}, want: true},
		{name: "deleting", endpoint: gluetypes.DevEndpoint{Status: aws.String("DELETING")}, want: false},
		{name: "failed", endpoint: gluetypes.DevEndpoint{Status: aws.String("FAILED")}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := glueDevEndpointImportable(tt.endpoint)
			if got != tt.want {
				t.Fatalf("glueDevEndpointImportable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGlueMLTransformImportable(t *testing.T) {
	tests := []struct {
		name      string
		transform gluetypes.MLTransform
		want      bool
	}{
		{name: "ready", transform: gluetypes.MLTransform{Status: gluetypes.TransformStatusTypeReady}, want: true},
		{name: "not ready", transform: gluetypes.MLTransform{Status: gluetypes.TransformStatusTypeNotReady}, want: true},
		{name: "deleting", transform: gluetypes.MLTransform{Status: gluetypes.TransformStatusTypeDeleting}, want: false},
		{name: "empty", transform: gluetypes.MLTransform{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := glueMLTransformImportable(tt.transform)
			if got != tt.want {
				t.Fatalf("glueMLTransformImportable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGlueRegistryImportable(t *testing.T) {
	tests := []struct {
		name     string
		registry gluetypes.RegistryListItem
		want     bool
	}{
		{name: "available", registry: gluetypes.RegistryListItem{Status: gluetypes.RegistryStatusAvailable}, want: true},
		{name: "deleting", registry: gluetypes.RegistryListItem{Status: gluetypes.RegistryStatusDeleting}, want: false},
		{name: "empty", registry: gluetypes.RegistryListItem{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := glueRegistryImportable(tt.registry)
			if got != tt.want {
				t.Fatalf("glueRegistryImportable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGlueResourceMissing(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "entity not found", err: &gluetypes.EntityNotFoundException{}, want: true},
		{name: "resource not found", err: &gluetypes.ResourceNotFoundException{}, want: true},
		{name: "wrapped", err: fmt.Errorf("wrapped: %w", &gluetypes.EntityNotFoundException{}), want: true},
		{name: "api notfound code", err: &smithy.GenericAPIError{Code: "ResourceNotFoundException"}, want: true},
		{name: "other", err: errors.New("boom"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := glueResourceMissing(tt.err)
			if got != tt.want {
				t.Fatalf("glueResourceMissing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGluePostConvertHookWrapsResourcePolicy(t *testing.T) {
	resource := terraformutils.NewSimpleResource("us-east-1", "resource_policy", "aws_glue_resource_policy", "aws", glueAllowEmptyValues)
	resource.Item = map[string]interface{}{"policy": "{\"Resource\":\"$" + "{aws:glue}\"}"}
	g := &GlueGenerator{}
	g.Resources = []terraformutils.Resource{resource}

	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}

	want := "<<POLICY\n{\"Resource\":\"$" + "$" + "{aws:glue}\"}\nPOLICY"
	if got := g.Resources[0].Item["policy"]; got != want {
		t.Fatalf("policy = %q, want %q", got, want)
	}
}
