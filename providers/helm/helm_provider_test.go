// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"errors"
	"strings"
	"testing"
)

func TestProviderInit(t *testing.T) {
	provider := &Provider{}
	if err := provider.Init(nil); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
}

func TestProviderSupportedServices(t *testing.T) {
	provider := &Provider{}
	services := provider.GetSupportedService()
	if _, ok := services["release"]; !ok {
		t.Fatalf("release service not registered: %#v", services)
	}
	if err := provider.InitService("release", false); err != nil {
		t.Fatalf("InitService(release) error = %v", err)
	}
	if provider.GetService() == nil {
		t.Fatal("InitService(release) did not set service")
	}
	if provider.GetService().GetName() != "release" {
		t.Fatalf("service name = %q, want release", provider.GetService().GetName())
	}
	if provider.GetService().GetProviderName() != "helm" {
		t.Fatalf("service provider name = %q, want helm", provider.GetService().GetProviderName())
	}
}

func TestProviderRejectsUnsupportedService(t *testing.T) {
	provider := &Provider{}
	if err := provider.InitService("chart", false); err == nil {
		t.Fatal("expected unsupported service error")
	} else if !strings.Contains(err.Error(), "chart not supported service") {
		t.Fatalf("InitService error = %q, want unsupported service", err)
	}
	if provider.GetService() != nil {
		t.Fatalf("unsupported service left stale service %T", provider.GetService())
	}
}

func TestProviderValidateImportRejectsReleaseUntilImplemented(t *testing.T) {
	provider := &Provider{}
	err := provider.ValidateImport([]string{"release"})
	if !errors.Is(err, ErrReleaseImportNotImplemented) {
		t.Fatalf("ValidateImport(release) error = %v, want %v", err, ErrReleaseImportNotImplemented)
	}
}
