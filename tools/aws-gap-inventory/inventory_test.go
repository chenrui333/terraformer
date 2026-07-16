// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	if !strings.Contains(err.Error(), "missing status") {
		t.Fatalf("readSkipList() error = %q, want missing status error", err)
	}
}

func TestReadSkipListRejectsWhitespaceOnlyFields(t *testing.T) {
	tests := []struct {
		name            string
		mutate          func(*skipListEntry)
		wantErrContains string
		wantEvidence    string
		wantSourceNotes string
	}{
		{name: "resource", mutate: func(entry *skipListEntry) { entry.Resource = " \t " }, wantErrContains: "missing resource"},
		{name: "service family", mutate: func(entry *skipListEntry) { entry.ServiceFamily = " \t " }, wantErrContains: "missing service_family"},
		{name: "reason", mutate: func(entry *skipListEntry) { entry.Reason = "\t" }, wantErrContains: "missing reason"},
		{name: "status", mutate: func(entry *skipListEntry) { entry.Status = " \n " }, wantErrContains: "missing status"},
		{
			name: "evidence without source notes",
			mutate: func(entry *skipListEntry) {
				entry.Evidence = " \n "
				entry.SourceNotes = ""
			},
			wantErrContains: "requires evidence or source_notes",
		},
		{
			name: "source notes without evidence",
			mutate: func(entry *skipListEntry) {
				entry.Evidence = ""
				entry.SourceNotes = " \n "
			},
			wantErrContains: "requires evidence or source_notes",
		},
		{
			name: "valid source notes with whitespace evidence",
			mutate: func(entry *skipListEntry) {
				entry.Evidence = " \n "
				entry.SourceNotes = "Legacy source notes."
			},
			wantEvidence:    " \n ",
			wantSourceNotes: "Legacy source notes.",
		},
		{
			name: "valid evidence with whitespace source notes",
			mutate: func(entry *skipListEntry) {
				entry.Evidence = "Provider read-path evidence."
				entry.SourceNotes = " \n "
			},
			wantEvidence:    "Provider read-path evidence.",
			wantSourceNotes: " \n ",
		},
		{
			name: "reference",
			mutate: func(entry *skipListEntry) {
				entry.References = []string{"https://example.com/evidence", " \t "}
			},
			wantErrContains: "empty reference",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entry := validSkipListEntry("aws_example_action", "example")
			test.mutate(&entry)
			entries, err := readSkipList(writeSkipListFixture(t, entry))
			if test.wantErrContains != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErrContains) {
					t.Fatalf("readSkipList() error = %v, want substring %q", err, test.wantErrContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("readSkipList() error = %v", err)
			}
			if entries[0].Evidence != test.wantEvidence || entries[0].SourceNotes != test.wantSourceNotes {
				t.Fatalf("readSkipList() = %#v, want evidence %q and source_notes %q preserved", entries[0], test.wantEvidence, test.wantSourceNotes)
			}
		})
	}
}

