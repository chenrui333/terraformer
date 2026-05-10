// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const awsProviderAddress = "registry.terraform.io/hashicorp/aws"

var (
	docsServiceRe   = regexp.MustCompile("^\\*\\s+`([^`]+)`")
	docsResourceRe  = regexp.MustCompile("^\\s+\\*\\s+`(aws_[^`]+)`")
	awsResourceRe   = regexp.MustCompile("^aws_[a-z0-9_]+$")
	validSkipStatus = map[string]bool{
		"deferred":         true,
		"needs-research":   true,
		"not-importable":   true,
		"unsupported":      true,
		"unsafe-discovery": true,
	}
)

type options struct {
	awsDir         string
	docsPath       string
	format         string
	providerSchema string
	skipListPath   string
}

type inventory struct {
	ProviderResources    []string          `json:"provider_resources,omitempty"`
	TerraformerResources []resourceRecord  `json:"terraformer_resources"`
	DocsResources        []resourceRecord  `json:"docs_resources"`
	SkippedResources     []skipListEntry   `json:"skipped_resources"`
	Families             []familyInventory `json:"families"`
	DocsAudit            docsAudit         `json:"docs_audit"`
}

type resourceRecord struct {
	Resource      string `json:"resource"`
	ServiceFamily string `json:"service_family"`
}

type familyInventory struct {
	ServiceFamily        string          `json:"service_family"`
	ProviderResources    []string        `json:"provider_resources,omitempty"`
	TerraformerResources []string        `json:"terraformer_resources"`
	SkippedResources     []skipListEntry `json:"skipped_resources,omitempty"`
	ProviderGaps         []string        `json:"provider_gaps,omitempty"`
}

type docsAudit struct {
	DocumentedButNotDetected []resourceRecord `json:"documented_but_not_detected"`
	DetectedButNotDocumented []resourceRecord `json:"detected_but_not_documented"`
}

type skipList struct {
	Version   int             `json:"version"`
	Resources []skipListEntry `json:"resources"`
}

type skipListEntry struct {
	Resource      string   `json:"resource"`
	ServiceFamily string   `json:"service_family"`
	Reason        string   `json:"reason"`
	Evidence      string   `json:"evidence,omitempty"`
	SourceNotes   string   `json:"source_notes,omitempty"`
	Status        string   `json:"status"`
	References    []string `json:"references,omitempty"`
}

func buildInventory(opts options) (inventory, error) {
	docsResources, err := parseDocsResources(opts.docsPath)
	if err != nil {
		return inventory{}, err
	}

	terraformerResources, err := scanTerraformerResources(opts.awsDir, docsResources)
	if err != nil {
		return inventory{}, err
	}

	skippedResources, err := readSkipList(opts.skipListPath)
	if err != nil {
		return inventory{}, err
	}

	var providerResources []string
	if opts.providerSchema != "" {
		providerResources, err = parseProviderSchema(opts.providerSchema)
		if err != nil {
			return inventory{}, err
		}
	}

	inv := inventory{
		ProviderResources:    providerResources,
		TerraformerResources: recordsFromMap(terraformerResources),
		DocsResources:        recordsFromMap(docsResources),
		SkippedResources:     skippedResources,
		DocsAudit:            compareDocs(terraformerResources, docsResources),
	}
	inv.Families = groupFamilies(providerResources, terraformerResources, docsResources, skippedResources)
	return inv, nil
}

func parseDocsResources(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open docs: %w", err)
	}
	defer file.Close()

	resources := map[string]string{}
	service := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if match := docsServiceRe.FindStringSubmatch(line); len(match) == 2 {
			service = match[1]
			continue
		}
		if match := docsResourceRe.FindStringSubmatch(line); len(match) == 2 && service != "" {
			resources[match[1]] = service
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan docs: %w", err)
	}
	return resources, nil
}

