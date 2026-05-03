// SPDX-License-Identifier: Apache-2.0

package terraformutils

import (
	"regexp"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
	"github.com/zclconf/go-cty/cty"
)

func TestNestedAttributeFiltering(t *testing.T) {
	attributes := map[string]string{
		"attribute":        "value1",
		"nested.attribute": "value2",
	}

	ignoreKeys := []*regexp.Regexp{
		regexp.MustCompile(`^attribute$`),
	}
	parser := NewFlatmapParser(attributes, ignoreKeys, []*regexp.Regexp{})

	attributesType := cty.Object(map[string]cty.Type{
		"attribute": cty.String,
		"nested": cty.Object(map[string]cty.Type{
			"attribute": cty.String,
		}),
	})

	result, _ := parser.Parse(attributesType)

	if _, ok := result["attribute"]; ok {
		t.Errorf("failed to resolve %v", result)
	}
	if val, ok := result["nested"].(map[string]interface{})["attribute"]; !ok && val != "value2" {
		t.Errorf("failed to resolve %v", result)
	}
}

func TestFromFlatmapList(t *testing.T) {
	attributes := map[string]string{
		"tags.#": "2",
		"tags.0": "web",
		"tags.1": "prod",
	}
	parser := NewFlatmapParser(attributes, nil, nil)
	ty := cty.Object(map[string]cty.Type{
		"tags": cty.List(cty.String),
	})

	result, err := parser.Parse(ty)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	tags, ok := result["tags"].([]interface{})
	if !ok {
		t.Fatalf("tags is not []interface{}, got %T", result["tags"])
	}
	if len(tags) != 2 {
		t.Errorf("tags length = %d, want 2", len(tags))
	}
}

func TestFromFlatmapListEmpty(t *testing.T) {
	attributes := map[string]string{
		"tags.#": "0",
	}
	parser := NewFlatmapParser(attributes, nil, []*regexp.Regexp{regexp.MustCompile("tags")})
	ty := cty.Object(map[string]cty.Type{
		"tags": cty.List(cty.String),
	})

	result, err := parser.Parse(ty)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if result["tags"] != nil {
		tags, ok := result["tags"].([]interface{})
		if !ok {
			t.Fatalf("tags is not []interface{}, got %T", result["tags"])
		}
		if len(tags) != 0 {
			t.Errorf("tags length = %d, want 0", len(tags))
		}
	}
}

func TestFromFlatmapMap(t *testing.T) {
	attributes := map[string]string{
		"labels.%":    "2",
		"labels.env":  "prod",
		"labels.team": "platform",
	}
	parser := NewFlatmapParser(attributes, nil, nil)
	ty := cty.Object(map[string]cty.Type{
		"labels": cty.Map(cty.String),
	})

	result, err := parser.Parse(ty)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	labels, ok := result["labels"].(map[string]interface{})
	if !ok {
		t.Fatalf("labels is not map[string]interface{}, got %T", result["labels"])
	}
	if labels["env"] != "prod" {
		t.Errorf("labels[env] = %v, want %q", labels["env"], "prod")
	}
	if labels["team"] != "platform" {
		t.Errorf("labels[team] = %v, want %q", labels["team"], "platform")
	}
}

func TestFromFlatmapMapOfLists(t *testing.T) {
	attributes := map[string]string{
		"pools.%":       "3",
		"pools.EU.#":    "1",
		"pools.EU.0":    "pool-eu",
		"pools.US.#":    "2",
		"pools.US.0":    "pool-us-a",
		"pools.US.1":    "pool-us-b",
		"pools.X.Foo.#": "1",
		"pools.X.Foo.0": "pool-dotted-key",
	}
	parser := NewFlatmapParser(attributes, nil, nil)
	ty := cty.Object(map[string]cty.Type{
		"pools": cty.Map(cty.List(cty.String)),
	})

	result, err := parser.Parse(ty)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	pools, ok := result["pools"].(map[string]interface{})
	if !ok {
		t.Fatalf("pools is not map[string]interface{}, got %T", result["pools"])
	}
	us, ok := pools["US"].([]interface{})
	if !ok {
		t.Fatalf("pools[US] is not []interface{}, got %T", pools["US"])
	}
	if len(us) != 2 {
		t.Errorf("pools[US] length = %d, want 2", len(us))
	}
	eu, ok := pools["EU"].([]interface{})
	if !ok {
		t.Fatalf("pools[EU] is not []interface{}, got %T", pools["EU"])
	}
	if len(eu) != 1 {
		t.Errorf("pools[EU] length = %d, want 1", len(eu))
	}
	dotted, ok := pools["X.Foo"].([]interface{})
	if !ok {
		t.Fatalf("pools[X.Foo] is not []interface{}, got %T", pools["X.Foo"])
	}
	if len(dotted) != 1 {
		t.Errorf("pools[X.Foo] length = %d, want 1", len(dotted))
	}
	if dotted[0] != "pool-dotted-key" {
		t.Errorf("pools[X.Foo][0] = %v, want %q", dotted[0], "pool-dotted-key")
	}
}

