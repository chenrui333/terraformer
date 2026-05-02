// SPDX-License-Identifier: Apache-2.0

package terraformutils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/hcl/ast"
	hclPrinter "github.com/hashicorp/hcl/hcl/printer"
	hclParser "github.com/hashicorp/hcl/json/parser"
)

// Copy code from https://github.com/kubernetes/kops project with few changes for support many provider and heredoc

const safeChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"

var unsafeChars = regexp.MustCompile(`[^0-9A-Za-z_\-]`)

// make HCL output reproducible by sorting the AST nodes
func sortHclTree(tree interface{}) {
	switch t := tree.(type) {
	case []*ast.ObjectItem:
		sort.Slice(t, func(i, j int) bool {
			var bI, bJ bytes.Buffer
			_, _ = hclPrinter.Fprint(&bI, t[i]), hclPrinter.Fprint(&bJ, t[j])
			return bI.String() < bJ.String()
		})
	case []ast.Node:
		sort.Slice(t, func(i, j int) bool {
			var bI, bJ bytes.Buffer
			_, _ = hclPrinter.Fprint(&bI, t[i]), hclPrinter.Fprint(&bJ, t[j])
			return bI.String() < bJ.String()
		})
	default:
	}
}

// sanitizer fixes up an invalid HCL AST, as produced by the HCL parser for JSON
type astSanitizer struct {
	sort bool
}

// output prints creates b printable HCL output and returns it.
func (v *astSanitizer) visit(n interface{}) {
	switch t := n.(type) {
	case *ast.File:
		v.visit(t.Node)
	case *ast.ObjectList:
		var index int
		if v.sort {
			sortHclTree(t.Items)
		}
		for index != len(t.Items) {
			v.visit(t.Items[index])
			index++
		}
	case *ast.ObjectKey:
	case *ast.ObjectItem:
		v.visitObjectItem(t)
	case *ast.LiteralType:
	case *ast.ListType:
		if v.sort {
			sortHclTree(t.List)
		}
	case *ast.ObjectType:
		if v.sort {
			sortHclTree(t.List)
		}
		v.visit(t.List)
	default:
		fmt.Printf(" unknown type: %T\n", n)
	}
}

func (v *astSanitizer) visitObjectItem(o *ast.ObjectItem) {
	for i, k := range o.Keys {
		if i == 0 {
			text := k.Token.Text
			if text != "" && text[0] == '"' && text[len(text)-1] == '"' {
				v := text[1 : len(text)-1]
				safe := true
				for _, c := range v {
					if !strings.ContainsRune(safeChars, c) {
						safe = false
						break
					}
				}
				if strings.HasPrefix(v, "--") { // if the key starts with "--", we must quote it. Seen in aws_glue_job.default_arguments parameter
					v = fmt.Sprintf(`"%s"`, v)
				}
				if safe {
					k.Token.Text = v
				}
			}
		}
	}
	switch t := o.Val.(type) {
	case *ast.LiteralType: // heredoc support
		if strings.HasPrefix(t.Token.Text, `"<<`) {
			t.Token.Text = t.Token.Text[1:]
			t.Token.Text = t.Token.Text[:len(t.Token.Text)-1]
			t.Token.Text = strings.ReplaceAll(t.Token.Text, `\n`, "\n")
			t.Token.Text = strings.ReplaceAll(t.Token.Text, `\t`, "")
			t.Token.Type = 10
			// check if text json for Unquote and Indent
			jsonTest := t.Token.Text
			lines := strings.Split(jsonTest, "\n")
			jsonTest = strings.Join(lines[1:len(lines)-1], "\n")
			jsonTest = strings.ReplaceAll(jsonTest, "\\\"", "\"")
			// it's json we convert to heredoc back
			var tmp interface{} = map[string]interface{}{}
			err := json.Unmarshal([]byte(jsonTest), &tmp)
			if err != nil {
				tmp = make([]interface{}, 0)
				err = json.Unmarshal([]byte(jsonTest), &tmp)
			}
			if err == nil {
				dataJSONBytes, err := json.MarshalIndent(tmp, "", "  ")
				if err == nil {
					jsonData := strings.Split(string(dataJSONBytes), "\n")
					// first line for heredoc
					jsonData = append([]string{lines[0]}, jsonData...)
					// last line for heredoc
					jsonData = append(jsonData, lines[len(lines)-1])
					hereDoc := strings.Join(jsonData, "\n")
					t.Token.Text = hereDoc
				}
			}
		}
	case *ast.ListType:
		if v.sort {
			sortHclTree(t.List)
		}
	default:
	}

	// A hack so that Assign.IsValid is true, so that the printer will output =
	o.Assign.Line = 1

	v.visit(o.Val)
}

func Print(data interface{}, mapsObjects map[string]struct{}, format string, sort bool) ([]byte, error) {
	switch format {
	case "hcl":
		return hclPrint(data, mapsObjects, sort)
	case "json":
		return jsonPrint(data)
	}
	return []byte{}, errors.New("error: unknown output format")
}

