// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/providerwrapper"
)

type facadeImportProviderConfigurer struct {
	terraformutils.Service
	called bool
	err    error
}

func (s *facadeImportProviderConfigurer) ConfigureImportProvider(_ *providerwrapper.ProviderWrapper) error {
	s.called = true
	return s.err
}

func TestAwsFacadeConfigureImportProviderForwardsToService(t *testing.T) {
	service := &facadeImportProviderConfigurer{}
	facade := &AwsFacade{service: service}

	if err := facade.ConfigureImportProvider(nil); err != nil {
		t.Fatalf("ConfigureImportProvider returned error: %v", err)
	}
	if !service.called {
		t.Fatal("ConfigureImportProvider did not forward to wrapped service")
	}
}

func TestAwsFacadeConfigureImportProviderReturnsWrappedError(t *testing.T) {
	wantErr := errors.New("restart failed")
	facade := &AwsFacade{service: &facadeImportProviderConfigurer{err: wantErr}}

	if err := facade.ConfigureImportProvider(nil); !errors.Is(err, wantErr) {
		t.Fatalf("ConfigureImportProvider error = %v, want %v", err, wantErr)
	}
}

func TestAwsFacadeConfigureImportProviderNoop(t *testing.T) {
	facade := &AwsFacade{service: &terraformutils.Service{}}

	if err := facade.ConfigureImportProvider(nil); err != nil {
		t.Fatalf("ConfigureImportProvider without wrapped configurer returned error: %v", err)
	}
}
