// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildInventoryReportsProviderGapsAndDocsDrift(t *testing.T) {
	root := t.TempDir()
	awsDir := filepath.Join(root, "providers", "aws")
	if err := os.MkdirAll(awsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(awsDir, "aws_provider.go"),
		"package aws\n\n"+
			"func (p *AWSProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {\n"+
			"\treturn map[string]terraformutils.ServiceGenerator{\n"+
			"\t\t\"example\": &AwsFacade{service: &ExampleGenerator{}},\n"+
			"\t}\n"+
			"}\n")
	writeFile(t, filepath.Join(awsDir, "example.go"),
		"package aws\n\n"+
			"type ExampleGenerator struct{}\n\n"+
			"func (g *ExampleGenerator) InitResources() {\n"+
			"\tterraformutils.NewSimpleResource(\"id\", \"name\", \"aws_example_supported\", \"aws\", nil)\n"+
			"\tterraformutils.NewSimpleResource(\"id\", \"name\", \"aws_example_undocumented\", \"aws\", nil)\n"+
			"}\n")

	tick := string(rune(96))
	docsPath := filepath.Join(root, "docs", "aws.md")
	writeFile(t, docsPath,
		"#### Supported services\n\n"+
			"*   "+tick+"example"+tick+"\n"+
			"    * "+tick+"aws_example_supported"+tick+"\n"+
			"    * "+tick+"aws_example_documented_only"+tick+"\n")

	skipListPath := filepath.Join(awsDir, "unsupported_resources.json")
	writeFile(t, skipListPath,
		"{\n"+
			"  \"version\": 1,\n"+
			"  \"resources\": [\n"+
			"    {\n"+
			"      \"resource\": \"aws_example_skipped\",\n"+
			"      \"service_family\": \"example\",\n"+
			"      \"reason\": \"Discovery requires parent context not available yet.\",\n"+
			"      \"evidence\": \"Terraform AWS provider schema check.\",\n"+
			"      \"status\": \"unsupported\"\n"+
			"    }\n"+
			"  ]\n"+
			"}\n")

	providerSchemaPath := filepath.Join(root, "schema.json")
	writeFile(t, providerSchemaPath,
		"{\n"+
			"  \"provider_schemas\": {\n"+
			"    \"registry.terraform.io/hashicorp/aws\": {\n"+
			"      \"resource_schemas\": {\n"+
			"        \"aws_example_supported\": {},\n"+
			"        \"aws_example_missing\": {},\n"+
			"        \"aws_example_skipped\": {}\n"+
			"      }\n"+
			"    }\n"+
			"  }\n"+
			"}\n")

	inv, err := buildInventory(options{
		awsDir:         awsDir,
		docsPath:       docsPath,
		providerSchema: providerSchemaPath,
		skipListPath:   skipListPath,
	})
	if err != nil {
		t.Fatalf("buildInventory() error = %v", err)
	}

	assertRecords(t, inv.DocsAudit.DocumentedButNotDetected, []resourceRecord{
		{Resource: "aws_example_documented_only", ServiceFamily: "example"},
	})
	assertRecords(t, inv.DocsAudit.DetectedButNotDocumented, []resourceRecord{
		{Resource: "aws_example_undocumented", ServiceFamily: "example"},
	})
	if len(inv.Families) != 1 {
		t.Fatalf("families len = %d, want 1", len(inv.Families))
	}
	family := inv.Families[0]
	assertStrings(t, family.ProviderGaps, []string{"aws_example_missing"})
	if len(family.SkippedResources) != 1 || family.SkippedResources[0].Resource != "aws_example_skipped" {
		t.Fatalf("skipped resources = %#v, want aws_example_skipped", family.SkippedResources)
	}
}

func TestReadSkipListValidatesRequiredFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unsupported_resources.json")
	writeFile(t, path,
		"{\n"+
			"  \"version\": 1,\n"+
			"  \"resources\": [\n"+
			"    {\n"+
			"      \"resource\": \"aws_example_missing_status\",\n"+
			"      \"service_family\": \"example\",\n"+
			"      \"reason\": \"Missing status should fail.\",\n"+
			"      \"source_notes\": \"test\"\n"+
			"    }\n"+
			"  ]\n"+
			"}\n")

	_, err := readSkipList(path)
	if err == nil {
		t.Fatal("readSkipList() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "missing a required field") {
		t.Fatalf("readSkipList() error = %q, want required field error", err)
	}
}

func TestWriteMarkdownOmitsProviderCountsWithoutSchema(t *testing.T) {
	var output bytes.Buffer
	err := writeMarkdown(&output, inventory{
		TerraformerResources: []resourceRecord{{Resource: "aws_example_supported", ServiceFamily: "example"}},
		Families: []familyInventory{
			{
				ServiceFamily:        "example",
				TerraformerResources: []string{"aws_example_supported"},
			},
		},
	}, "")
	if err != nil {
		t.Fatalf("writeMarkdown() error = %v", err)
	}
	if !strings.Contains(output.String(), "| Terraform AWS provider resources | not supplied |") {
		t.Fatalf("markdown output did not note missing provider schema:\n%s", output.String())
	}
	if strings.Contains(output.String(), "Terraform provider gaps") {
		t.Fatalf("markdown output included provider gap section without schema:\n%s", output.String())
	}
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertRecords(t *testing.T, got []resourceRecord, want []resourceRecord) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("records len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("records[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func assertStrings(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("strings len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("strings[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
