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
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
)

const (
	bedrockGuardrailResourceType                  = "aws_bedrock_guardrail"
	bedrockGuardrailVersionResourceType           = "aws_bedrock_guardrail_version"
	bedrockInferenceProfileResourceType           = "aws_bedrock_inference_profile"
	bedrockModelInvocationLoggingResourceType     = "aws_bedrock_model_invocation_logging_configuration"
	bedrockProvisionedModelThroughputResourceType = "aws_bedrock_provisioned_model_throughput"
	bedrockGuardrailDraftVersion                  = "DRAFT"
	bedrockGuardrailImportIDSeparator             = ","
	bedrockResourceNameFallback                   = "bedrock-resource"
)

var (
	bedrockAllowEmptyValues = []string{"tags."}
	bedrockResourceTypes    = []string{
		bedrockServiceName(bedrockGuardrailResourceType),
		bedrockServiceName(bedrockGuardrailVersionResourceType),
		bedrockServiceName(bedrockInferenceProfileResourceType),
		bedrockServiceName(bedrockModelInvocationLoggingResourceType),
		bedrockServiceName(bedrockProvisionedModelThroughputResourceType),
	}
)

type BedrockGenerator struct {
	AWSService
}

func (g *BedrockGenerator) InitialCleanup() {
	if len(g.Filter) == 0 {
		return
	}
	filteredResources := []terraformutils.Resource{}
	for _, resource := range g.Resources {
		serviceName := bedrockServiceName(resource.InstanceInfo.Type)
		if g.hasTypedBedrockFilter() && !g.hasTypedFilterFor(serviceName) && !g.hasUntypedFilter() {
			continue
		}
		allPredicatesTrue := true
		for _, filter := range g.Filter {
			if filter.FieldPath != "id" {
				continue
			}
			allPredicatesTrue = allPredicatesTrue && filter.Filter(resource)
		}
		if allPredicatesTrue && !terraformutils.ContainsResource(filteredResources, resource) {
			filteredResources = append(filteredResources, resource)
		}
	}
	g.Resources = filteredResources
}

func (g *BedrockGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := bedrock.NewFromConfig(config)

	loadGuardrails := g.shouldLoadBedrockResource(bedrockServiceName(bedrockGuardrailResourceType))
	loadGuardrailVersions := g.shouldLoadBedrockResource(bedrockServiceName(bedrockGuardrailVersionResourceType))
	if loadGuardrails || loadGuardrailVersions {
		guardrails, err := listBedrockGuardrails(svc, "")
		if err != nil {
			return err
		}
		if loadGuardrails {
			g.loadGuardrails(guardrails)
		}
		if loadGuardrailVersions {
			if err := g.loadGuardrailVersions(svc, guardrails); err != nil {
				return err
			}
		}
	}
	if g.shouldLoadBedrockResource(bedrockServiceName(bedrockInferenceProfileResourceType)) {
		if err := g.loadInferenceProfiles(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadBedrockResource(bedrockServiceName(bedrockModelInvocationLoggingResourceType)) {
		if err := g.loadModelInvocationLoggingConfiguration(svc, config.Region); err != nil {
			return err
		}
	}
	if g.shouldLoadBedrockResource(bedrockServiceName(bedrockProvisionedModelThroughputResourceType)) {
		if err := g.loadProvisionedModelThroughputs(svc); err != nil {
			return err
		}
	}

	return nil
}

func (g *BedrockGenerator) shouldLoadBedrockResource(serviceName string) bool {
	if !g.hasTypedBedrockFilter() {
		return true
	}
	return g.hasTypedFilterFor(serviceName) || g.hasUntypedFilter()
}

func (g *BedrockGenerator) hasTypedBedrockFilter() bool {
	for _, serviceName := range bedrockResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *BedrockGenerator) hasTypedFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName {
			return true
		}
	}
	return false
}

func (g *BedrockGenerator) hasUntypedFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "" {
			return true
		}
	}
	return false
}

func bedrockServiceName(resourceType string) string {
	return strings.TrimPrefix(resourceType, "aws_")
}

func listBedrockGuardrails(svc *bedrock.Client, guardrailIdentifier string) ([]bedrocktypes.GuardrailSummary, error) {
	input := &bedrock.ListGuardrailsInput{}
	if guardrailIdentifier != "" {
		input.GuardrailIdentifier = &guardrailIdentifier
	}
	p := bedrock.NewListGuardrailsPaginator(svc, input)
	guardrails := []bedrocktypes.GuardrailSummary{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		guardrails = append(guardrails, page.Guardrails...)
	}
	return guardrails, nil
}

