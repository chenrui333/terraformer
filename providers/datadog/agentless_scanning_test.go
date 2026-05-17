// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

func TestAgentlessScanningAwsCreateResource(t *testing.T) {
	data := datadogV2.NewAwsScanOptionsDataWithDefaults()
	data.SetId("123456789012")

	generator := &AgentlessScanningAwsScanOptionsGenerator{}
	resource, err := generator.createResource(*data)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "123456789012" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "123456789012")
	}
	if resource.InstanceInfo.Type != "datadog_agentless_scanning_aws_scan_options" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_agentless_scanning_aws_scan_options")
	}
	if v := resource.InstanceState.Attributes["aws_account_id"]; v != "123456789012" {
		t.Fatalf("aws_account_id = %q, want %q", v, "123456789012")
	}
}

func TestAgentlessScanningAwsCreateResourceMissingID(t *testing.T) {
	generator := &AgentlessScanningAwsScanOptionsGenerator{}
	_, err := generator.createResource(datadogV2.AwsScanOptionsData{})
	if err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestAgentlessScanningAwsCreateResources(t *testing.T) {
	first := datadogV2.NewAwsScanOptionsDataWithDefaults()
	first.SetId("111111111111")
	second := datadogV2.NewAwsScanOptionsDataWithDefaults()
	second.SetId("222222222222")

	generator := &AgentlessScanningAwsScanOptionsGenerator{}
	resources, err := generator.createResources([]datadogV2.AwsScanOptionsData{*first, *second})
	if err != nil {
		t.Fatalf("createResources returned error: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("resource count = %d, want %d", len(resources), 2)
	}
	if resources[0].InstanceState.ID == resources[1].InstanceState.ID {
		t.Fatal("resource IDs should be unique")
	}
}

func TestAgentlessScanningAwsCreateResourcesEmpty(t *testing.T) {
	generator := &AgentlessScanningAwsScanOptionsGenerator{}
	resources, err := generator.createResources(nil)
	if err != nil {
		t.Fatalf("createResources returned error: %v", err)
	}
	if len(resources) != 0 {
		t.Fatalf("resource count = %d, want %d", len(resources), 0)
	}
}

func TestAgentlessScanningAzureCreateResource(t *testing.T) {
	data := datadogV2.NewAzureScanOptionsData("sub-id-123", datadogV2.AZURESCANOPTIONSDATATYPE_AZURE_SCAN_OPTIONS)

	generator := &AgentlessScanningAzureScanOptionsGenerator{}
	resource, err := generator.createResource(*data)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "sub-id-123" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "sub-id-123")
	}
	if resource.InstanceInfo.Type != "datadog_agentless_scanning_azure_scan_options" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_agentless_scanning_azure_scan_options")
	}
	if v := resource.InstanceState.Attributes["azure_subscription_id"]; v != "sub-id-123" {
		t.Fatalf("azure_subscription_id = %q, want %q", v, "sub-id-123")
	}
}

func TestAgentlessScanningAzureCreateResourceMissingID(t *testing.T) {
	generator := &AgentlessScanningAzureScanOptionsGenerator{}
	_, err := generator.createResource(datadogV2.AzureScanOptionsData{})
	if err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestAgentlessScanningGcpCreateResource(t *testing.T) {
	data := datadogV2.NewGcpScanOptionsData("my-project-123", datadogV2.GCPSCANOPTIONSDATATYPE_GCP_SCAN_OPTIONS)

	generator := &AgentlessScanningGcpScanOptionsGenerator{}
	resource, err := generator.createResource(*data)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "my-project-123" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "my-project-123")
	}
	if resource.InstanceInfo.Type != "datadog_agentless_scanning_gcp_scan_options" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_agentless_scanning_gcp_scan_options")
	}
	if v := resource.InstanceState.Attributes["gcp_project_id"]; v != "my-project-123" {
		t.Fatalf("gcp_project_id = %q, want %q", v, "my-project-123")
	}
}

func TestAgentlessScanningGcpCreateResourceMissingID(t *testing.T) {
	generator := &AgentlessScanningGcpScanOptionsGenerator{}
	_, err := generator.createResource(datadogV2.GcpScanOptionsData{})
	if err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}
