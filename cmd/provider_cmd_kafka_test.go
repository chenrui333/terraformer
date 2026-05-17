// SPDX-License-Identifier: Apache-2.0

package cmd

import "testing"

func TestKafkaProviderCommandRegistration(t *testing.T) {
	provider := newKafkaProvider()
	if provider.GetName() != "kafka" {
		t.Fatalf("provider name = %q, want kafka", provider.GetName())
	}
	if _, ok := provider.GetSupportedService()["topics"]; !ok {
		t.Fatal("kafka topics service is not registered")
	}
	if _, ok := provider.GetSupportedService()["acls"]; !ok {
		t.Fatal("kafka ACL service is not registered")
	}

	importers := providerImporterSubcommands()
	foundImporter := false
	for _, importer := range importers {
		cmd := importer(ImportOptions{})
		if cmd.Use == "kafka" {
			foundImporter = true
			if cmd.PersistentFlags().Lookup("bootstrap-servers") == nil {
				t.Fatal("kafka command missing bootstrap-servers flag")
			}
			break
		}
	}
	if !foundImporter {
		t.Fatal("kafka import subcommand is not registered")
	}

	generators := providerGenerators()
	if _, ok := generators["kafka"]; !ok {
		t.Fatal("kafka provider generator is not registered")
	}
}
