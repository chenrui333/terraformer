// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestBedrockGuardrailImportID(t *testing.T) {
	got := bedrockGuardrailImportID("gr-1234567890", "DRAFT")
	want := "gr-1234567890,DRAFT"
	if got != want {
		t.Fatalf("bedrockGuardrailImportID() = %q, want %q", got, want)
	}
}

func TestBedrockResourceNameFallback(t *testing.T) {
	if got := bedrockResourceName("", ""); got != bedrockResourceNameFallback {
		t.Fatalf("bedrockResourceName() = %q, want %q", got, bedrockResourceNameFallback)
	}
}

func TestBedrockResourceNameUniqueness(t *testing.T) {
	first := terraformutils.TfSanitize(bedrockResourceName("ab", "c"))
	second := terraformutils.TfSanitize(bedrockResourceName("a", "bc"))
	if first == second {
		t.Fatalf("bedrockResourceName() collision after sanitize: %q", first)
	}
}

func TestBedrockShouldLoadResourceHonorsTypedFilters(t *testing.T) {
	g := BedrockGenerator{}
	if !g.shouldLoadBedrockResource("bedrock_guardrail") ||
		!g.shouldLoadBedrockResource("bedrock_inference_profile") ||
		!g.shouldLoadBedrockResource("bedrock_provisioned_model_throughput") {
		t.Fatal("without typed filters, all Bedrock resource families should be loaded")
	}

	g.Filter = []terraformutils.ResourceFilter{{
		ServiceName:      "bedrock_guardrail",
		FieldPath:        "id",
		AcceptableValues: []string{"gr-1234567890,DRAFT"},
	}}
	if !g.shouldLoadBedrockResource("bedrock_guardrail") {
		t.Fatal("typed guardrail filter should load guardrails")
	}
	if g.shouldLoadBedrockResource("bedrock_inference_profile") {
		t.Fatal("typed guardrail filter should not load inference profiles")
	}
	if g.shouldLoadBedrockResource("bedrock_provisioned_model_throughput") {
		t.Fatal("typed guardrail filter should not load provisioned throughputs")
	}
}

