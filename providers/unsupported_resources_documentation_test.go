// SPDX-License-Identifier: Apache-2.0

package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/chenrui333/terraformer/internal/unsupportedresources"
)

const (
	unsupportedResourcesDocumentationPath = "../docs/unsupported-resources.md"
	unsupportedResourceStatusHeading      = "## Status Values"
	unsupportedResourceInventoryHeading   = "## Inventory"
)

// There are currently no non-provider directories directly under providers.
// Add any future shared directory here with a reason instead of silently
// excluding it from the documented inventory.
var unsupportedResourceInventoryExcludedDirectories = map[string]string{}

type parsedUnsupportedResourceStatusTable struct {
	statuses   map[string]struct{}
	duplicates []string
}

type unsupportedResourceProviderInventoryEntry struct {
	hasMetadata  bool
	hasLocalTest bool
}

type parsedUnsupportedResourceProviderInventory struct {
	entries    map[string]unsupportedResourceProviderInventoryEntry
	duplicates []string
}

type unsupportedResourceProviderFiles struct {
	hasMetadata  bool
	hasLocalTest bool
}

func TestUnsupportedResourceStatusesMatchDocumentation(t *testing.T) {
	markdown := readUnsupportedResourcesDocumentation(t)
	if err := validateUnsupportedResourceStatusDocumentation(markdown, stringSet(unsupportedresources.Statuses()...)); err != nil {
		t.Fatal(err)
	}
}

