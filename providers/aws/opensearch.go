// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	opensearchtypes "github.com/aws/aws-sdk-go-v2/service/opensearch/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	openSearchDomainResourceType                    = "aws_opensearch_domain"
	openSearchDomainPolicyResourceType              = "aws_opensearch_domain_policy"
	openSearchDomainSAMLOptionsResourceType         = "aws_opensearch_domain_saml_options"
	openSearchVPCEndpointResourceType               = "aws_opensearch_vpc_endpoint"
	openSearchPackageAssociationResourceType        = "aws_opensearch_package_association"
	openSearchOutboundConnectionResourceType        = "aws_opensearch_outbound_connection"
	openSearchInboundConnectionAccepterResourceType = "aws_opensearch_inbound_connection_accepter"

	openSearchDomainPolicyIDPrefix     = "esd-policy-"
	openSearchResourceNameFallback     = "opensearch-resource"
	openSearchVPCEndpointDescribeBatch = 100
)

var openSearchAllowEmptyValues = []string{
	"tags.",
	"^saml_options\\.\\d+\\.enabled$",
}

type OpenSearchGenerator struct {
	AWSService
}

type openSearchOptionalResourceLoader struct {
	name string
	load func() error
}

func (g *OpenSearchGenerator) loadOptionalResources(loaders []openSearchOptionalResourceLoader) error {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			if openSearchOptionalResourceErrorSkippable(err) {
				log.Printf("Skipping OpenSearch %s: %v", loader.name, err)
				continue
			}
			log.Printf("Failed OpenSearch %s discovery: %v", loader.name, err)
			return fmt.Errorf("loading OpenSearch %s: %w", loader.name, err)
		}
	}
	return nil
}

func openSearchOptionalResourceErrorSkippable(err error) bool {
	var notFound *opensearchtypes.ResourceNotFoundException
	if errors.As(err, &notFound) {
		return true
	}
	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && strings.Contains(strings.ToLower(apiErr.ErrorCode()), "accessdenied")
}

func (g *OpenSearchGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := opensearch.NewFromConfig(config)

	domains, err := g.loadDomains(svc)
	if err != nil {
		return err
	}
	return g.loadOptionalResources([]openSearchOptionalResourceLoader{
		{name: "VPC endpoints", load: func() error { return g.loadVPCEndpoints(svc) }},
		{name: "package associations", load: func() error { return g.loadPackageAssociations(svc, domains) }},
		{name: "outbound connections", load: func() error { return g.loadOutboundConnections(svc) }},
		{name: "inbound connection accepters", load: func() error { return g.loadInboundConnectionAccepters(svc) }},
	})
}

func (g *OpenSearchGenerator) PostConvertHook() error {
	for i := range g.Resources {
		if g.Resources[i].InstanceInfo == nil {
			continue
		}
		switch g.Resources[i].InstanceInfo.Type {
		case openSearchDomainResourceType:
			cleanOpenSearchDomainItem(&g.Resources[i])
		case openSearchDomainPolicyResourceType:
			wrapOpenSearchDomainPolicyHeredoc(g, &g.Resources[i])
		}
	}
	return nil
}

