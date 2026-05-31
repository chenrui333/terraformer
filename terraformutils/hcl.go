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

	"github.com/hashicorp/hcl/v2/hclwrite"
)

var (
	unsafeChars       = regexp.MustCompile(`[^0-9A-Za-z_\-]`)
	hclObjectKeyChars = regexp.MustCompile(`^[A-Za-z_][0-9A-Za-z_-]*$`)
	hclReservedKeys   = map[string]struct{}{
		"false": {},
		"for":   {},
		"if":    {},
		"in":    {},
		"null":  {},
		"true":  {},
	}
)

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
	var decoded interface{}
	if err := json.Unmarshal(dataBytesJSON, &decoded); err != nil {
		log.Println(string(dataBytesJSON))
		return []byte{}, fmt.Errorf("error parsing terraform json: %w", err)
	}
	root, ok := decoded.(map[string]interface{})
	if !ok {
		return []byte{}, fmt.Errorf("error writing HCL: root is %T", decoded)
	}

	var b strings.Builder
	if err := hclWriteRoot(&b, root, mapsObjects, sort); err != nil {
		return nil, fmt.Errorf("error writing HCL: %w", err)
	}
	s := b.String()

	// Remove extra whitespace...
	s = strings.ReplaceAll(s, "\n\n", "\n")

	// ...but leave whitespace between resources
	s = strings.ReplaceAll(s, "}\nresource", "}\n\nresource")

	// Apply Terraform style (alignment etc.)
	formatted := hclwrite.Format([]byte(s))

	return formatted, nil
}

func hclWriteRoot(b *strings.Builder, root map[string]interface{}, mapsObjects map[string]struct{}, sortOutput bool) error {
	for _, key := range orderedHCLKeys(root, sortOutput) {
		switch key {
		case "provider", "output":
			blocks, ok := root[key].(map[string]interface{})
			if !ok {
				return fmt.Errorf("%s section is %T", key, root[key])
			}
			for _, label := range orderedHCLKeys(blocks, sortOutput) {
				body, ok := blocks[label].(map[string]interface{})
				if !ok {
					return fmt.Errorf("%s %s body is %T", key, label, blocks[label])
				}
				fmt.Fprintf(b, "%s %s {\n", key, quoteHCLLabel(label))
				if err := hclWriteBlockBody(b, body, "", 2, mapsObjects, sortOutput); err != nil {
					return err
				}
				b.WriteString("}\n\n")
			}
		case "resource":
			resources, ok := root[key].(map[string]interface{})
			if !ok {
				return fmt.Errorf("resource section is %T", root[key])
			}
			for _, resourceType := range orderedHCLKeys(resources, sortOutput) {
				instances, ok := resources[resourceType].(map[string]interface{})
				if !ok {
					return fmt.Errorf("resource %s section is %T", resourceType, resources[resourceType])
				}
				for _, name := range orderedHCLKeys(instances, sortOutput) {
					body, ok := instances[name].(map[string]interface{})
					if !ok {
						return fmt.Errorf("resource %s.%s body is %T", resourceType, name, instances[name])
					}
					fmt.Fprintf(b, "resource %s %s {\n", quoteHCLLabel(resourceType), quoteHCLLabel(name))
					if err := hclWriteBlockBody(b, body, "", 2, mapsObjects, sortOutput); err != nil {
						return err
					}
					b.WriteString("}\n\n")
				}
			}
		case "terraform":
			body, ok := root[key].(map[string]interface{})
			if !ok {
				return fmt.Errorf("terraform section is %T", root[key])
			}
			b.WriteString("terraform {\n")
			if err := hclWriteBlockBody(b, body, "", 2, mapsObjects, sortOutput); err != nil {
				return err
			}
			b.WriteString("}\n\n")
		default:
			if err := hclWriteConfigEntry(b, key, root[key], key, 0, mapsObjects, sortOutput); err != nil {
				return err
			}
		}
	}
	return nil
}

func hclWriteBlockBody(b *strings.Builder, body map[string]interface{}, parentPath string, indent int, mapsObjects map[string]struct{}, sortOutput bool) error {
	for _, key := range orderedHCLKeys(body, sortOutput) {
		path := key
		if parentPath != "" {
			path = parentPath + "." + key
		}
		if key == "required_providers" {
			if err := hclWriteRequiredProviders(b, body[key], indent, sortOutput); err != nil {
				return err
			}
			continue
		}
		if err := hclWriteConfigEntry(b, key, body[key], path, indent, mapsObjects, sortOutput); err != nil {
			return err
		}
	}
	return nil
}

func hclWriteRequiredProviders(b *strings.Builder, value interface{}, indent int, sortOutput bool) error {
	indentText := strings.Repeat(" ", indent)
	b.WriteString(indentText + "required_providers {\n")
	providers := map[string]interface{}{}
	switch value := value.(type) {
	case []interface{}:
		for _, item := range value {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				return fmt.Errorf("required_providers entry is %T", item)
			}
			for provider, config := range itemMap {
				providers[provider] = config
			}
		}
	case map[string]interface{}:
		providers = value
	default:
		return fmt.Errorf("required_providers is %T", value)
	}
	for _, provider := range orderedHCLKeys(providers, sortOutput) {
		b.WriteString(strings.Repeat(" ", indent+2) + formatMapKey(provider) + " = ")
		if err := hclWriteTerraformValue(b, providers[provider], indent+2, sortOutput); err != nil {
			return err
		}
		b.WriteString("\n")
	}
	b.WriteString(indentText + "}\n")
	return nil
}