func TestReadSkipListRejectsDuplicateResourceFamilyPairs(t *testing.T) {
	tests := []struct {
		name            string
		entries         []skipListEntry
		wantErrContains string
		wantOrder       []resourceRecord
	}{
		{
			name:            "exact duplicate pair",
			entries:         []skipListEntry{validSkipListEntry("aws_example_one", "example"), validSkipListEntry("aws_example_one", "example")},
			wantErrContains: `duplicate resource "aws_example_one" in service family "example"`,
		},
		{
			name: "duplicate pair with different status",
			entries: func() []skipListEntry {
				first := validSkipListEntry("aws_example_one", "example")
				second := validSkipListEntry("aws_example_one", "example")
				second.Status = "deferred"
				return []skipListEntry{first, second}
			}(),
			wantErrContains: `duplicate resource "aws_example_one" in service family "example"`,
		},
		{
			name: "duplicate pair with different reason",
			entries: func() []skipListEntry {
				first := validSkipListEntry("aws_example_one", "example")
				second := validSkipListEntry("aws_example_one", "example")
				second.Reason = "A different limitation."
				return []skipListEntry{first, second}
			}(),
			wantErrContains: `duplicate resource "aws_example_one" in service family "example"`,
		},
		{
			name:      "same resource in different families",
			entries:   []skipListEntry{validSkipListEntry("aws_example_one", "zeta"), validSkipListEntry("aws_example_one", "alpha")},
			wantOrder: []resourceRecord{{Resource: "aws_example_one", ServiceFamily: "alpha"}, {Resource: "aws_example_one", ServiceFamily: "zeta"}},
		},
		{
			name:      "different resources in same family",
			entries:   []skipListEntry{validSkipListEntry("aws_example_two", "example"), validSkipListEntry("aws_example_one", "example")},
			wantOrder: []resourceRecord{{Resource: "aws_example_one", ServiceFamily: "example"}, {Resource: "aws_example_two", ServiceFamily: "example"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entries, err := readSkipList(writeSkipListFixture(t, test.entries...))
			if test.wantErrContains != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErrContains) {
					t.Fatalf("readSkipList() error = %v, want substring %q", err, test.wantErrContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("readSkipList() error = %v", err)
			}
			gotOrder := make([]resourceRecord, 0, len(entries))
			for _, entry := range entries {
				gotOrder = append(gotOrder, resourceRecord{Resource: entry.Resource, ServiceFamily: entry.ServiceFamily})
			}
			assertRecords(t, gotOrder, test.wantOrder)
		})
	}
}

func TestReadSkipListPreservesCanonicalStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unsupported_resources.json")
	writeFile(t, path,
		"{\n"+
			"  \"version\": 1,\n"+
			"  \"resources\": [\n"+
			"    {\n"+
			"      \"resource\": \"aws_example_action\",\n"+
			"      \"service_family\": \"example\",\n"+
			"      \"reason\": \"The resource performs an operation.\",\n"+
			"      \"evidence\": \"The provider invokes an action instead of managing durable configuration.\",\n"+
			"      \"status\": \"action-style\"\n"+
			"    }\n"+
			"  ]\n"+
			"}\n")

	entries, err := readSkipList(path)
	if err != nil {
		t.Fatalf("readSkipList() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Status != "action-style" {
		t.Fatalf("readSkipList() = %#v, want preserved action-style status", entries)
	}
}

func TestReadSkipListRejectsUnknownStatuses(t *testing.T) {
	for _, status := range []string{"future-status", "needs-research", "unsafe-discovery", " action-style "} {
		t.Run(status, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "unsupported_resources.json")
			writeFile(t, path, fmt.Sprintf(`{
  "version": 1,
  "resources": [
    {
      "resource": "aws_example_action",
      "service_family": "example",
      "reason": "The resource performs an operation.",
      "evidence": "The provider invokes an action instead of managing durable configuration.",
      "status": %q
    }
  ]
}
`, status))

			_, err := readSkipList(path)
			if err == nil {
				t.Fatalf("readSkipList() error = nil, want invalid status error for %q", status)
			}
			if !strings.Contains(err.Error(), "invalid status") {
				t.Fatalf("readSkipList() error = %q, want invalid status error", err)
			}
		})
	}
}

