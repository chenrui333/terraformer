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
