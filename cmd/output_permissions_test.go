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

func assertOwnerOnly(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		t.Fatalf("%s mode = %#o, want no group/other permissions", path, perm)
	}
}

func TestExportPlanFileIsOwnerOnly(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "terraformer")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A planfile left by an older release must be tightened on overwrite.
	planPath := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(planPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(planPath, 0o644); err != nil {
		t.Fatal(err)
	}

	plan := &ImportPlan{Provider: "test", Options: ImportOptions{Resources: []string{"svc"}}}
	if err := ExportPlanFile(plan, dir, "plan.json"); err != nil {
		t.Fatalf("ExportPlanFile() error = %v", err)
	}
	assertOwnerOnly(t, planPath)

	newDir := filepath.Join(t.TempDir(), "fresh")
	if err := ExportPlanFile(plan, newDir, "plan.json"); err != nil {
		t.Fatalf("ExportPlanFile() error = %v", err)
	}
	assertOwnerOnly(t, filepath.Join(newDir, "plan.json"))
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
	assertOwnerOnly(t, filepath.Join(out, "permtest", "svc", "terraform.tfstate"))
}