func scanTerraformerResources(awsDir string, docsResources map[string]string) (map[string]string, error) {
	serviceByFile, err := servicesByFile(awsDir)
	if err != nil {
		return nil, err
	}

	resources := map[string]string{}
	err = filepath.WalkDir(awsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if filepath.Base(path) == "aws_provider.go" {
			return nil
		}

		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		serviceFamily := serviceByFile[path]
		if serviceFamily == "" {
			serviceFamily = fallbackServiceFamily(path)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := unquote(literal.Value)
			if err != nil || !awsResourceRe.MatchString(value) {
				return true
			}
			if docsFamily := docsResources[value]; docsFamily != "" {
				resources[value] = docsFamily
				return true
			}
			resources[value] = serviceFamily
			return true
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resources, nil
}

func servicesByFile(awsDir string) (map[string]string, error) {
	providerPath := filepath.Join(awsDir, "aws_provider.go")
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, providerPath, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse aws provider: %w", err)
	}

	serviceByGenerator := map[string]string{}
	ast.Inspect(file, func(node ast.Node) bool {
		kv, ok := node.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		key, ok := kv.Key.(*ast.BasicLit)
		if !ok || key.Kind != token.STRING {
			return true
		}
		service, err := unquote(key.Value)
		if err != nil {
			return true
		}
		for _, generator := range generatorNames(kv.Value) {
			serviceByGenerator[generator] = service
		}
		return true
	})

	serviceByFile := map[string]string{}
	err = filepath.WalkDir(awsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || filepath.Base(path) == "aws_provider.go" {
			return nil
		}
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		for _, declaration := range file.Decls {
			genericDeclaration, ok := declaration.(*ast.GenDecl)
			if !ok || genericDeclaration.Tok != token.TYPE {
				continue
			}
			for _, spec := range genericDeclaration.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if service := serviceByGenerator[typeSpec.Name.Name]; service != "" {
					serviceByFile[path] = service
					return nil
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return serviceByFile, nil
}

func generatorNames(expr ast.Expr) []string {
	var names []string
	ast.Inspect(expr, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.CompositeLit:
			if ident, ok := value.Type.(*ast.Ident); ok && strings.HasSuffix(ident.Name, "Generator") {
				names = append(names, ident.Name)
			}
		case *ast.CallExpr:
			if ident, ok := value.Fun.(*ast.Ident); ok && strings.HasPrefix(ident.Name, "New") && strings.HasSuffix(ident.Name, "Generator") {
				names = append(names, ident.Name)
			}
		}
		return true
	})
	return names
}

func fallbackServiceFamily(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), ".go")
	return strings.ReplaceAll(base, "_", "-")
}