func TestFromFlatmapMapOfObjectsWithDottedUnknownKey(t *testing.T) {
	attributes := map[string]string{
		"headers.%":          "1",
		"headers.X.Foo.name": tfcompat.UnknownVariableValue,
	}
	parser := NewFlatmapParser(attributes, nil, nil)
	ty := cty.Object(map[string]cty.Type{
		"headers": cty.Map(cty.Object(map[string]cty.Type{
			"name": cty.String,
		})),
	})

	result, err := parser.Parse(ty)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	headers, ok := result["headers"].(map[string]interface{})
	if !ok {
		t.Fatalf("headers is not map[string]interface{}, got %T", result["headers"])
	}
	if _, ok := headers["X.Foo.name"]; ok {
		t.Fatal("headers unexpectedly contains nested attribute path as map key X.Foo.name")
	}
	dotted, ok := headers["X.Foo"].(map[string]interface{})
	if !ok {
		t.Fatalf("headers[X.Foo] is not map[string]interface{}, got %T", headers["X.Foo"])
	}
	if dotted["name"] != tfcompat.UnknownVariableValue {
		t.Errorf("headers[X.Foo].name = %v, want %q", dotted["name"], tfcompat.UnknownVariableValue)
	}
}

func TestFromFlatmapMapOfObjectsWithPrefixCollidingDottedKeys(t *testing.T) {
	attributes := map[string]string{
		"headers.%":           "2",
		"headers.X.name":      "short",
		"headers.X.name.name": "long",
	}
	parser := NewFlatmapParser(attributes, nil, nil)
	ty := cty.Object(map[string]cty.Type{
		"headers": cty.Map(cty.Object(map[string]cty.Type{
			"name": cty.String,
		})),
	})

	result, err := parser.Parse(ty)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	headers, ok := result["headers"].(map[string]interface{})
	if !ok {
		t.Fatalf("headers is not map[string]interface{}, got %T", result["headers"])
	}
	if len(headers) != 2 {
		t.Fatalf("headers length = %d, want 2: %#v", len(headers), headers)
	}
	short, ok := headers["X"].(map[string]interface{})
	if !ok {
		t.Fatalf("headers[X] is not map[string]interface{}, got %T", headers["X"])
	}
	if short["name"] != "short" {
		t.Errorf("headers[X].name = %v, want %q", short["name"], "short")
	}
	dotted, ok := headers["X.name"].(map[string]interface{})
	if !ok {
		t.Fatalf("headers[X.name] is not map[string]interface{}, got %T", headers["X.name"])
	}
	if dotted["name"] != "long" {
		t.Errorf("headers[X.name].name = %v, want %q", dotted["name"], "long")
	}
}

func TestFromFlatmapSet(t *testing.T) {
	attributes := map[string]string{
		"ingress.#":               "1",
		"ingress.12345.from_port": "80",
		"ingress.12345.to_port":   "80",
		"ingress.12345.protocol":  "tcp",
	}
	parser := NewFlatmapParser(attributes, nil, nil)
	ty := cty.Object(map[string]cty.Type{
		"ingress": cty.Set(cty.Object(map[string]cty.Type{
			"from_port": cty.String,
			"to_port":   cty.String,
			"protocol":  cty.String,
		})),
	})

	result, err := parser.Parse(ty)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	ingress, ok := result["ingress"].([]interface{})
	if !ok {
		t.Fatalf("ingress is not []interface{}, got %T", result["ingress"])
	}
	if len(ingress) != 1 {
		t.Errorf("ingress length = %d, want 1", len(ingress))
	}
}

func TestFromFlatmapTuple(t *testing.T) {
	attributes := map[string]string{
		"values.#": "2",
		"values.0": "hello",
		"values.1": "world",
	}
	parser := NewFlatmapParser(attributes, nil, nil)
	ty := cty.Object(map[string]cty.Type{
		"values": cty.Tuple([]cty.Type{cty.String, cty.String}),
	})

	result, err := parser.Parse(ty)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	values, ok := result["values"].([]interface{})
	if !ok {
		t.Fatalf("values is not []interface{}, got %T", result["values"])
	}
	if len(values) != 2 {
		t.Errorf("values length = %d, want 2", len(values))
	}
	if values[0] != "hello" {
		t.Errorf("values[0] = %v, want %q", values[0], "hello")
	}
}
