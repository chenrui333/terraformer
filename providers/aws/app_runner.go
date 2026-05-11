// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/apprunner"
	apprunnertypes "github.com/aws/aws-sdk-go-v2/service/apprunner/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	appRunnerAutoScalingConfigurationVersionResourceType = "aws_apprunner_auto_scaling_configuration_version"
	appRunnerConnectionResourceType                      = "aws_apprunner_connection"
	appRunnerCustomDomainAssociationResourceType         = "aws_apprunner_custom_domain_association"
	appRunnerObservabilityConfigurationResourceType      = "aws_apprunner_observability_configuration"
	appRunnerServiceResourceType                         = "aws_apprunner_service"
	appRunnerVpcConnectorResourceType                    = "aws_apprunner_vpc_connector"
	appRunnerVpcIngressConnectionResourceType            = "aws_apprunner_vpc_ingress_connection"

	appRunnerCustomDomainAssociationIDSeparator = ","
)

var appRunnerAllowEmptyValues = []string{"tags."}

type AppRunnerGenerator struct {
	AWSService
}

type appRunnerServiceReference struct {
	arn  string
	name string
}

type appRunnerOptionalResourceLoader struct {
	name string
	load func() error
}

func (g *AppRunnerGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := apprunner.NewFromConfig(config)

	if err := g.loadAutoScalingConfigurations(svc); err != nil {
		return err
	}
	if err := g.loadConnections(svc); err != nil {
		return err
	}
	if err := g.loadObservabilityConfigurations(svc); err != nil {
		return err
	}
	if err := g.loadVpcConnectors(svc); err != nil {
		return err
	}
	services, err := g.loadServices(svc)
	if err != nil {
		return err
	}
	if err := g.loadVpcIngressConnections(svc); err != nil {
		return err
	}

	g.getOptionalAppRunnerResources(
		appRunnerOptionalResourceLoader{name: "custom domain associations", load: func() error {
			return g.loadCustomDomainAssociations(svc, services)
		}},
	)

	return nil
}