func (g *OpenSearchGenerator) loadDomains(svc *opensearch.Client) ([]opensearchtypes.DomainStatus, error) {
	output, err := svc.ListDomainNames(context.TODO(), &opensearch.ListDomainNamesInput{
		EngineType: opensearchtypes.EngineTypeOpenSearch,
	})
	if err != nil {
		return nil, err
	}
	var domains []opensearchtypes.DomainStatus
	for _, domainInfo := range output.DomainNames {
		name := StringValue(domainInfo.DomainName)
		if name == "" {
			continue
		}
		domainOutput, err := svc.DescribeDomain(context.TODO(), &opensearch.DescribeDomainInput{
			DomainName: &name,
		})
		if err != nil {
			if openSearchOptionalResourceErrorSkippable(err) {
				continue
			}
			return nil, err
		}
		if domainOutput.DomainStatus == nil {
			continue
		}
		domain := *domainOutput.DomainStatus
		if resource, ok := newOpenSearchDomainResource(domain); ok {
			g.Resources = append(g.Resources, resource)
			domains = append(domains, domain)
			if resource, ok := newOpenSearchDomainPolicyResource(domain); ok {
				g.Resources = append(g.Resources, resource)
			}
			if resource, ok := newOpenSearchDomainSAMLOptionsResource(domain); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return domains, nil
}

func (g *OpenSearchGenerator) loadVPCEndpoints(svc *opensearch.Client) error {
	var ids []string
	input := &opensearch.ListVpcEndpointsInput{}
	for {
		output, err := svc.ListVpcEndpoints(context.TODO(), input)
		if err != nil {
			return err
		}
		for _, summary := range output.VpcEndpointSummaryList {
			if id := StringValue(summary.VpcEndpointId); id != "" {
				ids = append(ids, id)
			}
		}
		if output.NextToken == nil || StringValue(output.NextToken) == "" {
			break
		}
		input.NextToken = output.NextToken
	}
	for _, chunk := range openSearchStringChunks(ids, openSearchVPCEndpointDescribeBatch) {
		output, err := svc.DescribeVpcEndpoints(context.TODO(), &opensearch.DescribeVpcEndpointsInput{
			VpcEndpointIds: chunk,
		})
		if err != nil {
			if openSearchOptionalResourceErrorSkippable(err) {
				continue
			}
			return err
		}
		for _, endpoint := range output.VpcEndpoints {
			if resource, ok := newOpenSearchVPCEndpointResource(endpoint); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *OpenSearchGenerator) loadPackageAssociations(svc *opensearch.Client, domains []opensearchtypes.DomainStatus) error {
	for _, domain := range domains {
		domainName := openSearchDomainImportID(domain)
		p := opensearch.NewListPackagesForDomainPaginator(svc, &opensearch.ListPackagesForDomainInput{
			DomainName: &domainName,
		})
		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if err != nil {
				if openSearchOptionalResourceErrorSkippable(err) {
					break
				}
				return err
			}
			for _, association := range page.DomainPackageDetailsList {
				if resource, ok := newOpenSearchPackageAssociationResource(association); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func (g *OpenSearchGenerator) loadOutboundConnections(svc *opensearch.Client) error {
	p := opensearch.NewDescribeOutboundConnectionsPaginator(svc, &opensearch.DescribeOutboundConnectionsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, connection := range page.Connections {
			if resource, ok := newOpenSearchOutboundConnectionResource(connection); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *OpenSearchGenerator) loadInboundConnectionAccepters(svc *opensearch.Client) error {
	p := opensearch.NewDescribeInboundConnectionsPaginator(svc, &opensearch.DescribeInboundConnectionsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, connection := range page.Connections {
			if resource, ok := newOpenSearchInboundConnectionAccepterResource(connection); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newOpenSearchDomainResource(domain opensearchtypes.DomainStatus) (terraformutils.Resource, bool) {
	importID := openSearchDomainImportID(domain)
	if importID == "" || !openSearchDomainImportable(domain) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		importID,
		openSearchResourceName("domain", importID),
		openSearchDomainResourceType,
		"aws",
		map[string]string{"domain_name": importID},
		openSearchAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newOpenSearchDomainPolicyResource(domain opensearchtypes.DomainStatus) (terraformutils.Resource, bool) {
	importID := openSearchDomainPolicyImportID(domain)
	policy := StringValue(domain.AccessPolicies)
	if importID == "" || policy == "" {
		return terraformutils.Resource{}, false
	}
	domainName := openSearchDomainImportID(domain)
	return terraformutils.NewResource(
		importID,
		openSearchResourceName("domain-policy", domainName),
		openSearchDomainPolicyResourceType,
		"aws",
		map[string]string{
			"access_policies": policy,
			"domain_name":     domainName,
		},
		openSearchAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newOpenSearchDomainSAMLOptionsResource(domain opensearchtypes.DomainStatus) (terraformutils.Resource, bool) {
	importID := openSearchDomainSAMLImportID(domain)
	if importID == "" || !openSearchDomainSAMLImportable(domain) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"domain_name": importID,
	}
	for key, value := range openSearchSAMLAttributes("saml_options", domain.AdvancedSecurityOptions.SAMLOptions) {
		attributes[key] = value
	}
	return terraformutils.NewResource(
		importID,
		openSearchResourceName("domain-saml-options", importID),
		openSearchDomainSAMLOptionsResourceType,
		"aws",
		attributes,
		openSearchAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newOpenSearchVPCEndpointResource(endpoint opensearchtypes.VpcEndpoint) (terraformutils.Resource, bool) {
	importID := openSearchVPCEndpointImportID(endpoint)
	if importID == "" || !openSearchVPCEndpointImportable(endpoint) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"domain_arn":    StringValue(endpoint.DomainArn),
		"vpc_options.#": "1",
	}
	if endpoint.VpcOptions != nil {
		for key, value := range openSearchStringSliceAttributes("vpc_options.0.subnet_ids", endpoint.VpcOptions.SubnetIds) {
			attributes[key] = value
		}
		for key, value := range openSearchStringSliceAttributes("vpc_options.0.security_group_ids", endpoint.VpcOptions.SecurityGroupIds) {
			attributes[key] = value
		}
	}
	return terraformutils.NewResource(
		importID,
		openSearchResourceName("vpc-endpoint", importID),
		openSearchVPCEndpointResourceType,
		"aws",
		attributes,
		openSearchAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newOpenSearchPackageAssociationResource(association opensearchtypes.DomainPackageDetails) (terraformutils.Resource, bool) {
	importID := openSearchPackageAssociationImportID(association)
	if importID == "" || !openSearchPackageAssociationImportable(association) {
		return terraformutils.Resource{}, false
	}
	domainName := StringValue(association.DomainName)
	packageID := StringValue(association.PackageID)
	return terraformutils.NewResource(
		importID,
		openSearchResourceName("package-association", domainName, packageID),
		openSearchPackageAssociationResourceType,
		"aws",
		map[string]string{
			"domain_name": domainName,
			"package_id":  packageID,
		},
		openSearchAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newOpenSearchOutboundConnectionResource(connection opensearchtypes.OutboundConnection) (terraformutils.Resource, bool) {
	importID := openSearchOutboundConnectionImportID(connection)
	if importID == "" || !openSearchOutboundConnectionImportable(connection) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"connection_alias": StringValue(connection.ConnectionAlias),
	}
	if connection.ConnectionMode != "" {
		attributes["connection_mode"] = string(connection.ConnectionMode)
	}
	for key, value := range openSearchConnectionDomainInfoAttributes("local_domain_info", connection.LocalDomainInfo) {
		attributes[key] = value
	}
	for key, value := range openSearchConnectionDomainInfoAttributes("remote_domain_info", connection.RemoteDomainInfo) {
		attributes[key] = value
	}
	for key, value := range openSearchConnectionPropertiesAttributes("connection_properties", connection.ConnectionProperties) {
		attributes[key] = value
	}
	return terraformutils.NewResource(
		importID,
		openSearchResourceName("outbound-connection", StringValue(connection.ConnectionAlias), importID),
		openSearchOutboundConnectionResourceType,
		"aws",
		attributes,
		openSearchAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newOpenSearchInboundConnectionAccepterResource(connection opensearchtypes.InboundConnection) (terraformutils.Resource, bool) {
	importID := openSearchInboundConnectionAccepterImportID(connection)
	if importID == "" || !openSearchInboundConnectionAccepterImportable(connection) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		importID,
		openSearchResourceName("inbound-connection-accepter", importID),
		openSearchInboundConnectionAccepterResourceType,
		"aws",
		map[string]string{
			"connection_id": importID,
		},
		openSearchAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func openSearchDomainImportID(domain opensearchtypes.DomainStatus) string {
	return StringValue(domain.DomainName)
}

func openSearchDomainPolicyImportID(domain opensearchtypes.DomainStatus) string {
	domainName := openSearchDomainImportID(domain)
	if domainName == "" {
		return ""
	}
	return openSearchDomainPolicyIDPrefix + domainName
}

func openSearchDomainSAMLImportID(domain opensearchtypes.DomainStatus) string {
	return openSearchDomainImportID(domain)
}

func openSearchVPCEndpointImportID(endpoint opensearchtypes.VpcEndpoint) string {
	return StringValue(endpoint.VpcEndpointId)
}

func openSearchPackageAssociationImportID(association opensearchtypes.DomainPackageDetails) string {
	domainName := StringValue(association.DomainName)
	packageID := StringValue(association.PackageID)
	if domainName == "" || packageID == "" {
		return ""
	}
	return domainName + "-" + packageID
}

func openSearchOutboundConnectionImportID(connection opensearchtypes.OutboundConnection) string {
	return StringValue(connection.ConnectionId)
}

func openSearchInboundConnectionAccepterImportID(connection opensearchtypes.InboundConnection) string {
	return StringValue(connection.ConnectionId)
}

func openSearchDomainImportable(domain opensearchtypes.DomainStatus) bool {
	if openSearchDomainImportID(domain) == "" {
		return false
	}
	if domain.Created != nil && !*domain.Created {
		return false
	}
	if domain.Deleted != nil && *domain.Deleted {
		return false
	}
	switch domain.DomainProcessingStatus {
	case opensearchtypes.DomainProcessingStatusTypeCreating,
		opensearchtypes.DomainProcessingStatusTypeDeleting:
		return false
	default:
		return true
	}
}

func openSearchDomainSAMLImportable(domain opensearchtypes.DomainStatus) bool {
	if domain.AdvancedSecurityOptions == nil || domain.AdvancedSecurityOptions.SAMLOptions == nil {
		return false
	}
	options := domain.AdvancedSecurityOptions.SAMLOptions
	if options.Enabled != nil && !*options.Enabled {
		return false
	}
	return options.Idp != nil &&
		StringValue(options.Idp.EntityId) != "" &&
		StringValue(options.Idp.MetadataContent) != ""
}

func openSearchVPCEndpointImportable(endpoint opensearchtypes.VpcEndpoint) bool {
	return openSearchVPCEndpointImportID(endpoint) != "" &&
		StringValue(endpoint.DomainArn) != "" &&
		endpoint.VpcOptions != nil &&
		len(endpoint.VpcOptions.SubnetIds) > 0 &&
		openSearchVPCEndpointStatusImportable(endpoint.Status)
}

func openSearchVPCEndpointStatusImportable(status opensearchtypes.VpcEndpointStatus) bool {
	switch status {
	case opensearchtypes.VpcEndpointStatusActive,
		opensearchtypes.VpcEndpointStatusUpdating:
		return true
	default:
		return false
	}
}

func openSearchPackageAssociationImportable(association opensearchtypes.DomainPackageDetails) bool {
	return openSearchPackageAssociationImportID(association) != "" &&
		association.DomainPackageStatus == opensearchtypes.DomainPackageStatusActive
}

func openSearchOutboundConnectionImportable(connection opensearchtypes.OutboundConnection) bool {
	if openSearchOutboundConnectionImportID(connection) == "" ||
		StringValue(connection.ConnectionAlias) == "" ||
		!openSearchConnectionDomainInfoImportable(connection.LocalDomainInfo) ||
		!openSearchConnectionDomainInfoImportable(connection.RemoteDomainInfo) ||
		connection.ConnectionStatus == nil {
		return false
	}
	switch connection.ConnectionStatus.StatusCode {
	case opensearchtypes.OutboundConnectionStatusCodeActive,
		opensearchtypes.OutboundConnectionStatusCodeApproved,
		opensearchtypes.OutboundConnectionStatusCodePendingAcceptance:
		return true
	default:
		return false
	}
}

func openSearchInboundConnectionAccepterImportable(connection opensearchtypes.InboundConnection) bool {
	return openSearchInboundConnectionAccepterImportID(connection) != "" &&
		connection.ConnectionStatus != nil &&
		connection.ConnectionStatus.StatusCode == opensearchtypes.InboundConnectionStatusCodeActive
}

func openSearchConnectionDomainInfoImportable(info *opensearchtypes.DomainInformationContainer) bool {
	if info == nil || info.AWSDomainInformation == nil {
		return false
	}
	domainInfo := info.AWSDomainInformation
	return StringValue(domainInfo.DomainName) != "" &&
		StringValue(domainInfo.OwnerId) != "" &&
		StringValue(domainInfo.Region) != ""
}

func openSearchSAMLAttributes(prefix string, options *opensearchtypes.SAMLOptionsOutput) map[string]string {
	attributes := map[string]string{
		prefix + ".#": "1",
	}
	if options == nil {
		return attributes
	}
	enabled := true
	if options.Enabled != nil {
		enabled = *options.Enabled
	}
	attributes[prefix+".0.enabled"] = strconv.FormatBool(enabled)
	if !enabled {
		return attributes
	}
	if options.Idp != nil {
		attributes[prefix+".0.idp.#"] = "1"
		attributes[prefix+".0.idp.0.entity_id"] = StringValue(options.Idp.EntityId)
		attributes[prefix+".0.idp.0.metadata_content"] = StringValue(options.Idp.MetadataContent)
	}
	if value := StringValue(options.RolesKey); value != "" {
		attributes[prefix+".0.roles_key"] = value
	}
	if options.SessionTimeoutMinutes != nil {
		attributes[prefix+".0.session_timeout_minutes"] = strconv.Itoa(int(*options.SessionTimeoutMinutes))
	}
	if value := StringValue(options.SubjectKey); value != "" {
		attributes[prefix+".0.subject_key"] = value
	}
	return attributes
}

func openSearchConnectionDomainInfoAttributes(prefix string, info *opensearchtypes.DomainInformationContainer) map[string]string {
	attributes := map[string]string{
		prefix + ".#": "1",
	}
	if info == nil || info.AWSDomainInformation == nil {
		return attributes
	}
	domainInfo := info.AWSDomainInformation
	attributes[prefix+".0.domain_name"] = StringValue(domainInfo.DomainName)
	attributes[prefix+".0.owner_id"] = StringValue(domainInfo.OwnerId)
	attributes[prefix+".0.region"] = StringValue(domainInfo.Region)
	return attributes
}

func openSearchConnectionPropertiesAttributes(prefix string, properties *opensearchtypes.ConnectionProperties) map[string]string {
	attributes := map[string]string{}
	if properties == nil || properties.CrossClusterSearch == nil || properties.CrossClusterSearch.SkipUnavailable == "" {
		return attributes
	}
	attributes[prefix+".#"] = "1"
	attributes[prefix+".0.cross_cluster_search.#"] = "1"
	attributes[prefix+".0.cross_cluster_search.0.skip_unavailable"] = string(properties.CrossClusterSearch.SkipUnavailable)
	return attributes
}

func openSearchStringSliceAttributes(prefix string, values []string) map[string]string {
	attributes := map[string]string{
		prefix + ".#": strconv.Itoa(len(values)),
	}
	for i, value := range values {
		attributes[prefix+"."+strconv.Itoa(i)] = value
	}
	return attributes
}

func openSearchStringChunks(values []string, size int) [][]string {
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

func openSearchResourceName(parts ...string) string {
	var cleanParts []string
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}
	if len(cleanParts) == 0 {
		return openSearchResourceNameFallback
	}
	return strings.Join(cleanParts, "/")
}

func cleanOpenSearchDomainItem(resource *terraformutils.Resource) {
	if resource.Item == nil {
		return
	}
	delete(resource.Item, "access_policies")
	if resource.InstanceState != nil {
		delete(resource.InstanceState.Attributes, "access_policies")
	}
	if resource.InstanceState.Attributes["cognito_options.0.enabled"] == "false" {
		delete(resource.Item, "cognito_options")
	}
	clusterConfig, ok := resource.Item["cluster_config"].([]interface{})
	if !ok || len(clusterConfig) == 0 {
		return
	}
	clusterConfigItem, ok := clusterConfig[0].(map[string]interface{})
	if !ok {
		return
	}
	if resource.InstanceState.Attributes["cluster_config.0.warm_count"] == "0" {
		delete(clusterConfigItem, "warm_count")
	}
}

func wrapOpenSearchDomainPolicyHeredoc(g *OpenSearchGenerator, resource *terraformutils.Resource) {
	if resource.Item == nil {
		return
	}
	policy, ok := resource.Item["access_policies"].(string)
	if !ok || policy == "" {
		return
	}
	resource.Item["access_policies"] = fmt.Sprintf("<<POLICY\n%s\nPOLICY", g.escapeAwsInterpolation(policy))
}
