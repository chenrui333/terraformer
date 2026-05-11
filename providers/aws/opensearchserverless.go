// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/opensearchserverless"
	opensearchserverlesstypes "github.com/aws/aws-sdk-go-v2/service/opensearchserverless/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	openSearchServerlessCollectionResourceType      = "aws_opensearchserverless_collection"
	openSearchServerlessAccessPolicyResourceType    = "aws_opensearchserverless_access_policy"
	openSearchServerlessSecurityPolicyResourceType  = "aws_opensearchserverless_security_policy"
	openSearchServerlessSecurityConfigResourceType  = "aws_opensearchserverless_security_config"
	openSearchServerlessLifecyclePolicyResourceType = "aws_opensearchserverless_lifecycle_policy"
	openSearchServerlessVPCEndpointResourceType     = "aws_opensearchserverless_vpc_endpoint"

	openSearchServerlessResourceNameFallback = "opensearchserverless-resource"
)

var openSearchServerlessAllowEmptyValues = []string{
	"tags.",
}

type OpenSearchServerlessGenerator struct {
	AWSService
}

type openSearchServerlessOptionalResourceLoader struct {
	name string
	load func() error
}

func (g *OpenSearchServerlessGenerator) loadOptionalResources(loaders []openSearchServerlessOptionalResourceLoader) error {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			if openSearchServerlessOptionalResourceErrorSkippable(err) {
				log.Printf("Skipping OpenSearch Serverless %s: %v", loader.name, err)
				continue
			}
			log.Printf("Failed OpenSearch Serverless %s discovery: %v", loader.name, err)
			return fmt.Errorf("loading OpenSearch Serverless %s: %w", loader.name, err)
		}
	}
	return nil
}

func openSearchServerlessOptionalResourceErrorSkippable(err error) bool {
	var notFound *opensearchserverlesstypes.ResourceNotFoundException
	if errors.As(err, &notFound) {
		return true
	}
	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && strings.Contains(strings.ToLower(apiErr.ErrorCode()), "accessdenied")
}

func (g *OpenSearchServerlessGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := opensearchserverless.NewFromConfig(config)
	ec2Svc := ec2.NewFromConfig(config)

	return g.loadOptionalResources([]openSearchServerlessOptionalResourceLoader{
		{name: "collections", load: func() error { return g.loadCollections(svc) }},
		{name: "access policies", load: func() error { return g.loadAccessPolicies(svc) }},
		{name: "security policies", load: func() error { return g.loadSecurityPolicies(svc) }},
		{name: "security configs", load: func() error { return g.loadSecurityConfigs(svc) }},
		{name: "lifecycle policies", load: func() error { return g.loadLifecyclePolicies(svc) }},
		{name: "VPC endpoints", load: func() error { return g.loadVPCEndpoints(svc, ec2Svc) }},
	})
}

func (g *OpenSearchServerlessGenerator) PostConvertHook() error {
	for i := range g.Resources {
		if g.Resources[i].InstanceInfo == nil {
			continue
		}
		switch g.Resources[i].InstanceInfo.Type {
		case openSearchServerlessAccessPolicyResourceType,
			openSearchServerlessSecurityPolicyResourceType,
			openSearchServerlessLifecyclePolicyResourceType:
			wrapOpenSearchServerlessPolicyHeredoc(g, &g.Resources[i])
		}
	}
	return nil
}