func (g *AppRunnerGenerator) loadAutoScalingConfigurations(svc *apprunner.Client) error {
	p := apprunner.NewListAutoScalingConfigurationsPaginator(svc, &apprunner.ListAutoScalingConfigurationsInput{
		LatestOnly: false,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, configuration := range page.AutoScalingConfigurationSummaryList {
			if resource, ok := newAppRunnerAutoScalingConfigurationVersionResource(configuration); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AppRunnerGenerator) loadConnections(svc *apprunner.Client) error {
	p := apprunner.NewListConnectionsPaginator(svc, &apprunner.ListConnectionsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, connection := range page.ConnectionSummaryList {
			if resource, ok := newAppRunnerConnectionResource(connection); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AppRunnerGenerator) loadObservabilityConfigurations(svc *apprunner.Client) error {
	p := apprunner.NewListObservabilityConfigurationsPaginator(svc, &apprunner.ListObservabilityConfigurationsInput{
		LatestOnly: false,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, configuration := range page.ObservabilityConfigurationSummaryList {
			arn := StringValue(configuration.ObservabilityConfigurationArn)
			if arn == "" {
				continue
			}
			output, err := svc.DescribeObservabilityConfiguration(context.TODO(), &apprunner.DescribeObservabilityConfigurationInput{
				ObservabilityConfigurationArn: &arn,
			})
			if err != nil {
				if appRunnerNotFound(err) {
					continue
				}
				return err
			}
			if output == nil || output.ObservabilityConfiguration == nil {
				continue
			}
			if resource, ok := newAppRunnerObservabilityConfigurationResource(*output.ObservabilityConfiguration); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AppRunnerGenerator) loadVpcConnectors(svc *apprunner.Client) error {
	p := apprunner.NewListVpcConnectorsPaginator(svc, &apprunner.ListVpcConnectorsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, connector := range page.VpcConnectors {
			if resource, ok := newAppRunnerVpcConnectorResource(connector); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AppRunnerGenerator) loadServices(svc *apprunner.Client) ([]appRunnerServiceReference, error) {
	services := []appRunnerServiceReference{}
	p := apprunner.NewListServicesPaginator(svc, &apprunner.ListServicesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		for _, service := range page.ServiceSummaryList {
			resource, ok := newAppRunnerServiceResource(service)
			if !ok {
				continue
			}
			g.Resources = append(g.Resources, resource)
			services = append(services, appRunnerServiceReference{
				arn:  StringValue(service.ServiceArn),
				name: StringValue(service.ServiceName),
			})
		}
	}
	return services, nil
}

func (g *AppRunnerGenerator) loadVpcIngressConnections(svc *apprunner.Client) error {
	p := apprunner.NewListVpcIngressConnectionsPaginator(svc, &apprunner.ListVpcIngressConnectionsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, connection := range page.VpcIngressConnectionSummaryList {
			arn := StringValue(connection.VpcIngressConnectionArn)
			if arn == "" {
				continue
			}
			output, err := svc.DescribeVpcIngressConnection(context.TODO(), &apprunner.DescribeVpcIngressConnectionInput{
				VpcIngressConnectionArn: &arn,
			})
			if err != nil {
				if appRunnerNotFound(err) {
					continue
				}
				return err
			}
			if output == nil || output.VpcIngressConnection == nil {
				continue
			}
			if resource, ok := newAppRunnerVpcIngressConnectionResource(*output.VpcIngressConnection); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AppRunnerGenerator) loadCustomDomainAssociations(svc *apprunner.Client, services []appRunnerServiceReference) error {
	for _, service := range services {
		if service.arn == "" {
			continue
		}
		p := apprunner.NewDescribeCustomDomainsPaginator(svc, &apprunner.DescribeCustomDomainsInput{
			ServiceArn: &service.arn,
		})
		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if err != nil {
				if appRunnerNotFound(err) {
					break
				}
				return err
			}
			for _, domain := range page.CustomDomains {
				if resource, ok := newAppRunnerCustomDomainAssociationResource(service, domain); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func (g *AppRunnerGenerator) getOptionalAppRunnerResources(loaders ...appRunnerOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			log.Printf("skipping App Runner %s discovery: %v", loader.name, err)
		}
	}
}

func newAppRunnerAutoScalingConfigurationVersionResource(configuration apprunnertypes.AutoScalingConfigurationSummary) (terraformutils.Resource, bool) {
	arn := StringValue(configuration.AutoScalingConfigurationArn)
	name := StringValue(configuration.AutoScalingConfigurationName)
	if arn == "" || name == "" || !appRunnerAutoScalingConfigurationImportable(configuration) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		appRunnerARNImportID(arn),
		appRunnerResourceName("auto_scaling_configuration_version", name, strconv.Itoa(int(configuration.AutoScalingConfigurationRevision)), arnLastSegment(arn, "/")),
		appRunnerAutoScalingConfigurationVersionResourceType,
		"aws",
		map[string]string{
			"arn":                             arn,
			"auto_scaling_configuration_name": name,
		},
		appRunnerAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newAppRunnerConnectionResource(connection apprunnertypes.ConnectionSummary) (terraformutils.Resource, bool) {
	name := StringValue(connection.ConnectionName)
	providerType := string(connection.ProviderType)
	if name == "" || providerType == "" || !appRunnerConnectionImportable(connection) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"connection_name": name,
		"provider_type":   providerType,
	}
	if arn := StringValue(connection.ConnectionArn); arn != "" {
		attributes["arn"] = arn
	}
	return terraformutils.NewResource(
		appRunnerConnectionImportID(name),
		appRunnerResourceName("connection", name),
		appRunnerConnectionResourceType,
		"aws",
		attributes,
		appRunnerAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newAppRunnerObservabilityConfigurationResource(configuration apprunnertypes.ObservabilityConfiguration) (terraformutils.Resource, bool) {
	arn := StringValue(configuration.ObservabilityConfigurationArn)
	name := StringValue(configuration.ObservabilityConfigurationName)
	if arn == "" || name == "" || !appRunnerObservabilityConfigurationImportable(configuration) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		appRunnerARNImportID(arn),
		appRunnerResourceName("observability_configuration", name, strconv.Itoa(int(configuration.ObservabilityConfigurationRevision)), arnLastSegment(arn, "/")),
		appRunnerObservabilityConfigurationResourceType,
		"aws",
		map[string]string{
			"arn":                              arn,
			"observability_configuration_name": name,
		},
		appRunnerAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newAppRunnerVpcConnectorResource(connector apprunnertypes.VpcConnector) (terraformutils.Resource, bool) {
	arn := StringValue(connector.VpcConnectorArn)
	name := StringValue(connector.VpcConnectorName)
	if arn == "" || name == "" || !appRunnerVpcConnectorImportable(connector) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		appRunnerARNImportID(arn),
		appRunnerResourceName("vpc_connector", name, strconv.Itoa(int(connector.VpcConnectorRevision)), arnLastSegment(arn, "/")),
		appRunnerVpcConnectorResourceType,
		"aws",
		map[string]string{
			"arn":                arn,
			"vpc_connector_name": name,
		},
		appRunnerAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newAppRunnerServiceResource(service apprunnertypes.ServiceSummary) (terraformutils.Resource, bool) {
	arn := StringValue(service.ServiceArn)
	name := StringValue(service.ServiceName)
	if arn == "" || name == "" || !appRunnerServiceImportable(service) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		appRunnerARNImportID(arn),
		appRunnerResourceName("service", name, arnLastSegment(arn, "/")),
		appRunnerServiceResourceType,
		"aws",
		map[string]string{
			"arn":          arn,
			"service_name": name,
		},
		appRunnerAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newAppRunnerCustomDomainAssociationResource(service appRunnerServiceReference, domain apprunnertypes.CustomDomain) (terraformutils.Resource, bool) {
	domainName := StringValue(domain.DomainName)
	if service.arn == "" || domainName == "" || !appRunnerCustomDomainImportable(domain) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		appRunnerCustomDomainAssociationImportID(domainName, service.arn),
		appRunnerResourceName("custom_domain_association", service.name, domainName),
		appRunnerCustomDomainAssociationResourceType,
		"aws",
		map[string]string{
			"domain_name": domainName,
			"service_arn": service.arn,
		},
		appRunnerAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newAppRunnerVpcIngressConnectionResource(connection apprunnertypes.VpcIngressConnection) (terraformutils.Resource, bool) {
	arn := StringValue(connection.VpcIngressConnectionArn)
	if arn == "" || !appRunnerVpcIngressConnectionImportable(connection) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"arn": arn,
	}
	if serviceARN := StringValue(connection.ServiceArn); serviceARN != "" {
		attributes["service_arn"] = serviceARN
	}
	return terraformutils.NewResource(
		appRunnerARNImportID(arn),
		appRunnerARNResourceName("vpc_ingress_connection", arn),
		appRunnerVpcIngressConnectionResourceType,
		"aws",
		attributes,
		appRunnerAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func appRunnerARNImportID(arn string) string {
	return arn
}

func appRunnerConnectionImportID(name string) string {
	return name
}

func appRunnerCustomDomainAssociationImportID(domainName, serviceARN string) string {
	return strings.Join([]string{domainName, serviceARN}, appRunnerCustomDomainAssociationIDSeparator)
}

func appRunnerARNResourceName(prefix, arn string) string {
	parts := []string{prefix}
	for _, part := range strings.Split(arnLastSegment(arn, ":"), "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return appRunnerResourceName(parts...)
}

func appRunnerResourceName(parts ...string) string {
	var name strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		if name.Len() > 0 {
			name.WriteString("_")
		}
		name.WriteString(strconv.Itoa(len(part)))
		name.WriteString("_")
		name.WriteString(part)
	}
	return name.String()
}

func appRunnerAutoScalingConfigurationImportable(configuration apprunnertypes.AutoScalingConfigurationSummary) bool {
	return configuration.Status == "" || configuration.Status == apprunnertypes.AutoScalingConfigurationStatusActive
}

func appRunnerConnectionImportable(connection apprunnertypes.ConnectionSummary) bool {
	return connection.Status != apprunnertypes.ConnectionStatusDeleted
}

func appRunnerObservabilityConfigurationImportable(configuration apprunnertypes.ObservabilityConfiguration) bool {
	return configuration.Status == "" || configuration.Status == apprunnertypes.ObservabilityConfigurationStatusActive
}

func appRunnerVpcConnectorImportable(connector apprunnertypes.VpcConnector) bool {
	return connector.Status == "" || connector.Status == apprunnertypes.VpcConnectorStatusActive
}

func appRunnerServiceImportable(service apprunnertypes.ServiceSummary) bool {
	switch service.Status {
	case "", apprunnertypes.ServiceStatusRunning, apprunnertypes.ServiceStatusPaused:
		return true
	default:
		return false
	}
}

func appRunnerCustomDomainImportable(domain apprunnertypes.CustomDomain) bool {
	switch domain.Status {
	case "", apprunnertypes.CustomDomainAssociationStatusActive, apprunnertypes.CustomDomainAssociationStatusPendingCertificateDnsValidation, apprunnertypes.CustomDomainAssociationStatusBindingCertificate:
		return true
	default:
		return false
	}
}

func appRunnerVpcIngressConnectionImportable(connection apprunnertypes.VpcIngressConnection) bool {
	return connection.Status == "" || connection.Status == apprunnertypes.VpcIngressConnectionStatusAvailable
}

func appRunnerNotFound(err error) bool {
	var notFound *apprunnertypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
