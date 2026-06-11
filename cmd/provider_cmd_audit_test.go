// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestLaunchDarklyProviderCommandRegistration(t *testing.T) {
	provider := newLaunchDarklyProvider()
	if provider.GetName() != "launchdarkly" {
		t.Fatalf("provider name = %q, want launchdarkly", provider.GetName())
	}
	if _, ok := provider.GetSupportedService()["featureFlag"]; !ok {
		t.Fatal("LaunchDarkly featureFlag service is not registered")
	}
	if _, ok := provider.GetSupportedService()["project"]; !ok {
		t.Fatal("LaunchDarkly project service is not registered")
	}

	cmd := importerSubcommand(t, "launchdarkly")
	if cmd.PersistentFlags().Lookup("resources") == nil {
		t.Fatal("LaunchDarkly command missing resources flag")
	}
	if cmd.PersistentFlags().Lookup("filter") == nil {
		t.Fatal("LaunchDarkly command missing filter flag")
	}

	if _, ok := providerGenerators()["launchdarkly"]; !ok {
		t.Fatal("LaunchDarkly provider generator is not registered")
	}
}

func TestCloudflareProviderCommandRegistration(t *testing.T) {
	provider := newCloudflareProvider()
	if provider.GetName() != "cloudflare" {
		t.Fatalf("provider name = %q, want cloudflare", provider.GetName())
	}
	if _, ok := provider.GetSupportedService()["access"]; !ok {
		t.Fatal("Cloudflare access service is not registered")
	}
	if _, ok := provider.GetSupportedService()["settings"]; !ok {
		t.Fatal("Cloudflare settings service is not registered")
	}

	cmd := importerSubcommand(t, "cloudflare")
	if cmd.PersistentFlags().Lookup("resources") == nil {
		t.Fatal("Cloudflare command missing resources flag")
	}
	if cmd.PersistentFlags().Lookup("filter") == nil {
		t.Fatal("Cloudflare command missing filter flag")
	}

	if _, ok := providerGenerators()["cloudflare"]; !ok {
		t.Fatal("Cloudflare provider generator is not registered")
	}
}

func TestKubernetesProviderCommandRegistration(t *testing.T) {
	provider := newKubernetesProvider()
	if provider.GetName() != "kubernetes" {
		t.Fatalf("provider name = %q, want kubernetes", provider.GetName())
	}

	cmd := importerSubcommand(t, "kubernetes")
	if cmd.PersistentFlags().Lookup("resources") == nil {
		t.Fatal("Kubernetes command missing resources flag")
	}
	if cmd.PersistentFlags().Lookup("filter") == nil {
		t.Fatal("Kubernetes command missing filter flag")
	}
	if cmd.PersistentFlags().Lookup("verbose") == nil {
		t.Fatal("Kubernetes command missing verbose flag")
	}

	if _, ok := providerGenerators()["kubernetes"]; !ok {
		t.Fatal("Kubernetes provider generator is not registered")
	}
}

func TestMikrotikProviderCommandRegistration(t *testing.T) {
	provider := newMikrotikProvider()
	if provider.GetName() != "mikrotik" {
		t.Fatalf("provider name = %q, want mikrotik", provider.GetName())
	}
	if _, ok := provider.GetSupportedService()["dhcp_lease"]; !ok {
		t.Fatal("Mikrotik dhcp_lease service is not registered")
	}

	cmd := importerSubcommand(t, "mikrotik")
	if cmd.PersistentFlags().Lookup("resources") == nil {
		t.Fatal("Mikrotik command missing resources flag")
	}
	if cmd.PersistentFlags().Lookup("filter") == nil {
		t.Fatal("Mikrotik command missing filter flag")
	}

	if _, ok := providerGenerators()["mikrotik"]; !ok {
		t.Fatal("Mikrotik provider generator is not registered")
	}
}

func importerSubcommand(t *testing.T, use string) *cobra.Command {
	t.Helper()
	for _, importer := range providerImporterSubcommands() {
		cmd := importer(ImportOptions{})
		if cmd.Use == use {
			return cmd
		}
	}
	t.Fatalf("%s import subcommand is not registered", use)
	return nil
}