func hclPrint(data interface{}, mapsObjects map[string]struct{}, sort bool) ([]byte, error) {
	dataBytesJSON, err := jsonPrint(data)
	if err != nil {
		return dataBytesJSON, err
	}
	dataJSON := string(dataBytesJSON)
	nodes, err := hclParser.Parse([]byte(dataJSON))
	if err != nil {
		log.Println(dataJSON)
		return []byte{}, fmt.Errorf("error parsing terraform json: %w", err)
	}
	var sanitizer astSanitizer
	sanitizer.sort = sort
	sanitizer.visit(nodes)

	var b bytes.Buffer
	err = hclPrinter.Fprint(&b, nodes)
	if err != nil {
		return nil, fmt.Errorf("error writing HCL: %w", err)
	}
	s := b.String()

	// Remove extra whitespace...
	s = strings.ReplaceAll(s, "\n\n", "\n")

	// ...but leave whitespace between resources
	s = strings.ReplaceAll(s, "}\nresource", "}\n\nresource")

	// Apply Terraform style (alignment etc.)
	formatted, err := hclPrinter.Format([]byte(s))
	if err != nil {
		return nil, err
	}
	formatted = blockSyntaxAdjustments(formatted, mapsObjects)
	formatted = requiredProvidersObjectAdjustments(formatted)
	if err != nil {
		log.Println("Invalid HCL follows:")
		for i, line := range strings.Split(s, "\n") {
			fmt.Printf("%4d|\t%s\n", i+1, line)
		}
		return nil, fmt.Errorf("error formatting HCL: %w", err)
	}

	return formatted, nil
}

func blockSyntaxAdjustments(formatted []byte, mapsObjects map[string]struct{}) []byte {
	singletonListFix := regexp.MustCompile(`^\s*\w+ = {`)
	singletonListFixEnd := regexp.MustCompile(`^\s*}`)

	s := string(formatted)
	old := " = {"
	newEquals := " {"
	lines := strings.Split(s, "\n")
	prefix := make([]string, 0)
	for i, line := range lines {
		if singletonListFixEnd.MatchString(line) && len(prefix) > 0 {
			prefix = prefix[:len(prefix)-1]
			continue
		}
		if !singletonListFix.MatchString(line) {
			continue
		}
		key := strings.Trim(strings.Split(line, old)[0], " ")
		prefix = append(prefix, key)
		if _, exist := mapsObjects[strings.Join(prefix, ".")]; exist {
			continue
		}
		lines[i] = strings.ReplaceAll(line, old, newEquals)
	}
	s = strings.Join(lines, "\n")
	return []byte(s)
}

func formatMapKey(key string) string {
	if regexp.MustCompile("^[A-Za-z0-9_-]+$").MatchString(key) {
		return key
	}
	return quoteHCLLabel(key)
}

func quoteHCLLabel(key string) string {
	raw, err := json.Marshal(key)
	if err != nil {
		return "\"" + key + "\""
	}
	return string(raw)
}

func hclPrintManifestResources(resources []Resource, sortOutput bool) ([]byte, error) {
	if len(resources) == 0 {
		return []byte{}, nil
	}

	resources = append([]Resource{}, resources...)
	if sortOutput {
		sort.Slice(resources, func(i, j int) bool {
			return resources[i].ResourceName < resources[j].ResourceName
		})
	}

	var b strings.Builder
	for i, res := range resources {
		if i > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "resource %s %s {\n", quoteHCLLabel(res.InstanceInfo.Type), quoteHCLLabel(res.ResourceName))
		item := manifestConfigAttributes(res.Item)
		for _, key := range sortedHCLKeys(item) {
			b.WriteString("  " + formatMapKey(key) + " = ")
			if err := hclWriteValue(&b, item[key], 2, sortOutput); err != nil {
				return nil, err
			}
			b.WriteString("\n")
		}
		b.WriteString("}")
	}
	b.WriteString("\n")
	return []byte(b.String()), nil
}

func manifestConfigAttributes(item map[string]interface{}) map[string]interface{} {
	attributes := make(map[string]interface{}, len(item))
	for key, value := range item {
		if key == "object" {
			continue
		}
		attributes[key] = value
	}
	return attributes
}

func hclWriteValue(b *strings.Builder, value interface{}, indent int, sortOutput bool) error {
	switch value := value.(type) {
	case map[string]interface{}:
		return hclWriteMap(b, value, indent, sortOutput)
	case []interface{}:
		return hclWriteList(b, value, indent, sortOutput)
	default:
		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}
		b.Write(raw)
		return nil
	}
}

func hclWriteMap(b *strings.Builder, value map[string]interface{}, indent int, sortOutput bool) error {
	if len(value) == 0 {
		b.WriteString("{}")
		return nil
	}

	b.WriteString("{\n")
	childIndent := strings.Repeat(" ", indent+2)
	for _, key := range sortedHCLKeys(value) {
		b.WriteString(childIndent + formatMapKey(key) + " = ")
		if err := hclWriteValue(b, value[key], indent+2, sortOutput); err != nil {
			return err
		}
		b.WriteString("\n")
	}
	b.WriteString(strings.Repeat(" ", indent) + "}")
	return nil
}