func (g *OpenSearchServerlessGenerator) loadCollections(svc *opensearchserverless.Client) error {
	p := opensearchserverless.NewListCollectionsPaginator(svc, &opensearchserverless.ListCollectionsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, chunk := range openSearchServerlessStringChunks(openSearchServerlessCollectionIDs(page.CollectionSummaries), 100) {
			output, err := svc.BatchGetCollection(context.TODO(), &opensearchserverless.BatchGetCollectionInput{
				Ids: chunk,
			})
			if err != nil {
				if openSearchServerlessOptionalResourceErrorSkippable(err) {
					continue
				}
				return err
			}
			for _, collection := range output.CollectionDetails {
				if resource, ok := newOpenSearchServerlessCollectionResource(collection); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func (g *OpenSearchServerlessGenerator) loadAccessPolicies(svc *opensearchserverless.Client) error {
	p := opensearchserverless.NewListAccessPoliciesPaginator(svc, &opensearchserverless.ListAccessPoliciesInput{
		Type: opensearchserverlesstypes.AccessPolicyTypeData,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, summary := range page.AccessPolicySummaries {
			name := StringValue(summary.Name)
			if name == "" {
				continue
			}
			output, err := svc.GetAccessPolicy(context.TODO(), &opensearchserverless.GetAccessPolicyInput{
				Name: &name,
				Type: summary.Type,
			})
			if err != nil {
				if openSearchServerlessOptionalResourceErrorSkippable(err) {
					continue
				}
				return err
			}
			if output.AccessPolicyDetail == nil {
				continue
			}
			if resource, ok := newOpenSearchServerlessAccessPolicyResource(*output.AccessPolicyDetail); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *OpenSearchServerlessGenerator) loadSecurityPolicies(svc *opensearchserverless.Client) error {
	for _, policyType := range []opensearchserverlesstypes.SecurityPolicyType{
		opensearchserverlesstypes.SecurityPolicyTypeEncryption,
		opensearchserverlesstypes.SecurityPolicyTypeNetwork,
	} {
		p := opensearchserverless.NewListSecurityPoliciesPaginator(svc, &opensearchserverless.ListSecurityPoliciesInput{
			Type: policyType,
		})
		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if err != nil {
				return err
			}
			for _, summary := range page.SecurityPolicySummaries {
				name := StringValue(summary.Name)
				if name == "" {
					continue
				}
				output, err := svc.GetSecurityPolicy(context.TODO(), &opensearchserverless.GetSecurityPolicyInput{
					Name: &name,
					Type: summary.Type,
				})
				if err != nil {
					if openSearchServerlessOptionalResourceErrorSkippable(err) {
						continue
					}
					return err
				}
				if output.SecurityPolicyDetail == nil {
					continue
				}
				if resource, ok := newOpenSearchServerlessSecurityPolicyResource(*output.SecurityPolicyDetail); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func (g *OpenSearchServerlessGenerator) loadSecurityConfigs(svc *opensearchserverless.Client) error {
	p := opensearchserverless.NewListSecurityConfigsPaginator(svc, &opensearchserverless.ListSecurityConfigsInput{
		Type: opensearchserverlesstypes.SecurityConfigTypeSaml,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, summary := range page.SecurityConfigSummaries {
			id := StringValue(summary.Id)
			if id == "" {
				continue
			}
			output, err := svc.GetSecurityConfig(context.TODO(), &opensearchserverless.GetSecurityConfigInput{
				Id: &id,
			})
			if err != nil {
				if openSearchServerlessOptionalResourceErrorSkippable(err) {
					continue
				}
				return err
			}
			if output.SecurityConfigDetail == nil {
				continue
			}
			if resource, ok := newOpenSearchServerlessSecurityConfigResource(*output.SecurityConfigDetail); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *OpenSearchServerlessGenerator) loadLifecyclePolicies(svc *opensearchserverless.Client) error {
	p := opensearchserverless.NewListLifecyclePoliciesPaginator(svc, &opensearchserverless.ListLifecyclePoliciesInput{
		Type: opensearchserverlesstypes.LifecyclePolicyTypeRetention,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, summary := range page.LifecyclePolicySummaries {
			name := StringValue(summary.Name)
			if name == "" {
				continue
			}
			output, err := svc.BatchGetLifecyclePolicy(context.TODO(), &opensearchserverless.BatchGetLifecyclePolicyInput{
				Identifiers: []opensearchserverlesstypes.LifecyclePolicyIdentifier{
					{Name: &name, Type: summary.Type},
				},
			})
			if err != nil {
				if openSearchServerlessOptionalResourceErrorSkippable(err) {
					continue
				}
				return err
			}
			for _, policy := range output.LifecyclePolicyDetails {
				if resource, ok := newOpenSearchServerlessLifecyclePolicyResource(policy); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func (g *OpenSearchServerlessGenerator) loadVPCEndpoints(svc *opensearchserverless.Client, ec2Svc *ec2.Client) error {
	p := opensearchserverless.NewListVpcEndpointsPaginator(svc, &opensearchserverless.ListVpcEndpointsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		ids := openSearchServerlessVPCEndpointIDs(page.VpcEndpointSummaries)
		for _, chunk := range openSearchServerlessStringChunks(ids, 100) {
			securityGroupIDs, err := openSearchServerlessEC2VPCEndpointSecurityGroups(context.TODO(), ec2Svc, chunk)
			if err != nil {
				if openSearchServerlessEC2ErrorSkippable(err) {
					log.Printf("Skipping OpenSearch Serverless VPC endpoint discovery: %v", err)
					continue
				}
				return err
			}
			output, err := svc.BatchGetVpcEndpoint(context.TODO(), &opensearchserverless.BatchGetVpcEndpointInput{
				Ids: chunk,
			})
			if err != nil {
				if openSearchServerlessOptionalResourceErrorSkippable(err) {
					continue
				}
				return err
			}
			for _, endpoint := range output.VpcEndpointDetails {
				if resource, ok := newOpenSearchServerlessVPCEndpointResource(endpoint, securityGroupIDs[StringValue(endpoint.Id)]); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func newOpenSearchServerlessCollectionResource(collection opensearchserverlesstypes.CollectionDetail) (terraformutils.Resource, bool) {
	importID := openSearchServerlessCollectionImportID(collection)
	if importID == "" || !openSearchServerlessCollectionImportable(collection) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"name": StringValue(collection.Name),
	}
	if collection.Type != "" {
		attributes["type"] = string(collection.Type)
	}
	if description := StringValue(collection.Description); description != "" {
		attributes["description"] = description
	}
	if collection.StandbyReplicas != "" {
		attributes["standby_replicas"] = string(collection.StandbyReplicas)
	}
	return terraformutils.NewResource(
		importID,
		openSearchServerlessResourceName("collection", StringValue(collection.Name), importID),
		openSearchServerlessCollectionResourceType,
		"aws",
		attributes,
		openSearchServerlessAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newOpenSearchServerlessAccessPolicyResource(policy opensearchserverlesstypes.AccessPolicyDetail) (terraformutils.Resource, bool) {
	importID := openSearchServerlessAccessPolicyImportID(policy)
	policyJSON := openSearchServerlessPolicyDocumentString(policy.Policy)
	if importID == "" || policyJSON == "" {
		return terraformutils.Resource{}, false
	}
	attributes := openSearchServerlessPolicyAttributes(StringValue(policy.Name), string(policy.Type), policyJSON, StringValue(policy.Description))
	return terraformutils.NewResource(
		importID,
		openSearchServerlessResourceName("access-policy", string(policy.Type), StringValue(policy.Name)),
		openSearchServerlessAccessPolicyResourceType,
		"aws",
		attributes,
		openSearchServerlessAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newOpenSearchServerlessSecurityPolicyResource(policy opensearchserverlesstypes.SecurityPolicyDetail) (terraformutils.Resource, bool) {
	importID := openSearchServerlessSecurityPolicyImportID(policy)
	policyJSON := openSearchServerlessPolicyDocumentString(policy.Policy)
	if importID == "" || policyJSON == "" {
		return terraformutils.Resource{}, false
	}
	attributes := openSearchServerlessPolicyAttributes(StringValue(policy.Name), string(policy.Type), policyJSON, StringValue(policy.Description))
	return terraformutils.NewResource(
		importID,
		openSearchServerlessResourceName("security-policy", string(policy.Type), StringValue(policy.Name)),
		openSearchServerlessSecurityPolicyResourceType,
		"aws",
		attributes,
		openSearchServerlessAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newOpenSearchServerlessSecurityConfigResource(config opensearchserverlesstypes.SecurityConfigDetail) (terraformutils.Resource, bool) {
	importID := openSearchServerlessSecurityConfigImportID(config)
	name := openSearchServerlessSecurityConfigName(importID)
	if importID == "" || name == "" || !openSearchServerlessSecurityConfigImportable(config) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"name": name,
		"type": string(config.Type),
	}
	if description := StringValue(config.Description); description != "" {
		attributes["description"] = description
	}
	for key, value := range openSearchServerlessSamlOptionsAttributes("saml_options", config.SamlOptions) {
		attributes[key] = value
	}
	return terraformutils.NewResource(
		importID,
		openSearchServerlessResourceName("security-config", string(config.Type), name),
		openSearchServerlessSecurityConfigResourceType,
		"aws",
		attributes,
		openSearchServerlessAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newOpenSearchServerlessLifecyclePolicyResource(policy opensearchserverlesstypes.LifecyclePolicyDetail) (terraformutils.Resource, bool) {
	importID := openSearchServerlessLifecyclePolicyImportID(policy)
	policyJSON := openSearchServerlessPolicyDocumentString(policy.Policy)
	if importID == "" || policyJSON == "" {
		return terraformutils.Resource{}, false
	}
	attributes := openSearchServerlessPolicyAttributes(StringValue(policy.Name), string(policy.Type), policyJSON, StringValue(policy.Description))
	return terraformutils.NewResource(
		importID,
		openSearchServerlessResourceName("lifecycle-policy", string(policy.Type), StringValue(policy.Name)),
		openSearchServerlessLifecyclePolicyResourceType,
		"aws",
		attributes,
		openSearchServerlessAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newOpenSearchServerlessVPCEndpointResource(endpoint opensearchserverlesstypes.VpcEndpointDetail, securityGroupIDs []string) (terraformutils.Resource, bool) {
	importID := openSearchServerlessVPCEndpointImportID(endpoint)
	if importID == "" || !openSearchServerlessVPCEndpointImportable(endpoint, securityGroupIDs) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"name":   StringValue(endpoint.Name),
		"vpc_id": StringValue(endpoint.VpcId),
	}
	for key, value := range openSearchServerlessStringSliceAttributes("subnet_ids", endpoint.SubnetIds) {
		attributes[key] = value
	}
	for key, value := range openSearchServerlessStringSliceAttributes("security_group_ids", securityGroupIDs) {
		attributes[key] = value
	}
	return terraformutils.NewResource(
		importID,
		openSearchServerlessResourceName("vpc-endpoint", StringValue(endpoint.Name), importID),
		openSearchServerlessVPCEndpointResourceType,
		"aws",
		attributes,
		openSearchServerlessAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func openSearchServerlessCollectionImportID(collection opensearchserverlesstypes.CollectionDetail) string {
	return StringValue(collection.Id)
}

func openSearchServerlessAccessPolicyImportID(policy opensearchserverlesstypes.AccessPolicyDetail) string {
	return openSearchServerlessNameTypeImportID(StringValue(policy.Name), string(policy.Type))
}

func openSearchServerlessSecurityPolicyImportID(policy opensearchserverlesstypes.SecurityPolicyDetail) string {
	return openSearchServerlessNameTypeImportID(StringValue(policy.Name), string(policy.Type))
}

func openSearchServerlessSecurityConfigImportID(config opensearchserverlesstypes.SecurityConfigDetail) string {
	return StringValue(config.Id)
}

func openSearchServerlessLifecyclePolicyImportID(policy opensearchserverlesstypes.LifecyclePolicyDetail) string {
	return openSearchServerlessNameTypeImportID(StringValue(policy.Name), string(policy.Type))
}

func openSearchServerlessVPCEndpointImportID(endpoint opensearchserverlesstypes.VpcEndpointDetail) string {
	return StringValue(endpoint.Id)
}

func openSearchServerlessNameTypeImportID(name, resourceType string) string {
	if name == "" || resourceType == "" {
		return ""
	}
	return name + "/" + resourceType
}

func openSearchServerlessCollectionImportable(collection opensearchserverlesstypes.CollectionDetail) bool {
	return openSearchServerlessCollectionImportID(collection) != "" &&
		StringValue(collection.Name) != "" &&
		openSearchServerlessCollectionStatusImportable(collection.Status)
}

func openSearchServerlessCollectionStatusImportable(status opensearchserverlesstypes.CollectionStatus) bool {
	switch status {
	case opensearchserverlesstypes.CollectionStatusActive,
		opensearchserverlesstypes.CollectionStatusUpdating:
		return true
	default:
		return false
	}
}

func openSearchServerlessSecurityConfigImportable(config opensearchserverlesstypes.SecurityConfigDetail) bool {
	return openSearchServerlessSecurityConfigImportID(config) != "" &&
		config.Type == opensearchserverlesstypes.SecurityConfigTypeSaml &&
		config.SamlOptions != nil &&
		StringValue(config.SamlOptions.Metadata) != ""
}

func openSearchServerlessVPCEndpointImportable(endpoint opensearchserverlesstypes.VpcEndpointDetail, securityGroupIDs []string) bool {
	return openSearchServerlessVPCEndpointImportID(endpoint) != "" &&
		StringValue(endpoint.Name) != "" &&
		StringValue(endpoint.VpcId) != "" &&
		len(endpoint.SubnetIds) > 0 &&
		len(securityGroupIDs) > 0 &&
		endpoint.Status == opensearchserverlesstypes.VpcEndpointStatusActive
}

func openSearchServerlessPolicyAttributes(name, resourceType, policy, description string) map[string]string {
	attributes := map[string]string{
		"name":   name,
		"type":   resourceType,
		"policy": policy,
	}
	if description != "" {
		attributes["description"] = description
	}
	return attributes
}

func openSearchServerlessSecurityConfigName(importID string) string {
	parts := strings.Split(importID, "/")
	if len(parts) != 3 {
		return ""
	}
	return parts[2]
}

func openSearchServerlessSamlOptionsAttributes(prefix string, options *opensearchserverlesstypes.SamlConfigOptions) map[string]string {
	if options == nil {
		return map[string]string{
			prefix + ".#": "0",
		}
	}
	attributes := map[string]string{
		prefix + ".#": "1",
	}
	attributes[prefix+".0.metadata"] = StringValue(options.Metadata)
	if value := StringValue(options.GroupAttribute); value != "" {
		attributes[prefix+".0.group_attribute"] = value
	}
	if options.SessionTimeout != nil {
		attributes[prefix+".0.session_timeout"] = strconv.Itoa(int(*options.SessionTimeout))
	}
	if value := StringValue(options.UserAttribute); value != "" {
		attributes[prefix+".0.user_attribute"] = value
	}
	return attributes
}

func openSearchServerlessPolicyDocumentString(policy interface{}) string {
	if policy == nil {
		return ""
	}
	if value, ok := policy.(string); ok {
		return value
	}
	if value, ok := policy.(interface{ MarshalSmithyDocument() ([]byte, error) }); ok {
		b, err := value.MarshalSmithyDocument()
		if err == nil {
			return string(b)
		}
	}
	if value, ok := policy.(interface{ UnmarshalSmithyDocument(interface{}) error }); ok {
		var document interface{}
		if err := value.UnmarshalSmithyDocument(&document); err != nil {
			return ""
		}
		policy = document
	}
	b, err := json.Marshal(policy)
	if err != nil {
		return ""
	}
	return string(b)
}

func openSearchServerlessCollectionIDs(summaries []opensearchserverlesstypes.CollectionSummary) []string {
	var ids []string
	for _, summary := range summaries {
		if id := StringValue(summary.Id); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func openSearchServerlessVPCEndpointIDs(summaries []opensearchserverlesstypes.VpcEndpointSummary) []string {
	var ids []string
	for _, summary := range summaries {
		if id := StringValue(summary.Id); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func openSearchServerlessEC2VPCEndpointSecurityGroups(ctx context.Context, svc *ec2.Client, ids []string) (map[string][]string, error) {
	securityGroupIDs := map[string][]string{}
	if svc == nil || len(ids) == 0 {
		return securityGroupIDs, nil
	}
	output, err := svc.DescribeVpcEndpoints(ctx, &ec2.DescribeVpcEndpointsInput{
		VpcEndpointIds: ids,
	})
	if err != nil {
		return securityGroupIDs, err
	}
	for _, endpoint := range output.VpcEndpoints {
		id := StringValue(endpoint.VpcEndpointId)
		if id == "" {
			continue
		}
		securityGroupIDs[id] = openSearchServerlessEC2SecurityGroupIDs(endpoint.Groups)
	}
	return securityGroupIDs, nil
}

func openSearchServerlessEC2SecurityGroupIDs(groups []ec2types.SecurityGroupIdentifier) []string {
	var ids []string
	for _, group := range groups {
		if id := StringValue(group.GroupId); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func openSearchServerlessEC2ErrorSkippable(err error) bool {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	code := strings.ToLower(apiErr.ErrorCode())
	return strings.Contains(code, "accessdenied") ||
		strings.Contains(code, "unauthorized") ||
		strings.Contains(code, "notfound")
}

func openSearchServerlessStringChunks(values []string, size int) [][]string {
	if size <= 0 {
		return nil
	}
	var chunks [][]string
	for len(values) > 0 {
		if len(values) < size {
			chunks = append(chunks, values)
			break
		}
		chunks = append(chunks, values[:size])
		values = values[size:]
	}
	return chunks
}

func openSearchServerlessStringSliceAttributes(prefix string, values []string) map[string]string {
	attributes := map[string]string{
		prefix + ".#": strconv.Itoa(len(values)),
	}
	for i, value := range values {
		attributes[prefix+"."+strconv.Itoa(i)] = value
	}
	return attributes
}

func openSearchServerlessResourceName(parts ...string) string {
	var cleanParts []string
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}
	if len(cleanParts) == 0 {
		return openSearchServerlessResourceNameFallback
	}
	return strings.Join(cleanParts, "/")
}

func wrapOpenSearchServerlessPolicyHeredoc(g *OpenSearchServerlessGenerator, resource *terraformutils.Resource) {
	if resource.Item == nil {
		return
	}
	policy, ok := resource.Item["policy"].(string)
	if !ok || policy == "" {
		return
	}
	resource.Item["policy"] = fmt.Sprintf("<<POLICY\n%s\nPOLICY", g.escapeAwsInterpolation(policy))
}