func TestBuildInventoryPreservesDuplicateDocsResourceFamilies(t *testing.T) {
	root := t.TempDir()
	awsDir := filepath.Join(root, "providers", "aws")
	if err := os.MkdirAll(awsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(awsDir, "aws_provider.go"),
		"package aws\n\n"+
			"func (p *AWSProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {\n"+
			"\treturn map[string]terraformutils.ServiceGenerator{\n"+
			"\t\t\"wafv2_cloudfront\": &AwsFacade{service: NewWafv2CloudfrontGenerator()},\n"+
			"\t\t\"wafv2_regional\": &AwsFacade{service: NewWafv2RegionalGenerator()},\n"+
			"\t}\n"+
			"}\n")
	writeFile(t, filepath.Join(awsDir, "wafv2.go"),
		"package aws\n\n"+
			"type Wafv2Generator struct{}\n\n"+
			"func NewWafv2CloudfrontGenerator() *Wafv2Generator { return &Wafv2Generator{} }\n"+
			"func NewWafv2RegionalGenerator() *Wafv2Generator { return &Wafv2Generator{} }\n"+
			"func (g *Wafv2Generator) InitResources() {\n"+
			"\t_ = \"aws_wafv2_web_acl\"\n"+
			"\t_ = \"aws_wafv2_web_acl_association\"\n"+
			"}\n")

	tick := string(rune(96))
	docsPath := filepath.Join(root, "docs", "aws.md")
	writeFile(t, docsPath,
		"#### Supported services\n\n"+
			"*   "+tick+"wafv2_cloudfront"+tick+"\n"+
			"    * "+tick+"aws_wafv2_web_acl"+tick+"\n"+
			"*   "+tick+"wafv2_regional"+tick+"\n"+
			"    * "+tick+"aws_wafv2_web_acl"+tick+"\n"+
			"    * "+tick+"aws_wafv2_web_acl_association"+tick+"\n")
	skipListPath := filepath.Join(awsDir, "unsupported_resources.json")
	writeFile(t, skipListPath, "{\n  \"version\": 1,\n  \"resources\": []\n}\n")

	inv, err := buildInventory(options{
		awsDir:       awsDir,
		docsPath:     docsPath,
		skipListPath: skipListPath,
	})
	if err != nil {
		t.Fatalf("buildInventory() error = %v", err)
	}
	assertRecords(t, inv.DocsAudit.DocumentedButNotDetected, nil)
	assertRecords(t, inv.DocsAudit.DetectedButNotDocumented, nil)
	assertStrings(t, familyByName(t, inv, "wafv2_cloudfront").TerraformerResources, []string{"aws_wafv2_web_acl"})
	assertStrings(t, familyByName(t, inv, "wafv2_regional").TerraformerResources, []string{"aws_wafv2_web_acl", "aws_wafv2_web_acl_association"})
}

func TestBuildInventoryKeepsMixedLexResourcesInCorrectFamilies(t *testing.T) {
	root := t.TempDir()
	awsDir := filepath.Join(root, "providers", "aws")
	if err := os.MkdirAll(awsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(awsDir, "aws_provider.go"),
		"package aws\n\n"+
			"func (p *AWSProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {\n"+
			"\treturn map[string]terraformutils.ServiceGenerator{\n"+
			"\t\t\"lex\": &AwsFacade{service: &LexGenerator{}},\n"+
			"\t\t\"lexv2models\": &AwsFacade{service: &LexV2ModelsGenerator{}},\n"+
			"\t}\n"+
			"}\n")
	writeFile(t, filepath.Join(awsDir, "lex.go"),
		"package aws\n\n"+
			"type LexGenerator struct{}\n"+
			"type LexV2ModelsGenerator struct{}\n"+
			"func (g *LexGenerator) InitResources() {\n"+
			"\t_ = \"aws_lex_bot\"\n"+
			"}\n"+
			"func (g *LexV2ModelsGenerator) InitResources() {\n"+
			"\t_ = \"aws_lexv2models_bot\"\n"+
			"}\n")

	tick := string(rune(96))
	docsPath := filepath.Join(root, "docs", "aws.md")
	writeFile(t, docsPath,
		"#### Supported services\n\n"+
			"*   "+tick+"lex"+tick+"\n"+
			"    * "+tick+"aws_lex_bot"+tick+"\n"+
			"*   "+tick+"lexv2models"+tick+"\n"+
			"    * "+tick+"aws_lexv2models_bot"+tick+"\n")
	skipListPath := filepath.Join(awsDir, "unsupported_resources.json")
	writeFile(t, skipListPath, "{\n  \"version\": 1,\n  \"resources\": []\n}\n")

	inv, err := buildInventory(options{
		awsDir:       awsDir,
		docsPath:     docsPath,
		skipListPath: skipListPath,
	})
	if err != nil {
		t.Fatalf("buildInventory() error = %v", err)
	}
	assertRecords(t, inv.DocsAudit.DocumentedButNotDetected, nil)
	assertRecords(t, inv.DocsAudit.DetectedButNotDocumented, nil)
	assertStrings(t, familyByName(t, inv, "lex").TerraformerResources, []string{"aws_lex_bot"})
	assertStrings(t, familyByName(t, inv, "lexv2models").TerraformerResources, []string{"aws_lexv2models_bot"})
}

