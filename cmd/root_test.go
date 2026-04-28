// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestNewCmdRootHasVersionFlag(t *testing.T) {
	cmd := NewCmdRoot()
	if cmd.Version == "" {
		t.Fatal("root command version is empty")
	}
}

func TestNewCmdRootHasSubcommands(t *testing.T) {
	cmd := NewCmdRoot()

	want := map[string]bool{
		"import":  false,
		"plan":    false,
		"version": false,
	}

	for _, sub := range cmd.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Errorf("expected subcommand %q not found", name)
		}
	}
}

func TestImportCmdHasAllProviderSubcommands(t *testing.T) {
	cmd := NewCmdRoot()

	var importCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "import" {
			importCmd = sub
			break
		}
	}
	if importCmd == nil {
		t.Fatal("import subcommand not found")
	}

	providers := providerImporterSubcommands()
	if len(importCmd.Commands()) < len(providers) {
		t.Errorf("import has %d subcommands, want at least %d (one per provider)",
			len(importCmd.Commands()), len(providers))
	}
}

func TestVersionSubcommandOutput(t *testing.T) {
	cmd := NewCmdRoot()
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("version command failed: %v", err)
	}
}

func TestImportCmdNoProviderShowsHelp(t *testing.T) {
	cmd := NewCmdRoot()
	cmd.SetArgs([]string{"import"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("import with no provider should show help, got error: %v", err)
	}
}

func TestRootHelpDoesNotError(t *testing.T) {
	cmd := NewCmdRoot()
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("root --help failed: %v", err)
	}
}

func TestImportHelpDoesNotError(t *testing.T) {
	cmd := NewCmdRoot()
	cmd.SetArgs([]string{"import", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("import --help failed: %v", err)
	}
}

func TestPlanHelpDoesNotError(t *testing.T) {
	cmd := NewCmdRoot()
	cmd.SetArgs([]string{"plan", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("plan --help failed: %v", err)
	}
}

func TestProviderGeneratorsNonEmpty(t *testing.T) {
	generators := providerGenerators()
	if len(generators) == 0 {
		t.Fatal("providerGenerators returned empty map")
	}
}

func TestProviderImporterSubcommandsNonEmpty(t *testing.T) {
	importers := providerImporterSubcommands()
	if len(importers) == 0 {
		t.Fatal("providerImporterSubcommands returned empty slice")
	}
}

func TestInvalidSubcommandErrors(t *testing.T) {
	cmd := NewCmdRoot()
	cmd.SetArgs([]string{"nonexistent"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for nonexistent subcommand, got nil")
	}
}
