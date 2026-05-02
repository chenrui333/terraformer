// SPDX-License-Identifier: Apache-2.0

package ibm

import (
	"errors"
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