func parseProviderSchema(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read provider schema: %w", err)
	}

	var schema struct {
		ProviderSchemas map[string]struct {
			ResourceSchemas map[string]json.RawMessage `json:"resource_schemas"`
		} `json:"provider_schemas"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("decode provider schema: %w", err)
	}

	resourceSet := map[string]bool{}
	for providerAddress, provider := range schema.ProviderSchemas {
		if providerAddress != awsProviderAddress && !strings.HasSuffix(providerAddress, "/aws") {
			continue
		}
		for resource := range provider.ResourceSchemas {
			if awsResourceRe.MatchString(resource) {
				resourceSet[resource] = true
			}
		}
	}
	resources := keys(resourceSet)
	if len(resources) == 0 {
		return nil, fmt.Errorf("provider schema %s did not contain AWS resource schemas", path)
	}
	return resources, nil
}

func readSkipList(path string) ([]skipListEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read skip-list: %w", err)
	}
	var list skipList
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("decode skip-list: %w", err)
	}
	if list.Version != 1 {
		return nil, fmt.Errorf("unsupported skip-list version %d", list.Version)
	}
	for i, entry := range list.Resources {
		if entry.Resource == "" || entry.ServiceFamily == "" || entry.Reason == "" || entry.Status == "" {
			return nil, fmt.Errorf("skip-list resource %d is missing a required field", i)
		}
		if entry.Evidence == "" && entry.SourceNotes == "" {
			return nil, fmt.Errorf("skip-list resource %q requires evidence or source_notes", entry.Resource)
		}
		if !awsResourceRe.MatchString(entry.Resource) {
			return nil, fmt.Errorf("skip-list resource %d has invalid resource %q", i, entry.Resource)
		}
		if !validSkipStatus[entry.Status] {
			return nil, fmt.Errorf("skip-list resource %q has invalid status %q", entry.Resource, entry.Status)
		}
	}
	sort.Slice(list.Resources, func(i, j int) bool {
		if list.Resources[i].ServiceFamily == list.Resources[j].ServiceFamily {
			return list.Resources[i].Resource < list.Resources[j].Resource
		}
		return list.Resources[i].ServiceFamily < list.Resources[j].ServiceFamily
	})
	return list.Resources, nil
}

func compareDocs(terraformerResources, docsResources map[string]string) docsAudit {
	audit := docsAudit{}
	for resource, service := range docsResources {
		if _, ok := terraformerResources[resource]; !ok {
			audit.DocumentedButNotDetected = append(audit.DocumentedButNotDetected, resourceRecord{Resource: resource, ServiceFamily: service})
		}
	}
	for resource, service := range terraformerResources {
		if _, ok := docsResources[resource]; !ok {
			audit.DetectedButNotDocumented = append(audit.DetectedButNotDocumented, resourceRecord{Resource: resource, ServiceFamily: service})
		}
	}
	sortRecords(audit.DocumentedButNotDetected)
	sortRecords(audit.DetectedButNotDocumented)
	return audit
}

func groupFamilies(providerResources []string, terraformerResources, docsResources map[string]string, skippedResources []skipListEntry) []familyInventory {
	terraformerSet := map[string]bool{}
	for resource := range terraformerResources {
		terraformerSet[resource] = true
	}
	skipSet := map[string]skipListEntry{}
	for _, resource := range skippedResources {
		skipSet[resource.Resource] = resource
	}

	families := map[string]*familyInventory{}
	for resource, service := range terraformerResources {
		family := familyEntry(families, service)
		family.TerraformerResources = append(family.TerraformerResources, resource)
	}
	for _, resource := range providerResources {
		service := serviceFamily(resource, docsResources, terraformerResources, skipSet)
		family := familyEntry(families, service)
		family.ProviderResources = append(family.ProviderResources, resource)
		if !terraformerSet[resource] && skipSet[resource].Resource == "" {
			family.ProviderGaps = append(family.ProviderGaps, resource)
		}
	}
	for _, resource := range skippedResources {
		family := familyEntry(families, resource.ServiceFamily)
		family.SkippedResources = append(family.SkippedResources, resource)
	}

	result := make([]familyInventory, 0, len(families))
	for _, family := range families {
		sort.Strings(family.ProviderResources)
		sort.Strings(family.TerraformerResources)
		sort.Strings(family.ProviderGaps)
		sort.Slice(family.SkippedResources, func(i, j int) bool {
			return family.SkippedResources[i].Resource < family.SkippedResources[j].Resource
		})
		result = append(result, *family)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ServiceFamily < result[j].ServiceFamily
	})
	return result
}

func familyEntry(families map[string]*familyInventory, service string) *familyInventory {
	if service == "" {
		service = "unknown"
	}
	if _, ok := families[service]; !ok {
		families[service] = &familyInventory{ServiceFamily: service}
	}
	return families[service]
}

func serviceFamily(resource string, docsResources, terraformerResources map[string]string, skipSet map[string]skipListEntry) string {
	if service := docsResources[resource]; service != "" {
		return service
	}
	if service := terraformerResources[resource]; service != "" {
		return service
	}
	if skip := skipSet[resource]; skip.ServiceFamily != "" {
		return skip.ServiceFamily
	}
	return guessAWSService(resource)
}

func guessAWSService(resource string) string {
	name := strings.TrimPrefix(resource, "aws_")
	parts := strings.Split(name, "_")
	if len(parts) == 0 || parts[0] == "" {
		return "unknown"
	}
	if len(parts) >= 2 {
		switch parts[0] + "_" + parts[1] {
		case "cloudwatch_log":
			return "logs"
		case "ec2_transit":
			return "transit_gateway"
		case "elastic_beanstalk":
			return "elastic_beanstalk"
		case "kinesis_firehose":
			return "firehose"
		case "network_acl":
			return "nacl"
		case "nat_gateway":
			return "nat"
		case "sfn_activity", "sfn_state":
			return "sfn"
		}
	}
	switch parts[0] {
	case "lb":
		return "alb"
	case "volume":
		return "ebs"
	case "instance":
		return "ec2_instance"
	case "internet":
		return "igw"
	case "main":
		return "route_table"
	case "organizations":
		return "organization"
	case "route53":
		return "route53"
	case "route":
		return "route_table"
	case "vpn":
		return "vpn_connection"
	}
	return parts[0]
}

func writeJSON(writer io.Writer, inv inventory) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(inv)
}

func writeMarkdown(writer io.Writer, inv inventory, providerSchema string) error {
	var buffer bytes.Buffer
	buffer.WriteString("# AWS Provider Gap Inventory\n\n")
	buffer.WriteString("| Source | Count |\n")
	buffer.WriteString("| --- | ---: |\n")
	buffer.WriteString(fmt.Sprintf("| Terraformer detected resources | %d |\n", len(inv.TerraformerResources)))
	buffer.WriteString(fmt.Sprintf("| docs/aws.md resources | %d |\n", len(inv.DocsResources)))
	if providerSchema == "" {
		buffer.WriteString("| Terraform AWS provider resources | not supplied |\n")
	} else {
		buffer.WriteString(fmt.Sprintf("| Terraform AWS provider resources | %d |\n", len(inv.ProviderResources)))
	}
	buffer.WriteString(fmt.Sprintf("| Skip-list resources | %d |\n\n", len(inv.SkippedResources)))

	buffer.WriteString("## Docs Audit\n\n")
	writeRecordList(&buffer, "Documented in docs/aws.md but not detected in AWS provider code", inv.DocsAudit.DocumentedButNotDetected)
	writeRecordList(&buffer, "Detected in AWS provider code but missing from docs/aws.md", inv.DocsAudit.DetectedButNotDocumented)

	buffer.WriteString("## Service Families\n\n")
	buffer.WriteString("| Service family | Provider | Terraformer | Skipped | Gap |\n")
	buffer.WriteString("| --- | ---: | ---: | ---: | ---: |\n")
	for _, family := range inv.Families {
		providerCount := "-"
		gapCount := "-"
		if providerSchema != "" {
			providerCount = fmt.Sprintf("%d", len(family.ProviderResources))
			gapCount = fmt.Sprintf("%d", len(family.ProviderGaps))
		}
		buffer.WriteString(fmt.Sprintf("| %s | %s | %d | %d | %s |\n",
			family.ServiceFamily,
			providerCount,
			len(family.TerraformerResources),
			len(family.SkippedResources),
			gapCount,
		))
	}
	buffer.WriteString("\n")

	for _, family := range inv.Families {
		buffer.WriteString(fmt.Sprintf("### %s\n\n", family.ServiceFamily))
		writeStringList(&buffer, "Terraformer resources", family.TerraformerResources)
		if providerSchema != "" {
			writeStringList(&buffer, "Terraform provider gaps", family.ProviderGaps)
		}
		writeSkipList(&buffer, family.SkippedResources)
	}

	_, err := writer.Write(buffer.Bytes())
	return err
}

func writeRecordList(buffer *bytes.Buffer, title string, records []resourceRecord) {
	buffer.WriteString(fmt.Sprintf("### %s\n\n", title))
	if len(records) == 0 {
		buffer.WriteString("_None._\n\n")
		return
	}
	for _, record := range records {
		buffer.WriteString(fmt.Sprintf("- `%s` (%s)\n", record.Resource, record.ServiceFamily))
	}
	buffer.WriteString("\n")
}

func writeStringList(buffer *bytes.Buffer, title string, values []string) {
	buffer.WriteString(fmt.Sprintf("#### %s\n\n", title))
	if len(values) == 0 {
		buffer.WriteString("_None._\n\n")
		return
	}
	for _, value := range values {
		buffer.WriteString(fmt.Sprintf("- `%s`\n", value))
	}
	buffer.WriteString("\n")
}

func writeSkipList(buffer *bytes.Buffer, values []skipListEntry) {
	buffer.WriteString("#### Skipped resources\n\n")
	if len(values) == 0 {
		buffer.WriteString("_None._\n\n")
		return
	}
	for _, value := range values {
		buffer.WriteString(fmt.Sprintf("- `%s` (%s): %s\n", value.Resource, value.Status, value.Reason))
	}
	buffer.WriteString("\n")
}

func recordsFromMap(resources map[string]string) []resourceRecord {
	records := make([]resourceRecord, 0, len(resources))
	for resource, service := range resources {
		records = append(records, resourceRecord{Resource: resource, ServiceFamily: service})
	}
	sortRecords(records)
	return records
}

func sortRecords(records []resourceRecord) {
	sort.Slice(records, func(i, j int) bool {
		if records[i].ServiceFamily == records[j].ServiceFamily {
			return records[i].Resource < records[j].Resource
		}
		return records[i].ServiceFamily < records[j].ServiceFamily
	})
}

func keys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func unquote(value string) (string, error) {
	var out string
	if err := json.Unmarshal([]byte(value), &out); err != nil {
		return "", err
	}
	return out, nil
}