func TestParseUnsupportedResourceStatusTable(t *testing.T) {
	tests := []struct {
		name            string
		markdown        string
		allowed         map[string]struct{}
		wantErrContains string
	}{
		{
			name:     "valid table",
			markdown: statusTableMarkdown("alpha", "beta"),
			allowed:  stringSet("alpha", "beta"),
		},
		{
			name:            "missing section",
			markdown:        "# Unsupported resources\n\nNo status table.\n",
			allowed:         stringSet("alpha"),
			wantErrContains: "missing section",
		},
		{
			name: "missing table",
			markdown: "# Unsupported resources\n\n" + unsupportedResourceStatusHeading +
				"\n\nStatus prose only.\n",
			allowed:         stringSet("alpha"),
			wantErrContains: "missing Markdown table",
		},
		{
			name:            "missing canonical status",
			markdown:        statusTableMarkdown("alpha"),
			allowed:         stringSet("alpha", "beta"),
			wantErrContains: "missing documentation statuses: [beta]",
		},
		{
			name:            "unknown status",
			markdown:        statusTableMarkdown("alpha", "gamma"),
			allowed:         stringSet("alpha"),
			wantErrContains: "unknown documented statuses: [gamma]",
		},
		{
			name:            "duplicate status",
			markdown:        statusTableMarkdown("alpha", "alpha"),
			allowed:         stringSet("alpha"),
			wantErrContains: "duplicate documented statuses: [alpha]",
		},
		{
			name: "malformed table row",
			markdown: "# Unsupported resources\n\n" + unsupportedResourceStatusHeading + "\n\n" +
				"| Status | Meaning |\n" +
				"| --- | --- |\n" +
				"| alpha | Missing code delimiters. |\n",
			allowed:         stringSet("alpha"),
			wantErrContains: "status must be a backtick-delimited",
		},
		{
			name: "prose outside table does not count",
			markdown: "# Unsupported resources\n\nThe `beta` status appears in prose.\n\n" +
				statusTableMarkdown("alpha"),
			allowed:         stringSet("alpha", "beta"),
			wantErrContains: "missing documentation statuses: [beta]",
		},
		{
			name: "unrelated table does not count",
			markdown: "# Unsupported resources\n\n" + unsupportedResourceStatusHeading + "\n\n" +
				"| Other | Value |\n| --- | --- |\n| `gamma` | Unrelated table. |\n\n" +
				"| Status | Meaning |\n| --- | --- |\n| `alpha` | Meaning for alpha. |\n",
			allowed: stringSet("alpha"),
		},
		{
			name: "section ends at next level two heading",
			markdown: statusTableMarkdown("alpha") + "\n## Other\n\n" +
				"| Status | Meaning |\n| --- | --- |\n| `gamma` | Outside the section. |\n",
			allowed: stringSet("alpha"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateUnsupportedResourceStatusDocumentation(test.markdown, test.allowed)
			if test.wantErrContains == "" {
				if err != nil {
					t.Fatalf("validateUnsupportedResourceStatusDocumentation() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErrContains) {
				t.Fatalf("validateUnsupportedResourceStatusDocumentation() error = %v, want substring %q", err, test.wantErrContains)
			}
		})
	}
}

func TestUnsupportedResourceProviderInventoryMatchesFilesystem(t *testing.T) {
	markdown := readUnsupportedResourcesDocumentation(t)
	actual, err := collectUnsupportedResourceProviderFiles(".", unsupportedResourceInventoryExcludedDirectories)
	if err != nil {
		t.Fatalf("collect provider inventory: %v", err)
	}
	if err := validateUnsupportedResourceProviderInventory(markdown, actual); err != nil {
		t.Fatal(err)
	}
}

func TestParseUnsupportedResourceProviderInventory(t *testing.T) {
	actual := map[string]unsupportedResourceProviderFiles{
		"alpha": {hasMetadata: true, hasLocalTest: true},
		"beta":  {hasMetadata: false, hasLocalTest: false},
	}
	validRows := []string{
		"| alpha | yes | yes | metadata [details](alpha.md); `provider-specific` assertions |",
		"| beta | no | no | not present yet |",
	}

	tests := []struct {
		name            string
		markdown        string
		wantErrContains string
	}{
		{
			name:     "valid inventory",
			markdown: providerInventoryMarkdown(validRows...),
		},
		{
			name:            "missing provider row",
			markdown:        providerInventoryMarkdown(validRows[0]),
			wantErrContains: "missing provider rows: [beta]",
		},
		{
			name: "duplicate provider row",
			markdown: providerInventoryMarkdown(
				validRows[0],
				validRows[0],
				validRows[1],
			),
			wantErrContains: "duplicate provider rows: [alpha]",
		},
		{
			name: "unknown provider row",
			markdown: providerInventoryMarkdown(
				validRows[0],
				validRows[1],
				"| gamma | no | no | unknown provider |",
			),
			wantErrContains: "unknown provider rows: [gamma]",
		},
		{
			name: "incorrect metadata value",
			markdown: providerInventoryMarkdown(
				"| alpha | no | yes | stale metadata value |",
				validRows[1],
			),
			wantErrContains: "metadata file mismatches: [alpha (documented no, actual yes)]",
		},
		{
			name: "incorrect local test value",
			markdown: providerInventoryMarkdown(
				"| alpha | yes | no | stale test value |",
				validRows[1],
			),
			wantErrContains: "local test mismatches: [alpha (documented no, actual yes)]",
		},
		{
			name: "malformed row",
			markdown: providerInventoryMarkdown(
				"| alpha | yes | yes |",
				validRows[1],
			),
			wantErrContains: "inventory row must contain 4 columns",
		},
		{
			name: "notes containing Markdown punctuation",
			markdown: providerInventoryMarkdown(
				"| alpha | yes | yes | metadata [details](alpha.md); `provider-specific` assertions and escaped \\| note |",
				validRows[1],
			),
		},
		{
			name: "rows outside inventory are ignored",
			markdown: "# Unsupported resources\n\n| Provider | Has `unsupported_resources.json` | Has provider-local `unsupported_resources_test.go` | Notes |\n" +
				"| --- | --- | --- | --- |\n| gamma | no | no | outside inventory |\n\n" +
				providerInventoryMarkdown(validRows...) +
				"\n## Other\n\n| Provider | Has `unsupported_resources.json` | Has provider-local `unsupported_resources_test.go` | Notes |\n" +
				"| --- | --- | --- | --- |\n| gamma | no | no | outside inventory |\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateUnsupportedResourceProviderInventory(test.markdown, actual)
			if test.wantErrContains == "" {
				if err != nil {
					t.Fatalf("validateUnsupportedResourceProviderInventory() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErrContains) {
				t.Fatalf("validateUnsupportedResourceProviderInventory() error = %v, want substring %q", err, test.wantErrContains)
			}
		})
	}
}

func validateUnsupportedResourceStatusDocumentation(markdown string, allowed map[string]struct{}) error {
	parsed, err := parseUnsupportedResourceStatusTable(markdown)
	if err != nil {
		return err
	}

	missing := setDifference(allowed, parsed.statuses)
	unknown := setDifference(parsed.statuses, allowed)
	duplicates := sortedUniqueStrings(parsed.duplicates)
	problems := make([]string, 0, 3)
	if len(missing) > 0 {
		problems = append(problems, fmt.Sprintf("missing documentation statuses: %v", missing))
	}
	if len(unknown) > 0 {
		problems = append(problems, fmt.Sprintf("unknown documented statuses: %v", unknown))
	}
	if len(duplicates) > 0 {
		problems = append(problems, fmt.Sprintf("duplicate documented statuses: %v", duplicates))
	}
	if len(problems) > 0 {
		return fmt.Errorf("unsupported resource status table mismatch: %s", strings.Join(problems, "; "))
	}
	return nil
}

func parseUnsupportedResourceStatusTable(markdown string) (parsedUnsupportedResourceStatusTable, error) {
	section, err := markdownLevelTwoSection(markdown, unsupportedResourceStatusHeading)
	if err != nil {
		return parsedUnsupportedResourceStatusTable{}, err
	}
	rows, err := markdownTableRows(section, []string{"Status", "Meaning"})
	if err != nil {
		return parsedUnsupportedResourceStatusTable{}, fmt.Errorf("parse %s: %w", unsupportedResourceStatusHeading, err)
	}

	parsed := parsedUnsupportedResourceStatusTable{statuses: map[string]struct{}{}}
	for index, row := range rows {
		if len(row) != 2 {
			return parsedUnsupportedResourceStatusTable{}, fmt.Errorf("parse %s row %d: status row must contain 2 columns, got %d", unsupportedResourceStatusHeading, index+1, len(row))
		}
		status, err := backtickDelimitedStatus(row[0])
		if err != nil {
			return parsedUnsupportedResourceStatusTable{}, fmt.Errorf("parse %s row %d: %w", unsupportedResourceStatusHeading, index+1, err)
		}
		if strings.TrimSpace(row[1]) == "" {
			return parsedUnsupportedResourceStatusTable{}, fmt.Errorf("parse %s row %d: status %q is missing a meaning", unsupportedResourceStatusHeading, index+1, status)
		}
		if _, exists := parsed.statuses[status]; exists {
			parsed.duplicates = append(parsed.duplicates, status)
		}
		parsed.statuses[status] = struct{}{}
	}
	return parsed, nil
}

func backtickDelimitedStatus(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) < 3 || value[0] != '`' || value[len(value)-1] != '`' || strings.Count(value, "`") != 2 {
		return "", fmt.Errorf("status must be a backtick-delimited non-empty name, got %q", value)
	}
	status := value[1 : len(value)-1]
	if strings.TrimSpace(status) != status || status == "" {
		return "", fmt.Errorf("status must be a backtick-delimited non-empty name, got %q", value)
	}
	for _, character := range []byte(status) {
		if !isLowerASCII(character) && !isASCIIDigit(character) && character != '-' {
			return "", fmt.Errorf("status %q must contain only lowercase letters, digits, and hyphens", status)
		}
	}
	return status, nil
}

func validateUnsupportedResourceProviderInventory(markdown string, actual map[string]unsupportedResourceProviderFiles) error {
	parsed, err := parseUnsupportedResourceProviderInventory(markdown)
	if err != nil {
		return err
	}

	actualProviders := make(map[string]struct{}, len(actual))
	for provider := range actual {
		actualProviders[provider] = struct{}{}
	}
	documentedProviders := make(map[string]struct{}, len(parsed.entries))
	for provider := range parsed.entries {
		documentedProviders[provider] = struct{}{}
	}

	problems := make([]string, 0, 5)
	if missing := setDifference(actualProviders, documentedProviders); len(missing) > 0 {
		problems = append(problems, fmt.Sprintf("missing provider rows: %v", missing))
	}
	if unknown := setDifference(documentedProviders, actualProviders); len(unknown) > 0 {
		problems = append(problems, fmt.Sprintf("unknown provider rows: %v", unknown))
	}
	if duplicates := sortedUniqueStrings(parsed.duplicates); len(duplicates) > 0 {
		problems = append(problems, fmt.Sprintf("duplicate provider rows: %v", duplicates))
	}

	metadataMismatches := []string{}
	localTestMismatches := []string{}
	for provider, files := range actual {
		entry, ok := parsed.entries[provider]
		if !ok {
			continue
		}
		if entry.hasMetadata != files.hasMetadata {
			metadataMismatches = append(metadataMismatches, inventoryBooleanMismatch(provider, entry.hasMetadata, files.hasMetadata))
		}
		if entry.hasLocalTest != files.hasLocalTest {
			localTestMismatches = append(localTestMismatches, inventoryBooleanMismatch(provider, entry.hasLocalTest, files.hasLocalTest))
		}
	}
	sort.Strings(metadataMismatches)
	sort.Strings(localTestMismatches)
	if len(metadataMismatches) > 0 {
		problems = append(problems, fmt.Sprintf("metadata file mismatches: %v", metadataMismatches))
	}
	if len(localTestMismatches) > 0 {
		problems = append(problems, fmt.Sprintf("local test mismatches: %v", localTestMismatches))
	}
	if len(problems) > 0 {
		return fmt.Errorf("unsupported resource provider inventory mismatch: %s", strings.Join(problems, "; "))
	}
	return nil
}

func parseUnsupportedResourceProviderInventory(markdown string) (parsedUnsupportedResourceProviderInventory, error) {
	section, err := markdownLevelTwoSection(markdown, unsupportedResourceInventoryHeading)
	if err != nil {
		return parsedUnsupportedResourceProviderInventory{}, err
	}
	rows, err := markdownTableRows(section, []string{
		"Provider",
		"Has `unsupported_resources.json`",
		"Has provider-local `unsupported_resources_test.go`",
		"Notes",
	})
	if err != nil {
		return parsedUnsupportedResourceProviderInventory{}, fmt.Errorf("parse %s: %w", unsupportedResourceInventoryHeading, err)
	}

	parsed := parsedUnsupportedResourceProviderInventory{entries: map[string]unsupportedResourceProviderInventoryEntry{}}
	for index, row := range rows {
		if len(row) != 4 {
			return parsedUnsupportedResourceProviderInventory{}, fmt.Errorf("parse %s row %d: inventory row must contain 4 columns, got %d", unsupportedResourceInventoryHeading, index+1, len(row))
		}
		provider := strings.TrimSpace(row[0])
		if provider == "" {
			return parsedUnsupportedResourceProviderInventory{}, fmt.Errorf("parse %s row %d: provider name is empty", unsupportedResourceInventoryHeading, index+1)
		}
		for _, character := range []byte(provider) {
			if !isLowerASCII(character) && !isASCIIDigit(character) && character != '-' && character != '_' {
				return parsedUnsupportedResourceProviderInventory{}, fmt.Errorf("parse %s row %d: invalid provider name %q", unsupportedResourceInventoryHeading, index+1, provider)
			}
		}
		hasMetadata, err := parseYesNo(row[1])
		if err != nil {
			return parsedUnsupportedResourceProviderInventory{}, fmt.Errorf("parse %s row %d metadata column: %w", unsupportedResourceInventoryHeading, index+1, err)
		}
		hasLocalTest, err := parseYesNo(row[2])
		if err != nil {
			return parsedUnsupportedResourceProviderInventory{}, fmt.Errorf("parse %s row %d local test column: %w", unsupportedResourceInventoryHeading, index+1, err)
		}
		notes := strings.TrimSpace(row[3])
		if notes == "" {
			return parsedUnsupportedResourceProviderInventory{}, fmt.Errorf("parse %s row %d: provider %q has empty notes", unsupportedResourceInventoryHeading, index+1, provider)
		}
		if _, exists := parsed.entries[provider]; exists {
			parsed.duplicates = append(parsed.duplicates, provider)
			continue
		}
		parsed.entries[provider] = unsupportedResourceProviderInventoryEntry{
			hasMetadata:  hasMetadata,
			hasLocalTest: hasLocalTest,
		}
	}
	return parsed, nil
}

func collectUnsupportedResourceProviderFiles(root string, excluded map[string]string) (map[string]unsupportedResourceProviderFiles, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	providers := map[string]unsupportedResourceProviderFiles{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, skip := excluded[entry.Name()]; skip {
			continue
		}
		hasMetadata, err := regularFileExists(filepath.Join(root, entry.Name(), "unsupported_resources.json"))
		if err != nil {
			return nil, err
		}
		hasLocalTest, err := regularFileExists(filepath.Join(root, entry.Name(), "unsupported_resources_test.go"))
		if err != nil {
			return nil, err
		}
		providers[entry.Name()] = unsupportedResourceProviderFiles{
			hasMetadata:  hasMetadata,
			hasLocalTest: hasLocalTest,
		}
	}
	return providers, nil
}

func markdownLevelTwoSection(markdown, heading string) ([]string, error) {
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	start := -1
	for index, line := range lines {
		if strings.TrimSpace(line) == heading {
			start = index + 1
			break
		}
	}
	if start == -1 {
		return nil, fmt.Errorf("missing section %q", heading)
	}

	end := len(lines)
	for index := start; index < len(lines); index++ {
		if strings.HasPrefix(strings.TrimSpace(lines[index]), "## ") {
			end = index
			break
		}
	}
	return lines[start:end], nil
}

func markdownTableRows(section []string, expectedHeader []string) ([][]string, error) {
	for index, line := range section {
		cells, ok, err := splitMarkdownTableRow(line)
		if err != nil || !ok || !equalStrings(cells, expectedHeader) {
			continue
		}
		if index+1 >= len(section) {
			return nil, fmt.Errorf("missing Markdown table separator")
		}
		separator, ok, err := splitMarkdownTableRow(section[index+1])
		if err != nil || !ok || !isMarkdownTableSeparator(separator, len(expectedHeader)) {
			return nil, fmt.Errorf("missing Markdown table separator after header")
		}

		rows := [][]string{}
		for rowIndex := index + 2; rowIndex < len(section); rowIndex++ {
			if !strings.HasPrefix(strings.TrimSpace(section[rowIndex]), "|") {
				break
			}
			row, ok, err := splitMarkdownTableRow(section[rowIndex])
			if err != nil {
				return nil, err
			}
			if !ok {
				break
			}
			rows = append(rows, row)
		}
		if len(rows) == 0 {
			return nil, fmt.Errorf("Markdown table has no data rows")
		}
		return rows, nil
	}
	return nil, fmt.Errorf("missing Markdown table with header %v", expectedHeader)
}

func splitMarkdownTableRow(line string) ([]string, bool, error) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") {
		return nil, false, nil
	}
	if !strings.HasSuffix(line, "|") {
		return nil, false, fmt.Errorf("malformed Markdown table row %q: missing trailing pipe", line)
	}

	line = strings.TrimSuffix(strings.TrimPrefix(line, "|"), "|")
	cells := []string{}
	var cell strings.Builder
	for index := 0; index < len(line); index++ {
		if line[index] == '\\' && index+1 < len(line) && line[index+1] == '|' {
			cell.WriteByte('|')
			index++
			continue
		}
		if line[index] == '|' {
			cells = append(cells, strings.TrimSpace(cell.String()))
			cell.Reset()
			continue
		}
		cell.WriteByte(line[index])
	}
	cells = append(cells, strings.TrimSpace(cell.String()))
	return cells, true, nil
}