func TestBedrockShouldLoadResourceAllowsUntypedFilters(t *testing.T) {
	tests := []struct {
		name   string
		filter terraformutils.ResourceFilter
	}{
		{
			name: "id",
			filter: terraformutils.ResourceFilter{
				FieldPath:        "id",
				AcceptableValues: []string{"arn:aws:bedrock:us-east-1:123456789012:provisioned-model/abc123"},
			},
		},
		{
			name: "post-refresh attribute",
			filter: terraformutils.ResourceFilter{
				FieldPath:        "tags.env",
				AcceptableValues: []string{"prod"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := BedrockGenerator{
				AWSService: AWSService{
					Service: terraformutils.Service{
						Filter: []terraformutils.ResourceFilter{
							{
								ServiceName:      "bedrock_guardrail",
								FieldPath:        "id",
								AcceptableValues: []string{"gr-1234567890,DRAFT"},
							},
							tt.filter,
						},
					},
				},
			}
			if !g.shouldLoadBedrockResource("bedrock_inference_profile") {
				t.Fatal("untyped filter should keep broad Bedrock discovery available")
			}
		})
	}
}

func TestNewBedrockGuardrailResource(t *testing.T) {
	resource, ok := newBedrockGuardrailResource(bedrocktypes.GuardrailSummary{
		Id:      aws.String("gr-1234567890"),
		Name:    aws.String("billing-policy"),
		Status:  bedrocktypes.GuardrailStatusReady,
		Version: aws.String("DRAFT"),
	})
	assertBedrockResource(t, resource, ok, "gr-1234567890,DRAFT", bedrockGuardrailResourceType)
	if got := resource.InstanceState.Attributes["guardrail_id"]; got != "gr-1234567890" {
		t.Fatalf("guardrail_id attribute = %q, want gr-1234567890", got)
	}
	if got := resource.InstanceState.Attributes["version"]; got != "DRAFT" {
		t.Fatalf("version attribute = %q, want DRAFT", got)
	}

	if _, ok := newBedrockGuardrailResource(bedrocktypes.GuardrailSummary{
		Status:  bedrocktypes.GuardrailStatusReady,
		Version: aws.String("DRAFT"),
	}); ok {
		t.Fatal("guardrail without ID should be skipped")
	}
	if _, ok := newBedrockGuardrailResource(bedrocktypes.GuardrailSummary{
		Id:      aws.String("gr-1234567890"),
		Status:  bedrocktypes.GuardrailStatusReady,
		Version: aws.String(""),
	}); ok {
		t.Fatal("guardrail without version should be skipped")
	}
	if _, ok := newBedrockGuardrailResource(bedrocktypes.GuardrailSummary{
		Id:      aws.String("gr-1234567890"),
		Status:  bedrocktypes.GuardrailStatusDeleting,
		Version: aws.String("DRAFT"),
	}); ok {
		t.Fatal("deleting guardrail should be skipped")
	}
}

func TestNewBedrockInferenceProfileResource(t *testing.T) {
	resource, ok := newBedrockInferenceProfileResource(bedrocktypes.InferenceProfileSummary{
		InferenceProfileId:   aws.String("ip-1234567890"),
		InferenceProfileName: aws.String("application-profile"),
		Status:               bedrocktypes.InferenceProfileStatusActive,
		Type:                 bedrocktypes.InferenceProfileTypeApplication,
	})
	assertBedrockResource(t, resource, ok, "ip-1234567890", bedrockInferenceProfileResourceType)
	if got := resource.InstanceState.Attributes["id"]; got != "ip-1234567890" {
		t.Fatalf("id attribute = %q, want ip-1234567890", got)
	}
	if got := resource.InstanceState.Attributes["name"]; got != "application-profile" {
		t.Fatalf("name attribute = %q, want application-profile", got)
	}

	if _, ok := newBedrockInferenceProfileResource(bedrocktypes.InferenceProfileSummary{
		Status: bedrocktypes.InferenceProfileStatusActive,
		Type:   bedrocktypes.InferenceProfileTypeApplication,
	}); ok {
		t.Fatal("inference profile without ID should be skipped")
	}
	if _, ok := newBedrockInferenceProfileResource(bedrocktypes.InferenceProfileSummary{
		InferenceProfileId: aws.String("us.amazon.nova-pro-v1:0"),
		Status:             bedrocktypes.InferenceProfileStatusActive,
		Type:               bedrocktypes.InferenceProfileTypeSystemDefined,
	}); ok {
		t.Fatal("system-defined inference profile should be skipped")
	}
}

func TestNewBedrockProvisionedModelThroughputResource(t *testing.T) {
	modelUnits := int32(2)
	resource, ok := newBedrockProvisionedModelThroughputResource(bedrocktypes.ProvisionedModelSummary{
		ModelArn:             aws.String("arn:aws:bedrock:us-east-1::foundation-model/amazon.nova-pro-v1:0"),
		ModelUnits:           &modelUnits,
		ProvisionedModelArn:  aws.String("arn:aws:bedrock:us-east-1:123456789012:provisioned-model/abc123"),
		ProvisionedModelName: aws.String("prod-throughput"),
		Status:               bedrocktypes.ProvisionedModelStatusInService,
	})
	assertBedrockResource(t, resource, ok, "arn:aws:bedrock:us-east-1:123456789012:provisioned-model/abc123", bedrockProvisionedModelThroughputResourceType)
	if got := resource.InstanceState.Attributes["model_units"]; got != "2" {
		t.Fatalf("model_units attribute = %q, want 2", got)
	}
	if got := resource.InstanceState.Attributes["model_arn"]; got == "" {
		t.Fatal("model_arn attribute should be seeded")
	}

	if _, ok := newBedrockProvisionedModelThroughputResource(bedrocktypes.ProvisionedModelSummary{
		ModelArn:             aws.String("arn:aws:bedrock:us-east-1::foundation-model/amazon.nova-pro-v1:0"),
		ProvisionedModelArn:  aws.String("arn:aws:bedrock:us-east-1:123456789012:provisioned-model/abc123"),
		ProvisionedModelName: aws.String("prod-throughput"),
		Status:               bedrocktypes.ProvisionedModelStatusInService,
	}); ok {
		t.Fatal("throughput without model units should be skipped")
	}
	if _, ok := newBedrockProvisionedModelThroughputResource(bedrocktypes.ProvisionedModelSummary{
		ModelArn:             aws.String("arn:aws:bedrock:us-east-1::foundation-model/amazon.nova-pro-v1:0"),
		ModelUnits:           &modelUnits,
		ProvisionedModelArn:  aws.String("arn:aws:bedrock:us-east-1:123456789012:provisioned-model/abc123"),
		ProvisionedModelName: aws.String("prod-throughput"),
		Status:               bedrocktypes.ProvisionedModelStatusFailed,
	}); ok {
		t.Fatal("failed throughput should be skipped")
	}
}

func TestBedrockImportableStatuses(t *testing.T) {
	if !bedrockGuardrailImportable(bedrocktypes.GuardrailStatusReady) {
		t.Fatal("READY guardrail should be importable")
	}
	for _, status := range []bedrocktypes.GuardrailStatus{
		bedrocktypes.GuardrailStatusCreating,
		bedrocktypes.GuardrailStatusUpdating,
		bedrocktypes.GuardrailStatusVersioning,
		bedrocktypes.GuardrailStatusFailed,
		bedrocktypes.GuardrailStatusDeleting,
	} {
		if bedrockGuardrailImportable(status) {
			t.Fatalf("%s guardrail should not be importable", status)
		}
	}

	if !bedrockProvisionedModelThroughputImportable(bedrocktypes.ProvisionedModelStatusInService) {
		t.Fatal("InService throughput should be importable")
	}
	for _, status := range []bedrocktypes.ProvisionedModelStatus{
		bedrocktypes.ProvisionedModelStatusCreating,
		bedrocktypes.ProvisionedModelStatusUpdating,
		bedrocktypes.ProvisionedModelStatusFailed,
	} {
		if bedrockProvisionedModelThroughputImportable(status) {
			t.Fatalf("%s throughput should not be importable", status)
		}
	}
}

func TestBedrockResourceNotFound(t *testing.T) {
	if !bedrockResourceNotFound(&bedrocktypes.ResourceNotFoundException{}) {
		t.Fatal("ResourceNotFoundException should be detected")
	}
	if !bedrockResourceNotFound(errors.Join(errors.New("lookup failed"), &bedrocktypes.ResourceNotFoundException{})) {
		t.Fatal("wrapped ResourceNotFoundException should be detected")
	}
	if bedrockResourceNotFound(errors.New("other error")) {
		t.Fatal("non-not-found error should not be detected")
	}
}

func TestBedrockInitialCleanupHonorsTypedFilters(t *testing.T) {
	guardrail, ok := newBedrockGuardrailResource(bedrocktypes.GuardrailSummary{
		Id:      aws.String("gr-1234567890"),
		Name:    aws.String("billing-policy"),
		Status:  bedrocktypes.GuardrailStatusReady,
		Version: aws.String("DRAFT"),
	})
	if !ok {
		t.Fatal("newBedrockGuardrailResource() should create guardrail")
	}
	profile, ok := newBedrockInferenceProfileResource(bedrocktypes.InferenceProfileSummary{
		InferenceProfileId:   aws.String("ip-1234567890"),
		InferenceProfileName: aws.String("application-profile"),
		Status:               bedrocktypes.InferenceProfileStatusActive,
		Type:                 bedrocktypes.InferenceProfileTypeApplication,
	})
	if !ok {
		t.Fatal("newBedrockInferenceProfileResource() should create profile")
	}
	modelUnits := int32(2)
	throughput, ok := newBedrockProvisionedModelThroughputResource(bedrocktypes.ProvisionedModelSummary{
		ModelArn:             aws.String("arn:aws:bedrock:us-east-1::foundation-model/amazon.nova-pro-v1:0"),
		ModelUnits:           &modelUnits,
		ProvisionedModelArn:  aws.String("arn:aws:bedrock:us-east-1:123456789012:provisioned-model/abc123"),
		ProvisionedModelName: aws.String("prod-throughput"),
		Status:               bedrocktypes.ProvisionedModelStatusInService,
	})
	if !ok {
		t.Fatal("newBedrockProvisionedModelThroughputResource() should create throughput")
	}

	g := BedrockGenerator{}
	g.Resources = []terraformutils.Resource{guardrail, profile, throughput}
	g.Filter = []terraformutils.ResourceFilter{{
		ServiceName:      "bedrock_guardrail",
		FieldPath:        "id",
		AcceptableValues: []string{"gr-1234567890,DRAFT"},
	}}

	g.InitialCleanup()

	if len(g.Resources) != 1 {
		t.Fatalf("InitialCleanup() resources len = %d, want 1", len(g.Resources))
	}
	if got := g.Resources[0].InstanceInfo.Type; got != bedrockGuardrailResourceType {
		t.Fatalf("InitialCleanup() kept resource type = %q, want %s", got, bedrockGuardrailResourceType)
	}
}

func TestBedrockInitialCleanupPreservesGlobalFilters(t *testing.T) {
	guardrail, profile, throughput := bedrockTestResources(t)
	g := BedrockGenerator{}
	g.Resources = []terraformutils.Resource{guardrail, profile, throughput}
	g.Filter = []terraformutils.ResourceFilter{
		{
			ServiceName:      "bedrock_guardrail",
			FieldPath:        "id",
			AcceptableValues: []string{"gr-1234567890,DRAFT"},
		},
		{
			FieldPath:        "tags.env",
			AcceptableValues: []string{"prod"},
		},
	}

	g.InitialCleanup()

	if len(g.Resources) != 3 {
		t.Fatalf("InitialCleanup() resources len = %d, want 3", len(g.Resources))
	}
}

func assertBedrockResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantType string) {
	t.Helper()
	if !ok {
		t.Fatal("resource should be created")
	}
	if got := resource.InstanceState.ID; got != wantID {
		t.Fatalf("resource ID = %q, want %q", got, wantID)
	}
	if got := resource.InstanceInfo.Type; got != wantType {
		t.Fatalf("resource type = %q, want %q", got, wantType)
	}
	if resource.ResourceName == "" {
		t.Fatal("resource name should not be empty")
	}
}