func hclWriteList(b *strings.Builder, value []interface{}, indent int, sortOutput bool) error {
	if len(value) == 0 {
		b.WriteString("[]")
		return nil
	}

	b.WriteString("[\n")
	childIndent := strings.Repeat(" ", indent+2)
	for _, item := range value {
		b.WriteString(childIndent)
		if err := hclWriteValue(b, item, indent+2, sortOutput); err != nil {
			return err
		}
		b.WriteString(",\n")
	}
	b.WriteString(strings.Repeat(" ", indent) + "]")
	return nil
}

func sortedHCLKeys(value map[string]interface{}) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func appendHCLSection(base []byte, section []byte) []byte {
	base = bytes.TrimRight(base, "\n")
	section = bytes.TrimLeft(section, "\n")
	if len(bytes.TrimSpace(base)) == 0 {
		return section
	}
	if len(bytes.TrimSpace(section)) == 0 {
		return base
	}
	out := append([]byte{}, base...)
	out = append(out, '\n', '\n')
	out = append(out, section...)
	return out
}

func requiredProvidersObjectAdjustments(formatted []byte) []byte {
	s := string(formatted)
	requiredProvidersRe := regexp.MustCompile("required_providers \".*\" {")
	endBraceRe := regexp.MustCompile(`^\s*}`)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if requiredProvidersRe.MatchString(line) {
			parts := strings.Split(strings.TrimSpace(line), " ")
			provider := strings.ReplaceAll(parts[1], "\"", "")
			lines[i] = "\trequired_providers {"
			var innerBlock []string
			inner := i + 1
			for ; !endBraceRe.MatchString(lines[inner]); inner++ {
				innerBlock = append(innerBlock, "\t"+lines[inner])
			}
			lines[i+1] = "\t\t" + provider + " = {\n" + strings.Join(innerBlock, "\n") + "\n\t\t}"
			lines = append(lines[:i+2], lines[inner:]...)
			break
		}
	}
	s = strings.Join(lines, "\n")
	return []byte(s)
}

func escapeRune(s string) string {
	return fmt.Sprintf("-%04X-", s)
}

// Sanitize name for terraform style
func TfSanitize(name string) string {
	name = unsafeChars.ReplaceAllStringFunc(name, escapeRune)
	name = "tfer--" + name
	return name
}

// Print hcl file from TerraformResource + provider
func HclPrintResource(resources []Resource, providerData map[string]interface{}, output string, sort bool) ([]byte, error) {
	resourcesByType := map[string]map[string]interface{}{}
	mapsObjects := map[string]struct{}{}
	manifestResources := make([]Resource, 0)
	manifestResourceNames := map[string]struct{}{}
	indexRe := regexp.MustCompile(`\.[0-9]+`)
	for _, res := range resources {
		if output == "hcl" && res.InstanceInfo.Type == "kubernetes_manifest" {
			if _, exists := manifestResourceNames[res.ResourceName]; exists {
				log.Println(resources)
				log.Printf("[ERR]: duplicate resource found: %s.%s", res.InstanceInfo.Type, res.ResourceName)
				continue
			}
			manifestResourceNames[res.ResourceName] = struct{}{}
			manifestResources = append(manifestResources, res)
			continue
		}

		r := resourcesByType[res.InstanceInfo.Type]
		if r == nil {
			r = make(map[string]interface{})
			resourcesByType[res.InstanceInfo.Type] = r
		}

		if r[res.ResourceName] != nil {
			log.Println(resources)
			log.Printf("[ERR]: duplicate resource found: %s.%s", res.InstanceInfo.Type, res.ResourceName)
			continue
		}

		r[res.ResourceName] = res.Item

		for k := range res.InstanceState.Attributes {
			if strings.HasSuffix(k, ".%") {
				key := strings.TrimSuffix(k, ".%")
				mapsObjects[indexRe.ReplaceAllString(key, "")] = struct{}{}
			}
		}
	}

	data := map[string]interface{}{}
	if len(resourcesByType) > 0 {
		data["resource"] = resourcesByType
	}
	if len(providerData) > 0 {
		data["provider"] = providerData
	}
	var err error

	var hclBytes []byte
	if output == "hcl" && len(manifestResources) > 0 {
		if len(data) > 0 {
			hclBytes, err = Print(data, mapsObjects, output, sort)
			if err != nil {
				return []byte{}, err
			}
		}
		manifestBytes, err := hclPrintManifestResources(manifestResources, sort)
		if err != nil {
			return []byte{}, err
		}
		return appendHCLSection(hclBytes, manifestBytes), nil
	}

	hclBytes, err = Print(data, mapsObjects, output, sort)
	if err != nil {
		return []byte{}, err
	}
	return hclBytes, nil
}