func hclWriteConfigEntry(b *strings.Builder, key string, value interface{}, path string, indent int, mapsObjects map[string]struct{}, sortOutput bool) error {
	indentText := strings.Repeat(" ", indent)
	if valueMap, ok := value.(map[string]interface{}); ok {
		if _, isMapObject := mapsObjects[path]; !isMapObject {
			b.WriteString(indentText + formatMapKey(key) + " {\n")
			if err := hclWriteBlockBody(b, valueMap, path, indent+2, mapsObjects, sortOutput); err != nil {
				return err
			}
			b.WriteString(indentText + "}\n")
			return nil
		}
	}
	if valueList, ok := value.([]interface{}); ok && hclListShouldRenderAsBlocks(valueList, path, mapsObjects) {
		for _, item := range valueList {
			itemMap := item.(map[string]interface{})
			b.WriteString(indentText + formatMapKey(key) + " {\n")
			if err := hclWriteBlockBody(b, itemMap, path, indent+2, mapsObjects, sortOutput); err != nil {
				return err
			}
			b.WriteString(indentText + "}\n")
		}
		return nil
	}
	b.WriteString(indentText + formatMapKey(key) + " = ")
	if err := hclWriteTerraformValue(b, value, indent, sortOutput); err != nil {
		return err
	}
	b.WriteString("\n")
	return nil
}

func hclListShouldRenderAsBlocks(value []interface{}, path string, mapsObjects map[string]struct{}) bool {
	if len(value) == 0 {
		return false
	}
	if _, isMapObject := mapsObjects[path]; isMapObject {
		return false
	}
	for _, item := range value {
		if _, ok := item.(map[string]interface{}); !ok {
			return false
		}
	}
	return true
}

func hclWriteTerraformValue(b *strings.Builder, value interface{}, indent int, sortOutput bool) error {
	switch value := value.(type) {
	case map[string]interface{}:
		return hclWriteTerraformMap(b, value, indent, sortOutput)
	case []interface{}:
		return hclWriteTerraformList(b, value, indent, sortOutput)
	case string:
		if strings.HasPrefix(value, "<<") {
			b.WriteString(formatHeredoc(value))
			return nil
		}
		b.WriteString(quoteRawHCLString(value))
		return nil
	default:
		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}
		b.Write(raw)
		return nil
	}
}

func hclWriteTerraformMap(b *strings.Builder, value map[string]interface{}, indent int, sortOutput bool) error {
	if len(value) == 0 {
		b.WriteString("{}")
		return nil
	}
	b.WriteString("{\n")
	childIndent := strings.Repeat(" ", indent+2)
	for _, key := range orderedHCLKeys(value, sortOutput) {
		b.WriteString(childIndent + formatMapKey(key) + " = ")
		if err := hclWriteTerraformValue(b, value[key], indent+2, sortOutput); err != nil {
			return err
		}
		b.WriteString("\n")
	}
	b.WriteString(strings.Repeat(" ", indent) + "}")
	return nil
}

func hclWriteTerraformList(b *strings.Builder, value []interface{}, indent int, sortOutput bool) error {
	if len(value) == 0 {
		b.WriteString("[]")
		return nil
	}
	b.WriteString("[\n")
	childIndent := strings.Repeat(" ", indent+2)
	for _, item := range value {
		b.WriteString(childIndent)
		if err := hclWriteTerraformValue(b, item, indent+2, sortOutput); err != nil {
			return err
		}
		b.WriteString(",\n")
	}
	b.WriteString(strings.Repeat(" ", indent) + "]")
	return nil
}

func quoteRawHCLString(value string) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "\"" + value + "\""
	}
	return string(raw)
}

func formatHeredoc(value string) string {
	lines := strings.Split(value, "\n")
	if len(lines) < 3 {
		return value
	}
	jsonTest := strings.Join(lines[1:len(lines)-1], "\n")
	var tmp interface{}
	if err := json.Unmarshal([]byte(jsonTest), &tmp); err != nil {
		return value
	}
	dataJSONBytes, err := json.MarshalIndent(tmp, "", "  ")
	if err != nil {
		return value
	}
	jsonData := append([]string{lines[0]}, strings.Split(string(dataJSONBytes), "\n")...)
	jsonData = append(jsonData, lines[len(lines)-1])
	return strings.Join(jsonData, "\n")
}

func orderedHCLKeys(value map[string]interface{}, sortOutput bool) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	if sortOutput {
		sort.Strings(keys)
	}
	return keys
}

func formatMapKey(key string) string {
	if isSafeHCLObjectKey(key) {
		return key
	}
	return quoteHCLString(key)
}

func isSafeHCLObjectKey(key string) bool {
	if !hclObjectKeyChars.MatchString(key) {
		return false
	}
	_, reserved := hclReservedKeys[key]
	return !reserved
}

func quoteHCLLabel(key string) string {
	raw, err := json.Marshal(key)
	if err != nil {
		return "\"" + key + "\""
	}
	return string(raw)
}

func quoteHCLString(value string) string {
	raw, err := json.Marshal(escapeTerraformTemplateMarkers(value))
	if err != nil {
		return "\"" + value + "\""
	}
	return string(raw)
}

func escapeTerraformTemplateMarkers(value string) string {
	value = strings.ReplaceAll(value, "${", "$${")
	value = strings.ReplaceAll(value, "%{", "%%{")
	return value
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
		if key == "object" || value == nil {
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
	case string:
		b.WriteString(quoteHCLString(value))
		return nil
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