func bedrockTestResources(t *testing.T) (terraformutils.Resource, terraformutils.Resource, terraformutils.Resource) {
	t.Helper()
	guardrail, ok := newBedrockGuardrailResource(bedrocktypes.GuardrailSummary{
		Id:      aws.String("gr-1234567890"),
		Name:    aws.String("billing-policy"),
		Status:  bedrocktypes.GuardrailStatusReady,
		Version: aws.String("DRAFT"),
	})
	if !ok {
		t.Fatal("newBedrockGuardrailResource() should create guardrail")
	}
	profile, ok := newBedrockInferenceProfileResource(bedrocktypes.InferenceProfileSummary{
		InferenceProfileId:   aws.String("ip-1234567890"),
		InferenceProfileName: aws.String("application-profile"),
		Status:               bedrocktypes.InferenceProfileStatusActive,
		Type:                 bedrocktypes.InferenceProfileTypeApplication,
	})
	if !ok {
		t.Fatal("newBedrockInferenceProfileResource() should create profile")
	}
	modelUnits := int32(2)
	throughput, ok := newBedrockProvisionedModelThroughputResource(bedrocktypes.ProvisionedModelSummary{
		ModelArn:             aws.String("arn:aws:bedrock:us-east-1::foundation-model/amazon.nova-pro-v1:0"),
		ModelUnits:           &modelUnits,
		ProvisionedModelArn:  aws.String("arn:aws:bedrock:us-east-1:123456789012:provisioned-model/abc123"),
		ProvisionedModelName: aws.String("prod-throughput"),
		Status:               bedrocktypes.ProvisionedModelStatusInService,
	})
	if !ok {
		t.Fatal("newBedrockProvisionedModelThroughputResource() should create throughput")
	}
	return guardrail, profile, throughput
}
