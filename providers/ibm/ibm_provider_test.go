// SPDX-License-Identifier: Apache-2.0

package ibm

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestProviderInitDoesNotRequireAPIKey(t *testing.T) {
	t.Setenv("IC_API_KEY", "")

	provider := &IBMProvider{}
	if err := provider.Init([]string{"", "", ""}); err != nil {
		t.Fatalf("expected provider initialization to succeed without API key: %v", err)
	}
	if provider.Region != DefaultRegion {
		t.Fatalf("expected default region %q, got %q", DefaultRegion, provider.Region)
	}
	if err := provider.InitService("ibm_is_image", false); err != nil {
		t.Fatalf("expected plan replay service initialization to succeed without API key: %v", err)
	}
}

func TestProviderInitRequiresArgs(t *testing.T) {
	t.Setenv("IC_REGION", "old-region")
	provider := &IBMProvider{
		ResourceGroup: "old-resource-group",
		Region:        "old-region",
		VPC:           "old-vpc",
	}
	err := provider.Init([]string{"", ""})
	if err == nil {
		t.Fatal("expected missing args error")
	}
	if err.Error() != "ibm: expected 3 init args (resource group, region, vpc)" {
		t.Fatalf("Init error = %q, want missing IBM args", err)
	}
	if provider.ResourceGroup != "" {
		t.Fatalf("ResourceGroup = %q, want empty after failed init", provider.ResourceGroup)
	}
	if provider.Region != "" {
		t.Fatalf("Region = %q, want empty after failed init", provider.Region)
	}
	if provider.VPC != "" {
		t.Fatalf("VPC = %q, want empty after failed init", provider.VPC)
	}
	if value, ok := os.LookupEnv("IC_REGION"); ok {
		t.Fatalf("IC_REGION = %q, want unset after failed init", value)
	}
}

func TestProviderInitReturnsRegionEnvError(t *testing.T) {
	const probe = "REDACT_PROBE_IBM_REGION"
	provider := &IBMProvider{}

	err := provider.Init([]string{"resource-group", probe + "\x00region", "vpc"})
	if err == nil {
		t.Fatal("expected region env error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "failed to set env IC_REGION") {
		t.Fatalf("Init error = %q, want IC_REGION context", err)
	}
	if strings.Contains(msg, probe) {
		t.Fatalf("Init error = %q, want env value redacted", err)
	}
}

func TestProviderValidateImportRequiresAPIKey(t *testing.T) {
	t.Setenv("IC_API_KEY", "")

	provider := &IBMProvider{}
	err := provider.ValidateImport([]string{"ibm_is_image"})
	if !errors.Is(err, errMissingICAPIKey) {
		t.Fatalf("expected missing API key error, got %v", err)
	}
}

func TestProviderValidateImportChecksToolchainTarget(t *testing.T) {
	t.Setenv("IC_API_KEY", "api-key")
	t.Setenv("IBM_CD_TOOLCHAIN_TARGET", "not-a-guid")

	provider := &IBMProvider{}
	if err := provider.ValidateImport([]string{"ibm_is_image"}); err != nil {
		t.Fatalf("expected unrelated IBM import to ignore toolchain target, got %v", err)
	}
	if err := provider.ValidateImport([]string{"ibm_cd_toolchain"}); !errors.Is(err, errInvalidCDToolchainTarget) {
		t.Fatalf("expected invalid toolchain target error, got %v", err)
	}
}

func TestToolchainGeneratorReturnsInvalidTargetError(t *testing.T) {
	t.Setenv("IBM_CD_TOOLCHAIN_TARGET", "not-a-guid")

	generator := &ToolchainGenerator{}
	generator.SetArgs(map[string]interface{}{
		"region": DefaultRegion,
	})

	err := generator.InitResources()
	if !errors.Is(err, errInvalidCDToolchainTarget) {
		t.Fatalf("expected invalid toolchain target error, got %v", err)
	}
}

func TestProviderInitDefaultsRegionWithAPIKey(t *testing.T) {
	t.Setenv("IC_API_KEY", "api-key")

	provider := &IBMProvider{}
	if err := provider.Init([]string{"", "", ""}); err != nil {
		t.Fatalf("expected provider initialization to succeed: %v", err)
	}
	if provider.Region != DefaultRegion {
		t.Fatalf("expected default region %q, got %q", DefaultRegion, provider.Region)
	}
}

func TestImageGeneratorReturnsMissingAPIKeyError(t *testing.T) {
	t.Setenv("IC_API_KEY", "")

	generator := &ImageGenerator{}
	generator.SetArgs(map[string]interface{}{
		"region": DefaultRegion,
	})

	err := generator.InitResources()
	if !errors.Is(err, errMissingICAPIKey) {
		t.Fatalf("expected missing API key error, got %v", err)
	}
}
