// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
)

func TestAPIGatewayV2ImportID(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "stage", parts: []string{"api-123", "prod"}, want: "api-123/prod"},
		{name: "model", parts: []string{"api-123", "model-456"}, want: "api-123/model-456"},
		{name: "route", parts: []string{"api-123", "route-456"}, want: "api-123/route-456"},
		{name: "route response", parts: []string{"api-123", "route-456", "response-789"}, want: "api-123/route-456/response-789"},
		{name: "integration", parts: []string{"api-123", "integration-456"}, want: "api-123/integration-456"},
		{name: "integration response", parts: []string{"api-123", "integration-456", "response-789"}, want: "api-123/integration-456/response-789"},
		{name: "deployment", parts: []string{"api-123", "deployment-456"}, want: "api-123/deployment-456"},
		{name: "api mapping", parts: []string{"mapping-123", "api.example.com"}, want: "mapping-123/api.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apiGatewayV2ImportID(tt.parts...); got != tt.want {
				t.Fatalf("apiGatewayV2ImportID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIGatewayV2ResourceNamePreservesSegments(t *testing.T) {
	if got, want := apiGatewayV2ResourceName("default", "x"), "default/x"; got != want {
		t.Fatalf("apiGatewayV2ResourceName() = %q, want %q", got, want)
	}
	if got, want := apiGatewayV2ResourceName("api", "", "route/default"), "api/route/default"; got != want {
		t.Fatalf("apiGatewayV2ResourceName() = %q, want %q", got, want)
	}
}

func TestAPIGatewayV2ProcessStagesUsesV2ResourceType(t *testing.T) {
	g := &APIGatewayV2Generator{}
	g.processStages([]types.Stage{{StageName: aws.String("prod")}}, "api-123")
	if len(g.Resources) != 1 {
		t.Fatalf("len(Resources) = %d, want 1", len(g.Resources))
	}
	resource := g.Resources[0]
	if got, want := resource.InstanceInfo.Type, "aws_apigatewayv2_stage"; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.ID, "api-123/prod"; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["name"], "prod"; got != want {
		t.Fatalf("stage name attr = %q, want %q", got, want)
	}
}

func TestAPIGatewayV2SkipsManagedQuickCreateResources(t *testing.T) {
	g := &APIGatewayV2Generator{}
	g.processStages([]types.Stage{{
		ApiGatewayManaged: aws.Bool(true),
		StageName:         aws.String("$default"),
	}}, "api-123")
	g.processRoutes(nil, []types.Route{{
		ApiGatewayManaged: aws.Bool(true),
		RouteId:           aws.String("route-1"),
	}}, "api-123")
	g.processIntegrations(nil, []types.Integration{{
		ApiGatewayManaged: aws.Bool(true),
		IntegrationId:     aws.String("integration-1"),
	}}, "api-123")

	if len(g.Resources) != 0 {
		t.Fatalf("len(Resources) = %d, want 0", len(g.Resources))
	}
}

func TestAPIGatewayV2ProcessChildResources(t *testing.T) {
	g := &APIGatewayV2Generator{}
	g.processModels([]types.Model{{
		ModelId:     aws.String("model-1"),
		Name:        aws.String("Pet"),
		ContentType: aws.String("application/json"),
		Schema:      aws.String("{}"),
	}}, "api-123")
	g.processRouteResponses([]types.RouteResponse{{
		RouteResponseId:  aws.String("response-1"),
		RouteResponseKey: aws.String("$default"),
	}}, "api-123", "route-1")
	g.processIntegrationResponses([]types.IntegrationResponse{{
		IntegrationResponseId:  aws.String("response-2"),
		IntegrationResponseKey: aws.String("/200/"),
	}}, "api-123", "integration-1")
	g.processDeployments([]types.Deployment{{
		DeploymentId: aws.String("deployment-1"),
	}}, "api-123")
	g.processAPIMappings([]types.ApiMapping{{
		ApiId:         aws.String("api-123"),
		ApiMappingId:  aws.String("mapping-1"),
		ApiMappingKey: aws.String("v1"),
		Stage:         aws.String("prod"),
	}}, "api.example.com")

	wantIDs := map[string]string{
		"aws_apigatewayv2_model":                "api-123/model-1",
		"aws_apigatewayv2_route_response":       "api-123/route-1/response-1",
		"aws_apigatewayv2_integration_response": "api-123/integration-1/response-2",
		"aws_apigatewayv2_deployment":           "api-123/deployment-1",
		"aws_apigatewayv2_api_mapping":          "mapping-1/api.example.com",
	}
	if len(g.Resources) != len(wantIDs) {
		t.Fatalf("len(Resources) = %d, want %d", len(g.Resources), len(wantIDs))
	}
	for _, resource := range g.Resources {
		wantID, ok := wantIDs[resource.InstanceInfo.Type]
		if !ok {
			t.Fatalf("unexpected resource type %q", resource.InstanceInfo.Type)
		}
		if resource.InstanceState.ID != wantID {
			t.Fatalf("%s ID = %q, want %q", resource.InstanceInfo.Type, resource.InstanceState.ID, wantID)
		}
	}
}

func TestAPIGatewayV2ResourceMissing(t *testing.T) {
	if !apiGatewayV2ResourceMissing(&types.NotFoundException{}) {
		t.Fatal("apiGatewayV2ResourceMissing() = false, want true for NotFoundException")
	}
	if apiGatewayV2ResourceMissing(errors.New("boom")) {
		t.Fatal("apiGatewayV2ResourceMissing() = true, want false for generic error")
	}
	if apiGatewayV2ResourceMissing(nil) {
		t.Fatal("apiGatewayV2ResourceMissing() = true, want false for nil error")
	}
}
