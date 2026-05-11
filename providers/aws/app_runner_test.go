// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	apprunnertypes "github.com/aws/aws-sdk-go-v2/service/apprunner/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestAppRunnerImportIDs(t *testing.T) {
	serviceARN := "arn:aws:apprunner:us-east-1:123456789012:service/example/8fe1e10304f84fd2b0df550fe98a71fa"
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "arn", got: appRunnerARNImportID(serviceARN), want: serviceARN},
		{name: "connection", got: appRunnerConnectionImportID("github"), want: "github"},
		{name: "custom domain", got: appRunnerCustomDomainAssociationImportID("example.com", serviceARN), want: "example.com," + serviceARN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("import ID = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestAppRunnerResourceNameAvoidsSanitizedCollisions(t *testing.T) {
	tests := []struct {
		name   string
		first  []string
		second []string
	}{
		{name: "separator boundary", first: []string{"custom_domain_association", "a_b", "c"}, second: []string{"custom_domain_association", "a", "b_c"}},
		{name: "slash encoding", first: []string{"vpc_connector", "a/b"}, second: []string{"vpc_connector", "a-002F-b"}},
		{name: "at sign encoding", first: []string{"connection", "a@example.com"}, second: []string{"connection", "a-0040-example.com"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := terraformutils.TfSanitize(appRunnerResourceName(tt.first...))
			second := terraformutils.TfSanitize(appRunnerResourceName(tt.second...))
			if first == second {
				t.Fatalf("appRunnerResourceName() generated duplicate sanitized names %q", first)
			}
		})
	}
}

func TestNewAppRunnerAutoScalingConfigurationVersionResource(t *testing.T) {
	arn := "arn:aws:apprunner:us-east-1:123456789012:autoscalingconfiguration/core/2/a1b2c3d4567890ab"
	resource, ok := newAppRunnerAutoScalingConfigurationVersionResource(apprunnertypes.AutoScalingConfigurationSummary{
		AutoScalingConfigurationArn:      aws.String(arn),
		AutoScalingConfigurationName:     aws.String("core"),
		AutoScalingConfigurationRevision: 2,
		Status:                           apprunnertypes.AutoScalingConfigurationStatusActive,
	})
	assertAppRunnerResource(t, resource, ok, arn, appRunnerResourceName("auto_scaling_configuration_version", "core", "2", "a1b2c3d4567890ab"), appRunnerAutoScalingConfigurationVersionResourceType, map[string]string{
		"arn":                             arn,
		"auto_scaling_configuration_name": "core",
	})

	if _, ok := newAppRunnerAutoScalingConfigurationVersionResource(apprunnertypes.AutoScalingConfigurationSummary{
		AutoScalingConfigurationArn:  aws.String(arn),
		AutoScalingConfigurationName: aws.String("core"),
		Status:                       apprunnertypes.AutoScalingConfigurationStatusInactive,
	}); ok {
		t.Fatal("inactive auto scaling configuration should be skipped")
	}
	if _, ok := newAppRunnerAutoScalingConfigurationVersionResource(apprunnertypes.AutoScalingConfigurationSummary{
		AutoScalingConfigurationName: aws.String("core"),
	}); ok {
		t.Fatal("auto scaling configuration with empty ARN should be skipped")
	}
}

func TestNewAppRunnerConnectionResource(t *testing.T) {
	resource, ok := newAppRunnerConnectionResource(apprunnertypes.ConnectionSummary{
		ConnectionArn:  aws.String("arn:aws:apprunner:us-east-1:123456789012:connection/github/a1b2c3d4567890ab"),
		ConnectionName: aws.String("github"),
		ProviderType:   apprunnertypes.ProviderTypeGithub,
		Status:         apprunnertypes.ConnectionStatusAvailable,
	})
	assertAppRunnerResource(t, resource, ok, "github", appRunnerResourceName("connection", "github"), appRunnerConnectionResourceType, map[string]string{
		"connection_name": "github",
		"provider_type":   "GITHUB",
	})

	if _, ok := newAppRunnerConnectionResource(apprunnertypes.ConnectionSummary{
		ConnectionName: aws.String("github"),
		Status:         apprunnertypes.ConnectionStatusDeleted,
	}); ok {
		t.Fatal("deleted connection should be skipped")
	}
	if _, ok := newAppRunnerConnectionResource(apprunnertypes.ConnectionSummary{
		ConnectionName: aws.String("github"),
		Status:         apprunnertypes.ConnectionStatusAvailable,
	}); ok {
		t.Fatal("connection with empty provider type should be skipped")
	}
	if _, ok := newAppRunnerConnectionResource(apprunnertypes.ConnectionSummary{}); ok {
		t.Fatal("connection with empty name should be skipped")
	}
}

func TestNewAppRunnerObservabilityConfigurationResource(t *testing.T) {
	arn := "arn:aws:apprunner:us-east-1:123456789012:observabilityconfiguration/core/1/d75bc7ea55b71e724fe5c23452fe22a1"
	resource, ok := newAppRunnerObservabilityConfigurationResource(apprunnertypes.ObservabilityConfigurationSummary{
		ObservabilityConfigurationArn:      aws.String(arn),
		ObservabilityConfigurationName:     aws.String("core"),
		ObservabilityConfigurationRevision: 1,
	})
	assertAppRunnerResource(t, resource, ok, arn, appRunnerResourceName("observability_configuration", "core", "1", "d75bc7ea55b71e724fe5c23452fe22a1"), appRunnerObservabilityConfigurationResourceType, map[string]string{
		"arn":                              arn,
		"observability_configuration_name": "core",
	})

	if _, ok := newAppRunnerObservabilityConfigurationResource(apprunnertypes.ObservabilityConfigurationSummary{
		ObservabilityConfigurationName: aws.String("core"),
	}); ok {
		t.Fatal("observability configuration with empty ARN should be skipped")
	}
}

func TestNewAppRunnerVpcConnectorResource(t *testing.T) {
	arn := "arn:aws:apprunner:us-east-1:123456789012:vpcconnector/private/1/0a03292a89764e5882c41d8f991c82fe"
	resource, ok := newAppRunnerVpcConnectorResource(apprunnertypes.VpcConnector{
		VpcConnectorArn:      aws.String(arn),
		VpcConnectorName:     aws.String("private"),
		VpcConnectorRevision: 1,
		Status:               apprunnertypes.VpcConnectorStatusActive,
	})
	assertAppRunnerResource(t, resource, ok, arn, appRunnerResourceName("vpc_connector", "private", "1", "0a03292a89764e5882c41d8f991c82fe"), appRunnerVpcConnectorResourceType, map[string]string{
		"arn":                arn,
		"vpc_connector_name": "private",
	})

	if _, ok := newAppRunnerVpcConnectorResource(apprunnertypes.VpcConnector{
		VpcConnectorArn:  aws.String(arn),
		VpcConnectorName: aws.String("private"),
		Status:           apprunnertypes.VpcConnectorStatusInactive,
	}); ok {
		t.Fatal("inactive VPC connector should be skipped")
	}
	if _, ok := newAppRunnerVpcConnectorResource(apprunnertypes.VpcConnector{}); ok {
		t.Fatal("VPC connector with empty identifiers should be skipped")
	}
}

func TestNewAppRunnerServiceResource(t *testing.T) {
	arn := "arn:aws:apprunner:us-east-1:123456789012:service/core/8fe1e10304f84fd2b0df550fe98a71fa"
	for _, status := range []apprunnertypes.ServiceStatus{"", apprunnertypes.ServiceStatusRunning, apprunnertypes.ServiceStatusPaused} {
		resource, ok := newAppRunnerServiceResource(apprunnertypes.ServiceSummary{
			ServiceArn:  aws.String(arn),
			ServiceName: aws.String("core"),
			Status:      status,
		})
		assertAppRunnerResource(t, resource, ok, arn, appRunnerResourceName("service", "core", "8fe1e10304f84fd2b0df550fe98a71fa"), appRunnerServiceResourceType, map[string]string{
			"arn":          arn,
			"service_name": "core",
		})
	}

	for _, status := range []apprunnertypes.ServiceStatus{apprunnertypes.ServiceStatusCreateFailed, apprunnertypes.ServiceStatusDeleted, apprunnertypes.ServiceStatusDeleteFailed, apprunnertypes.ServiceStatusOperationInProgress} {
		if _, ok := newAppRunnerServiceResource(apprunnertypes.ServiceSummary{
			ServiceArn:  aws.String(arn),
			ServiceName: aws.String("core"),
			Status:      status,
		}); ok {
			t.Fatalf("service with status %q should be skipped", status)
		}
	}
	if _, ok := newAppRunnerServiceResource(apprunnertypes.ServiceSummary{}); ok {
		t.Fatal("service with empty identifiers should be skipped")
	}
}

func TestNewAppRunnerCustomDomainAssociationResource(t *testing.T) {
	service := appRunnerServiceReference{
		arn:  "arn:aws:apprunner:us-east-1:123456789012:service/core/8fe1e10304f84fd2b0df550fe98a71fa",
		name: "core",
	}
	for _, status := range []apprunnertypes.CustomDomainAssociationStatus{
		"",
		apprunnertypes.CustomDomainAssociationStatusActive,
		apprunnertypes.CustomDomainAssociationStatusPendingCertificateDnsValidation,
		apprunnertypes.CustomDomainAssociationStatusBindingCertificate,
	} {
		resource, ok := newAppRunnerCustomDomainAssociationResource(service, apprunnertypes.CustomDomain{
			DomainName: aws.String("example.com"),
			Status:     status,
		})
		assertAppRunnerResource(t, resource, ok, appRunnerCustomDomainAssociationImportID("example.com", service.arn), appRunnerResourceName("custom_domain_association", "core", "example.com"), appRunnerCustomDomainAssociationResourceType, map[string]string{
			"domain_name": "example.com",
			"service_arn": service.arn,
		})
	}

	if _, ok := newAppRunnerCustomDomainAssociationResource(service, apprunnertypes.CustomDomain{
		DomainName: aws.String("example.com"),
		Status:     apprunnertypes.CustomDomainAssociationStatusDeleting,
	}); ok {
		t.Fatal("deleting custom domain association should be skipped")
	}
	if _, ok := newAppRunnerCustomDomainAssociationResource(appRunnerServiceReference{}, apprunnertypes.CustomDomain{
		DomainName: aws.String("example.com"),
	}); ok {
		t.Fatal("custom domain association with empty service ARN should be skipped")
	}
}

func TestNewAppRunnerVpcIngressConnectionResource(t *testing.T) {
	arn := "arn:aws:apprunner:us-east-1:123456789012:vpcingressconnection/private/a1b2c3d4567890ab"
	serviceARN := "arn:aws:apprunner:us-east-1:123456789012:service/core/8fe1e10304f84fd2b0df550fe98a71fa"
	resource, ok := newAppRunnerVpcIngressConnectionResource(apprunnertypes.VpcIngressConnection{
		ServiceArn:               aws.String(serviceARN),
		Status:                   apprunnertypes.VpcIngressConnectionStatusAvailable,
		VpcIngressConnectionArn:  aws.String(arn),
		VpcIngressConnectionName: aws.String("private"),
		IngressVpcConfiguration:  &apprunnertypes.IngressVpcConfiguration{VpcId: aws.String("vpc-123"), VpcEndpointId: aws.String("vpce-123")},
	})
	assertAppRunnerResource(t, resource, ok, arn, appRunnerARNResourceName("vpc_ingress_connection", arn), appRunnerVpcIngressConnectionResourceType, map[string]string{
		"arn":         arn,
		"service_arn": serviceARN,
	})

	if _, ok := newAppRunnerVpcIngressConnectionResource(apprunnertypes.VpcIngressConnection{
		Status:                  apprunnertypes.VpcIngressConnectionStatusDeleted,
		VpcIngressConnectionArn: aws.String(arn),
	}); ok {
		t.Fatal("deleted VPC ingress connection should be skipped")
	}
	if _, ok := newAppRunnerVpcIngressConnectionResource(apprunnertypes.VpcIngressConnection{}); ok {
		t.Fatal("VPC ingress connection with empty ARN should be skipped")
	}
}

func TestAppRunnerNotFound(t *testing.T) {
	if !appRunnerNotFound(&apprunnertypes.ResourceNotFoundException{}) {
		t.Fatal("appRunnerNotFound() = false for ResourceNotFoundException, want true")
	}
	if appRunnerNotFound(errors.New("boom")) {
		t.Fatal("appRunnerNotFound() = true for generic error, want false")
	}
	if appRunnerNotFound(nil) {
		t.Fatal("appRunnerNotFound() = true for nil error, want false")
	}
}

func assertAppRunnerResource(t *testing.T, resource terraformutils.Resource, ok bool, resourceID, resourceName, resourceType string, attributes map[string]string) {
	t.Helper()
	if !ok {
		t.Fatal("resource should be importable")
	}
	if resource.InstanceState.ID != resourceID {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, resourceID)
	}
	if resource.ResourceName != terraformutils.TfSanitize(resourceName) {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, terraformutils.TfSanitize(resourceName))
	}
	if resource.InstanceInfo.Type != resourceType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, resourceType)
	}
	for name, want := range attributes {
		if got := resource.InstanceState.Attributes[name]; got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
}
