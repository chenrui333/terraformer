// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/pflag"
	"github.com/zclconf/go-cty/cty"
)

type testProvider struct {
	terraformutils.Provider
	name     string
	services map[string]terraformutils.ServiceGenerator
}

func (p *testProvider) Init(_ []string) error              { return nil }
func (p *testProvider) InitService(_ string, _ bool) error { return nil }
func (p *testProvider) GetName() string                    { return p.name }
func (p *testProvider) GetConfig() cty.Value               { return cty.EmptyObjectVal }
func (p *testProvider) GetBasicConfig() cty.Value          { return cty.EmptyObjectVal }
func (p *testProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return p.services
}
func (p *testProvider) GenerateFiles()                                         {}
func (p *testProvider) GetProviderData(_ ...string) map[string]interface{}     { return nil }
func (p *testProvider) GenerateOutputPath() error                              { return nil }
func (p *testProvider) GetResourceConnections() map[string]map[string][]string { return nil }

func TestPath(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		prov    string
		svc     string
		output  string
		want    string
	}{
		{"default pattern", DefaultPathPattern, "aws", "vpc", "generated", "generated/aws/vpc/"},
		{"no service", "{output}/{provider}/", "gcp", "", "out", "out/gcp/"},
		{"custom pattern", "{provider}-{service}", "azure", "rg", "", "azure-rg"},
		{"empty all", "{output}/{provider}/{service}/", "", "", "", "///"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Path(tc.pattern, tc.prov, tc.svc, tc.output); got != tc.want {
				t.Errorf("Path(%q, %q, %q, %q) = %q, want %q", tc.pattern, tc.prov, tc.svc, tc.output, got, tc.want)
			}
		})
	}
}

func TestProviderServices(t *testing.T) {
	tests := []struct {
		name     string
		services map[string]terraformutils.ServiceGenerator
		want     []string
	}{
		{"sorted services", map[string]terraformutils.ServiceGenerator{
			"vpc": &terraformutils.Service{},
			"ec2": &terraformutils.Service{},
			"s3":  &terraformutils.Service{},
		}, []string{"ec2", "s3", "vpc"}},
		{"empty", map[string]terraformutils.ServiceGenerator{}, nil},
		{"single", map[string]terraformutils.ServiceGenerator{
			"vpc": &terraformutils.Service{},
		}, []string{"vpc"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prov := &testProvider{name: "test", services: tc.services}
			got := providerServices(prov)
			if len(got) != len(tc.want) {
				t.Fatalf("providerServices() len = %d, want %d", len(got), len(tc.want))
			}
			for i, s := range tc.want {
				if got[i] != s {
					t.Errorf("providerServices()[%d] = %q, want %q", i, got[i], s)
				}
			}
		})
	}
}

func TestBaseProviderFlags(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	options := &ImportOptions{}
	baseProviderFlags(flags, options, "", "")

	wantFlags := []struct {
		name     string
		short    string
		defValue string
	}{
		{"connect", "c", "true"},
		{"compact", "C", "false"},
		{"resources", "r", "[]"},
		{"excludes", "x", "[]"},
		{"path-pattern", "p", DefaultPathPattern},
		{"path-output", "o", DefaultPathOutput},
		{"state", "s", DefaultState},
		{"bucket", "b", ""},
		{"filter", "f", "[]"},
		{"verbose", "v", "false"},
		{"no-sort", "S", "false"},
		{"output", "O", "hcl"},
		{"retry-number", "n", "5"},
		{"retry-sleep-ms", "m", "300"},
	}

	for _, wf := range wantFlags {
		t.Run(wf.name, func(t *testing.T) {
			f := flags.Lookup(wf.name)
			if f == nil {
				t.Fatalf("flag %q not registered", wf.name)
			}
			if f.Shorthand != wf.short {
				t.Errorf("flag %q shorthand = %q, want %q", wf.name, f.Shorthand, wf.short)
			}
			if f.DefValue != wf.defValue {
				t.Errorf("flag %q default = %q, want %q", wf.name, f.DefValue, wf.defValue)
			}
		})
	}
}

func TestListCmd(t *testing.T) {
	prov := &testProvider{
		name: "test",
		services: map[string]terraformutils.ServiceGenerator{
			"vpc": &terraformutils.Service{},
			"ec2": &terraformutils.Service{},
		},
	}
	cmd := listCmd(prov)

	if cmd.Use != "list" {
		t.Errorf("Use = %q, want %q", cmd.Use, "list")
	}
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("listCmd.Execute() error: %v", err)
	}
}

func TestPlanReplayCoversAllImportProviders(t *testing.T) {
	generators := providerGenerators()
	if len(generators) == 0 {
		t.Fatal("providerGenerators() is empty")
	}

	importers := providerImporterSubcommands()
	if len(generators) != len(importers) {
		t.Errorf("providerGenerators has %d entries but providerImporterSubcommands has %d — "+
			"every importer needs a matching generator for plan replay",
			len(generators), len(importers))
	}

	for name, genFn := range generators {
		t.Run("generator/"+name, func(t *testing.T) {
			prov := genFn()
			if got := prov.GetName(); got != name {
				t.Errorf("generator map key %q does not match GetName() %q", name, got)
			}
		})
	}

	seen := map[string]bool{}
	for _, fn := range importers {
		options := ImportOptions{}
		cmd := fn(options)
		name := cmd.Use
		t.Run("importer/"+name, func(t *testing.T) {
			if seen[name] {
				t.Errorf("duplicate import subcommand %q", name)
			}
			seen[name] = true
		})
	}
}