func isMarkdownTableSeparator(cells []string, wantColumns int) bool {
	if len(cells) != wantColumns {
		return false
	}
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		cell = strings.TrimPrefix(cell, ":")
		cell = strings.TrimSuffix(cell, ":")
		if len(cell) < 3 || strings.Trim(cell, "-") != "" {
			return false
		}
	}
	return true
}

func regularFileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.Mode().IsRegular(), nil
}

func parseYesNo(value string) (bool, error) {
	switch strings.TrimSpace(value) {
	case "yes":
		return true, nil
	case "no":
		return false, nil
	default:
		return false, fmt.Errorf("want yes or no, got %q", value)
	}
}

func isLowerASCII(value byte) bool {
	return value >= 'a' && value <= 'z'
}

func isASCIIDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func inventoryBooleanMismatch(provider string, documented, actual bool) string {
	return fmt.Sprintf("%s (documented %s, actual %s)", provider, yesNo(documented), yesNo(actual))
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func setDifference(left, right map[string]struct{}) []string {
	values := []string{}
	for value := range left {
		if _, ok := right[value]; !ok {
			values = append(values, value)
		}
	}
	sort.Strings(values)
	return values
}

func sortedUniqueStrings(values []string) []string {
	unique := map[string]struct{}{}
	for _, value := range values {
		unique[value] = struct{}{}
	}
	result := make([]string, 0, len(unique))
	for value := range unique {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func stringSet(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func readUnsupportedResourcesDocumentation(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(unsupportedResourcesDocumentationPath)
	if err != nil {
		t.Fatalf("read unsupported resources documentation: %v", err)
	}
	return string(data)
}

func statusTableMarkdown(statuses ...string) string {
	var markdown strings.Builder
	markdown.WriteString("# Unsupported resources\n\n")
	markdown.WriteString(unsupportedResourceStatusHeading)
	markdown.WriteString("\n\n| Status | Meaning |\n| --- | --- |\n")
	for _, status := range statuses {
		fmt.Fprintf(&markdown, "| `%s` | Meaning for %s. |\n", status, status)
	}
	return markdown.String()
}

func providerInventoryMarkdown(rows ...string) string {
	return "# Unsupported resources\n\n" + unsupportedResourceInventoryHeading + "\n\n" +
		"| Provider | Has `unsupported_resources.json` | Has provider-local `unsupported_resources_test.go` | Notes |\n" +
		"| --- | --- | --- | --- |\n" + strings.Join(rows, "\n") + "\n"
}