func TestBuildInventoryIgnoresAWSAttributeMapKeys(t *testing.T) {
	root := t.TempDir()
	awsDir := filepath.Join(root, "providers", "aws")
	if err := os.MkdirAll(awsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(awsDir, "aws_provider.go"),
		"package aws\n\n"+
			"func (p *AWSProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {\n"+
			"\treturn map[string]terraformutils.ServiceGenerator{\n"+
			"\t\t\"quicksight\": &AwsFacade{service: &QuickSightGenerator{}},\n"+
			"\t}\n"+
			"}\n")
	writeFile(t, filepath.Join(awsDir, "quicksight.go"),
		"package aws\n\n"+
			"type QuickSightGenerator struct{}\n"+
			"func (g *QuickSightGenerator) InitResources() {\n"+
			"\tterraformutils.NewResource(\"id\", \"name\", \"aws_quicksight_group\", \"aws\", map[string]string{\n"+
			"\t\t\"aws_account_id\": \"123456789012\",\n"+
			"\t}, nil, nil)\n"+
			"}\n")

	tick := string(rune(96))
	docsPath := filepath.Join(root, "docs", "aws.md")
	writeFile(t, docsPath,
		"#### Supported services\n\n"+
			"*   "+tick+"quicksight"+tick+"\n"+
			"    * "+tick+"aws_quicksight_group"+tick+"\n")
	skipListPath := filepath.Join(awsDir, "unsupported_resources.json")
	writeFile(t, skipListPath, "{\n  \"version\": 1,\n  \"resources\": []\n}\n")

	inv, err := buildInventory(options{
		awsDir:       awsDir,
		docsPath:     docsPath,
		skipListPath: skipListPath,
	})
	if err != nil {
		t.Fatalf("buildInventory() error = %v", err)
	}
	assertRecords(t, inv.DocsAudit.DocumentedButNotDetected, nil)
	assertRecords(t, inv.DocsAudit.DetectedButNotDocumented, nil)
	assertStrings(t, familyByName(t, inv, "quicksight").TerraformerResources, []string{"aws_quicksight_group"})
}

func TestFallbackServiceFamilyPreservesUnderscores(t *testing.T) {
	got := fallbackServiceFamily(filepath.Join("providers", "aws", "transit_gateway.go"))
	if got != "transit_gateway" {
		t.Fatalf("fallbackServiceFamily() = %q, want transit_gateway", got)
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

func familyByName(t *testing.T, inv inventory, name string) familyInventory {
	t.Helper()
	for _, family := range inv.Families {
		if family.ServiceFamily == name {
			return family
		}
	}
	t.Fatalf("family %q not found in %#v", name, inv.Families)
	return familyInventory{}
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

func writeSkipListFixture(t *testing.T, entries ...skipListEntry) string {
	t.Helper()
	data, err := json.Marshal(skipList{Version: 1, Resources: entries})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "unsupported_resources.json")
	writeFile(t, path, string(data))
	return path
}

func validSkipListEntry(resource, serviceFamily string) skipListEntry {
	return skipListEntry{
		Resource:      resource,
		ServiceFamily: serviceFamily,
		Reason:        "The resource cannot be imported safely.",
		Evidence:      "Provider and API behavior demonstrate the limitation.",
		Status:        "unsupported",
		References:    []string{"https://example.com/evidence"},
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