func (g *BedrockGenerator) loadGuardrails(guardrails []bedrocktypes.GuardrailSummary) {
	for _, guardrail := range guardrails {
		if resource, ok := newBedrockGuardrailResource(guardrail); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
}

func (g *BedrockGenerator) loadGuardrailVersions(svc *bedrock.Client, guardrails []bedrocktypes.GuardrailSummary) error {
	for _, guardrail := range guardrails {
		guardrailARN := StringValue(guardrail.Arn)
		if guardrailARN == "" {
			continue
		}
		versions, err := listBedrockGuardrails(svc, guardrailARN)
		if err != nil {
			if bedrockResourceNotFound(err) {
				continue
			}
			return err
		}
		for _, version := range versions {
			if resource, ok := newBedrockGuardrailVersionResource(version); ok {
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

func (g *BedrockGenerator) loadModelInvocationLoggingConfiguration(svc *bedrock.Client, region string) error {
	output, err := svc.GetModelInvocationLoggingConfiguration(context.TODO(), &bedrock.GetModelInvocationLoggingConfigurationInput{})
	if err != nil {
		if bedrockResourceNotFound(err) {
			return nil
		}
		return err
	}
	if resource, ok := newBedrockModelInvocationLoggingResource(region, output); ok {
		g.Resources = append(g.Resources, resource)
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

func newBedrockGuardrailVersionResource(guardrail bedrocktypes.GuardrailSummary) (terraformutils.Resource, bool) {
	guardrailARN := StringValue(guardrail.Arn)
	version := StringValue(guardrail.Version)
	if guardrailARN == "" || !bedrockGuardrailVersionImportable(guardrail) {
		return terraformutils.Resource{}, false
	}
	name := StringValue(guardrail.Name)
	if name == "" {
		name = StringValue(guardrail.Id)
	}
	if name == "" {
		name = guardrailARN
	}
	resource := terraformutils.NewResource(
		bedrockGuardrailVersionImportID(guardrailARN, version),
		bedrockResourceName("guardrail-version", name, guardrailARN, version),
		bedrockGuardrailVersionResourceType,
		"aws",
		map[string]string{
			"guardrail_arn": guardrailARN,
			"version":       version,
		},
		bedrockAllowEmptyValues,
		map[string]interface{}{},
	)
	if resource.InstanceState.Meta == nil {
		resource.InstanceState.Meta = map[string]interface{}{}
	}
	resource.InstanceState.Meta[tfcompat.MetaKeyPreserveIDAfterRefresh] = true
	return resource, true
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

func newBedrockModelInvocationLoggingResource(region string, output *bedrock.GetModelInvocationLoggingConfigurationOutput) (terraformutils.Resource, bool) {
	if region == "" || !bedrockModelInvocationLoggingConfigured(output) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		bedrockModelInvocationLoggingImportID(region),
		bedrockResourceName("model-invocation-logging-configuration", region),
		bedrockModelInvocationLoggingResourceType,
		"aws",
		bedrockAllowEmptyValues,
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

func bedrockGuardrailVersionImportID(guardrailARN, version string) string {
	return bedrockGuardrailImportID(guardrailARN, version)
}

func bedrockModelInvocationLoggingImportID(region string) string {
	return region
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

func bedrockGuardrailVersionImportable(guardrail bedrocktypes.GuardrailSummary) bool {
	version := StringValue(guardrail.Version)
	return version != "" && bedrockGuardrailImportable(guardrail.Status) && !strings.EqualFold(version, bedrockGuardrailDraftVersion)
}

func bedrockInferenceProfileImportable(profile bedrocktypes.InferenceProfileSummary) bool {
	return profile.Type == bedrocktypes.InferenceProfileTypeApplication && profile.Status == bedrocktypes.InferenceProfileStatusActive
}

func bedrockModelInvocationLoggingConfigured(output *bedrock.GetModelInvocationLoggingConfigurationOutput) bool {
	return output != nil && output.LoggingConfig != nil
}

func bedrockProvisionedModelThroughputImportable(status bedrocktypes.ProvisionedModelStatus) bool {
	return status == bedrocktypes.ProvisionedModelStatusInService
}

func bedrockResourceNotFound(err error) bool {
	var notFound *bedrocktypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
