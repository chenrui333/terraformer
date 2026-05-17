// SPDX-License-Identifier: Apache-2.0

package cmd

import "testing"

func TestHelmProviderCommandRegistration(t *testing.T) {
	provider := newHelmProvider()
	if provider.GetName() != "helm" {
		t.Fatalf("provider name = %q, want helm", provider.GetName())
	}
	if _, ok := provider.GetSupportedService()["release"]; !ok {
		t.Fatal("helm release service is not registered")
	}

	importers := providerImporterSubcommands()
	foundImporter := false
	for _, importer := range importers {
		cmd := importer(ImportOptions{})
		if cmd.Use == "helm" {
			foundImporter = true
			if cmd.PersistentFlags().Lookup("resources") == nil {
				t.Fatal("helm command missing resources flag")
			}
			break
		}
	}
	if !foundImporter {
		t.Fatal("helm import subcommand is not registered")
	}

	generators := providerGenerators()
	if _, ok := generators["helm"]; !ok {
		t.Fatal("helm provider generator is not registered")
	}
}
