// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
)

type outputTestProvider struct {
	testProvider
}

func (p *outputTestProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %#o, want %#o", path, got, want)
	}
}

func TestExportPlanFileIsOwnerOnly(t *testing.T) {
	// A plan directory and planfile left by an older release must be tightened on overwrite.
	dir := filepath.Join(t.TempDir(), "terraformer")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	planPath := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(planPath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(planPath, 0o644); err != nil {
		t.Fatal(err)
	}

	plan := &ImportPlan{Provider: "test", Options: ImportOptions{Resources: []string{"svc"}}}
	if err := ExportPlanFile(plan, dir, "plan.json"); err != nil {
		t.Fatalf("ExportPlanFile() error = %v", err)
	}
	assertMode(t, dir, 0o700)
	assertMode(t, planPath, 0o600)

	newDir := filepath.Join(t.TempDir(), "fresh")
	if err := ExportPlanFile(plan, newDir, "plan.json"); err != nil {
		t.Fatalf("ExportPlanFile() error = %v", err)
	}
	assertMode(t, newDir, 0o700)
	assertMode(t, filepath.Join(newDir, "plan.json"), 0o600)
}

func TestPrintServiceWritesOwnerOnlyTfState(t *testing.T) {
	out := t.TempDir()
	options := ImportOptions{
		PathPattern: "{output}/{provider}/{service}",
		PathOutput:  out,
		State:       "local",
		Output:      "hcl",
	}
	provider := &outputTestProvider{testProvider: testProvider{name: "permtest"}}
	resources := []terraformutils.Resource{}
	if err := printService(provider, "svc", options, resources, map[string][]terraformutils.Resource{"svc": resources}); err != nil {
		t.Fatalf("printService() error = %v", err)
	}
	assertMode(t, filepath.Join(out, "permtest", "svc", "terraform.tfstate"), 0o600)
}
