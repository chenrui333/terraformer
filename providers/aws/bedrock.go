// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	bedrockGuardrailResourceType                  = "aws_bedrock_guardrail"
	bedrockInferenceProfileResourceType           = "aws_bedrock_inference_profile"
	bedrockProvisionedModelThroughputResourceType = "aws_bedrock_provisioned_model_throughput"
	bedrockGuardrailImportIDSeparator             = ","
	bedrockResourceNameFallback                   = "bedrock-resource"
)

var bedrockAllowEmptyValues = []string{"tags."}

type BedrockGenerator struct {
	AWSService
}

func (g *BedrockGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := bedrock.NewFromConfig(config)

	if err := g.loadGuardrails(svc); err != nil {
		return err
	}
	if err := g.loadInferenceProfiles(svc); err != nil {
		return err
	}
	if err := g.loadProvisionedModelThroughputs(svc); err != nil {
		return err
	}

	return nil
}

func (g *BedrockGenerator) loadGuardrails(svc *bedrock.Client) error {
	p := bedrock.NewListGuardrailsPaginator(svc, &bedrock.ListGuardrailsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, guardrail := range page.Guardrails {
			if resource, ok := newBedrockGuardrailResource(guardrail); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *BedrockGenerator) loadInferenceProfiles(svc *bedrock.Client) error {
	p := bedrock.NewListInferenceProfilesPaginator(svc, &bedrock.ListInferenceProfilesInput{
		TypeEquals: bedrocktypes.InferenceProfileTypeApplication,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, profile := range page.InferenceProfileSummaries {
			if resource, ok := newBedrockInferenceProfileResource(profile); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *BedrockGenerator) loadProvisionedModelThroughputs(svc *bedrock.Client) error {
	p := bedrock.NewListProvisionedModelThroughputsPaginator(svc, &bedrock.ListProvisionedModelThroughputsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, throughput := range page.ProvisionedModelSummaries {
			if resource, ok := newBedrockProvisionedModelThroughputResource(throughput); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newBedrockGuardrailResource(guardrail bedrocktypes.GuardrailSummary) (terraformutils.Resource, bool) {
	guardrailID := StringValue(guardrail.Id)
	version := StringValue(guardrail.Version)
	if guardrailID == "" || version == "" || !bedrockGuardrailImportable(guardrail.Status) {
		return terraformutils.Resource{}, false
	}
	name := StringValue(guardrail.Name)
	if name == "" {
		name = guardrailID
	}
	return terraformutils.NewResource(
		bedrockGuardrailImportID(guardrailID, version),
		bedrockResourceName("guardrail", name, guardrailID, version),
		bedrockGuardrailResourceType,
		"aws",
		map[string]string{
			"guardrail_id": guardrailID,
			"version":      version,
		},
		bedrockAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newBedrockInferenceProfileResource(profile bedrocktypes.InferenceProfileSummary) (terraformutils.Resource, bool) {
	profileID := StringValue(profile.InferenceProfileId)
	if profileID == "" || !bedrockInferenceProfileImportable(profile) {
		return terraformutils.Resource{}, false
	}
	name := StringValue(profile.InferenceProfileName)
	if name == "" {
		name = profileID
	}
	return terraformutils.NewResource(
		profileID,
		bedrockResourceName("inference-profile", name, profileID),
		bedrockInferenceProfileResourceType,
		"aws",
		map[string]string{
			"id":   profileID,
			"name": name,
		},
		bedrockAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newBedrockProvisionedModelThroughputResource(throughput bedrocktypes.ProvisionedModelSummary) (terraformutils.Resource, bool) {
	provisionedModelARN := StringValue(throughput.ProvisionedModelArn)
	modelARN := StringValue(throughput.ModelArn)
	name := StringValue(throughput.ProvisionedModelName)
	if provisionedModelARN == "" || modelARN == "" || name == "" || throughput.ModelUnits == nil || !bedrockProvisionedModelThroughputImportable(throughput.Status) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		provisionedModelARN,
		bedrockResourceName("provisioned-model-throughput", name, provisionedModelARN),
		bedrockProvisionedModelThroughputResourceType,
		"aws",
		map[string]string{
			"id":                     provisionedModelARN,
			"model_arn":              modelARN,
			"model_units":            strconv.Itoa(int(*throughput.ModelUnits)),
			"provisioned_model_arn":  provisionedModelARN,
			"provisioned_model_name": name,
		},
		bedrockAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func bedrockGuardrailImportID(guardrailID, version string) string {
	return guardrailID + bedrockGuardrailImportIDSeparator + version
}

func bedrockResourceName(parts ...string) string {
	var cleanParts []string
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d-%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return bedrockResourceNameFallback
	}
	return strings.Join(cleanParts, "/")
}

func bedrockGuardrailImportable(status bedrocktypes.GuardrailStatus) bool {
	return status == bedrocktypes.GuardrailStatusReady
}

func bedrockInferenceProfileImportable(profile bedrocktypes.InferenceProfileSummary) bool {
	return profile.Type == bedrocktypes.InferenceProfileTypeApplication && profile.Status == bedrocktypes.InferenceProfileStatusActive
}

func bedrockProvisionedModelThroughputImportable(status bedrocktypes.ProvisionedModelStatus) bool {
	return status == bedrocktypes.ProvisionedModelStatusInService
}

func bedrockResourceNotFound(err error) bool {
	var notFound *bedrocktypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
