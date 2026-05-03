// SPDX-License-Identifier: Apache-2.0

package cmd

import "testing"

func TestProvidersClearServiceOnUnsupportedInitService(t *testing.T) {
	for name, genFn := range providerGenerators() {
		t.Run(name, func(t *testing.T) {
			provider := genFn()
			services := provider.GetSupportedService()
			if len(services) == 0 {
				t.Skip("provider has no supported services")
			}

			var validService string
			for service := range services {
				validService = service
				break
			}

			if err := provider.InitService(validService, false); err != nil {
				t.Fatalf("expected supported service %q to initialize: %v", validService, err)
			}
			if provider.GetService() == nil {
				t.Fatalf("expected supported service %q to set provider service", validService)
			}

			if err := provider.InitService("__unsupported_service__", false); err == nil {
				t.Fatal("expected unsupported service to fail")
			}
			if provider.GetService() != nil {
				t.Fatalf("expected unsupported service to clear stale provider service, got %T", provider.GetService())
			}
		})
	}
}
