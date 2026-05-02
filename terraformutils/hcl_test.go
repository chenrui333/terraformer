// SPDX-License-Identifier: Apache-2.0

package terraformutils

import (
	"strings"
	"testing"

	hclParser "github.com/hashicorp/hcl/hcl/parser"
)

func TestPrintResource(t *testing.T) {
	var resources []Resource
	var nested []map[string]interface{}
	nested = append(nested, mapI("field1", "egg"))
	importResource := prepare("ID1", "type1", map[string]string{
		"type1":                  "ID2",
		"map1.%":                 "1",
		"map1.foo":               "bar",
		"nested.#":               "1",
		"nested.0.map1.#":        "1",
		"nested.0.map1.0.field1": "egg",
		"nested2.#":              "1",
		"nested2.0.field1":       "spam",
		"nested2.0.map2.%":       "1",
		"nested2.0.map2.foo":     "bar",
	}, map[string]interface{}{
		"type1":   "ID2",
		"map1":    mapI("foo", "bar"),
		"nested":  mapI("map1", nested),
		"nested2": map[string]interface{}{"map2": mapI("bar", "foo"), "field1": "egg"},
	})
	resources = append(resources, importResource)
	providerData := map[string]interface{}{}
	output := "hcl"
	data, _ := HclPrintResource(resources, providerData, output, true)

	if strings.Count(string(data), "map1 = ") != 1 {
		t.Errorf("failed to parse data %s", string(data))
	}
	if strings.Count(string(data), "map2 = ") != 1 {
		t.Errorf("failed to parse data %s", string(data))
	}
}

func TestPrintManifestResourceKeepsNestedMapsRenderable(t *testing.T) {
	resource := NewSimpleResource(
		"apiVersion=example.com/v1,kind=Widget,namespace=default,name=sample",
		"example.com/v1/Widget/default/sample",
		"kubernetes_manifest",
		"kubernetes",
		nil,
	)
	resource.Item = map[string]interface{}{
		"manifest": map[string]interface{}{
			"apiVersion": "example.com/v1",
			"kind":       "Widget",
			"metadata": map[string]interface{}{
				"name":      "sample",
				"namespace": "default",
				"labels": map[string]interface{}{
					"app": "sample",
				},
			},
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "sample",
						},
					},
				},
				"versions": []interface{}{
					map[string]interface{}{
						"name": "v1",
						"schema": map[string]interface{}{
							"openAPIV3Schema": map[string]interface{}{
								"type": "object",
							},
						},
					},
					map[string]interface{}{
						"name": "v2",
						"schema": map[string]interface{}{
							"openAPIV3Schema": map[string]interface{}{
								"type": "object",
							},
						},
					},
				},
			},
		},
		"object": map[string]interface{}{
			"status": map[string]interface{}{
				"phase": "Ready",
			},
		},
	}

	data, err := HclPrintResource([]Resource{resource}, map[string]interface{}{}, "hcl", true)
	if err != nil {
		t.Fatalf("HclPrintResource() error = %v", err)
	}
	output := string(data)
	for _, want := range []string{
		"manifest = {",
		"metadata = {",
		"labels = {",
		"template = {",
		"versions = [",
		"schema = {",
		"openAPIV3Schema = {",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output does not contain %q:\n%s", want, output)
		}
	}
	for _, block := range []string{"manifest {", "schema {", "openAPIV3Schema {"} {
		if strings.Contains(output, block) {
			t.Fatalf("%s rendered as a block:\n%s", block, output)
		}
	}
	if strings.Contains(output, "object =") || strings.Contains(output, "status =") {
		t.Fatalf("computed object state was rendered:\n%s", output)
	}
	if _, err := hclParser.Parse(data); err != nil {
		t.Fatalf("generated HCL does not parse: %v\n%s", err, output)
	}
}
