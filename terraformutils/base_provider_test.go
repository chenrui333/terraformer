// SPDX-License-Identifier: Apache-2.0

package terraformutils

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestProviderGetConfig(t *testing.T) {
	want := cty.StringVal("test")
	p := &Provider{Config: want}
	if got := p.GetConfig(); !got.RawEquals(want) {
		t.Errorf("GetConfig() = %v, want %v", got, want)
	}
}

func TestProviderGetService(t *testing.T) {
	svc := &Service{Name: "test-svc"}
	p := &Provider{Service: svc}
	if got := p.GetService(); got != svc {
		t.Errorf("GetService() returned different service instance")
	}
}

func TestProviderGetBasicConfig(t *testing.T) {
	p := &Provider{}
	got := p.GetBasicConfig()
	if !got.Type().IsObjectType() {
		t.Errorf("GetBasicConfig() type = %v, want object", got.Type().FriendlyName())
	}
}

func TestSelectProviderService(t *testing.T) {
	service := &Service{}
	provider := &Provider{Service: &Service{Name: "stale"}}

	if ok := SelectProviderService(provider, map[string]ServiceGenerator{
		"vpc": service,
	}, "vpc", true, "aws"); !ok {
		t.Fatal("expected service selection to succeed")
	}
	if provider.Service != service {
		t.Fatalf("expected selected service to be stored, got %T", provider.Service)
	}
	if service.GetName() != "vpc" || service.GetProviderName() != "aws" || !service.Verbose {
		t.Fatalf("expected service metadata to be configured, got name=%q provider=%q verbose=%t", service.GetName(), service.GetProviderName(), service.Verbose)
	}

	if ok := SelectProviderService(provider, map[string]ServiceGenerator{}, "missing", false, "aws"); ok {
		t.Fatal("expected missing service selection to fail")
	}
	if provider.Service != nil {
		t.Fatalf("expected missing service to clear stale provider service, got %T", provider.Service)
	}
}

func TestProviderInitPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Init() did not panic")
		}
	}()
	p := &Provider{}
	_ = p.Init(nil)
}

func TestProviderGetNamePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("GetName() did not panic")
		}
	}()
	p := &Provider{}
	_ = p.GetName()
}

func TestProviderGetSupportedServicePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("GetSupportedService() did not panic")
		}
	}()
	p := &Provider{}
	_ = p.GetSupportedService()
}
