// SPDX-License-Identifier: Apache-2.0

package terraformutils

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
)

func TestOutputType(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		value    interface{}
		want     string
	}{
		{"explicit type", "list", nil, "list"},
		{"bool value", "", true, "bool"},
		{"int value", "", 42, "number"},
		{"float value", "", 3.14, "number"},
		{"string value", "", "hello", "string"},
		{"nil value defaults to string", "", nil, "string"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := outputType(tc.typeName, tc.value)
			if got != tc.want {
				t.Errorf("outputType(%q, %v) = %v, want %v", tc.typeName, tc.value, got, tc.want)
			}
		})
	}
}

func TestSchemaVersion(t *testing.T) {
	tests := []struct {
		name string
		meta map[string]interface{}
		want uint64
	}{
		{"nil meta", nil, 0},
		{"empty meta", map[string]interface{}{}, 0},
		{"int version", map[string]interface{}{"schema_version": 3}, 3},
		{"int64 version", map[string]interface{}{"schema_version": int64(5)}, 5},
		{"float64 version", map[string]interface{}{"schema_version": float64(2)}, 2},
		{"uint64 version", map[string]interface{}{"schema_version": uint64(7)}, 7},
		{"negative int", map[string]interface{}{"schema_version": -1}, 0},
		{"negative float", map[string]interface{}{"schema_version": float64(-1)}, 0},
		{"string version", map[string]interface{}{"schema_version": "bad"}, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := schemaVersion(tc.meta); got != tc.want {
				t.Errorf("schemaVersion() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestNewTfStateStructure(t *testing.T) {
	r := NewResource("id-1", "name-1", "aws_vpc", "aws",
		map[string]string{"cidr": "10.0.0.0/16"}, nil, nil)

	state := NewTfState([]Resource{r})

	if state.Version != 4 {
		t.Errorf("Version = %d, want 4", state.Version)
	}
	if state.TerraformVersion != tfStateTerraformVersion {
		t.Errorf("TerraformVersion = %q, want %q", state.TerraformVersion, tfStateTerraformVersion)
	}
	if len(state.Resources) != 1 {
		t.Fatalf("Resources count = %d, want 1", len(state.Resources))
	}
	res := state.Resources[0]
	if res.Mode != "managed" {
		t.Errorf("Mode = %q, want %q", res.Mode, "managed")
	}
	if res.Type != "aws_vpc" {
		t.Errorf("Type = %q, want %q", res.Type, "aws_vpc")
	}
}

func TestNewTfStateOutputs(t *testing.T) {
	r := NewResource("id-1", "name-1", "aws_vpc", "aws", nil, nil, nil)
	r.Outputs = map[string]*tfcompat.OutputState{
		"vpc_id": {Type: "string", Value: "vpc-123"},
	}

	state := NewTfState([]Resource{r})

	out, ok := state.Outputs["vpc_id"]
	if !ok {
		t.Fatal("output vpc_id not found")
	}
	if out.Value != "vpc-123" {
		t.Errorf("output value = %v, want %q", out.Value, "vpc-123")
	}
}

func TestNewTfStateSortsResources(t *testing.T) {
	r1 := NewResource("id-2", "b-name", "aws_subnet", "aws", map[string]string{}, nil, nil)
	r2 := NewResource("id-1", "a-name", "aws_vpc", "aws", map[string]string{}, nil, nil)
	r3 := NewResource("id-3", "a-name", "aws_subnet", "aws", map[string]string{}, nil, nil)

	state := NewTfState([]Resource{r1, r2, r3})

	if len(state.Resources) != 3 {
		t.Fatalf("Resources count = %d, want 3", len(state.Resources))
	}
	first := state.Resources[0].Type + "." + state.Resources[0].Name
	last := state.Resources[2].Type + "." + state.Resources[2].Name
	if first >= last {
		t.Errorf("resources not sorted: first=%q last=%q", first, last)
	}
}

func TestNewTfInstanceTypedAttributes(t *testing.T) {
	r := NewResource("id-1", "name-1", "aws_vpc", "aws", nil, nil, nil)
	r.InstanceState.TypedAttributes = json.RawMessage(`{"id":"id-1"}`)

	instance := newTfInstance(r)

	if instance.Attributes == nil {
		t.Fatal("Attributes should be set from TypedAttributes")
	}
	if instance.AttributesFlat != nil {
		t.Error("AttributesFlat should be nil when TypedAttributes is set")
	}
	if instance.SensitiveAttributes == nil {
		t.Error("SensitiveAttributes should be initialized")
	}
}

func TestNewTfInstanceFlatAttributes(t *testing.T) {
	r := NewResource("id-1", "name-1", "aws_vpc", "aws",
		map[string]string{"cidr": "10.0.0.0/16"}, nil, nil)

	instance := newTfInstance(r)

	if instance.AttributesFlat == nil {
		t.Fatal("AttributesFlat should be set")
	}
	if instance.AttributesFlat["id"] != "id-1" {
		t.Errorf("flat id = %q, want %q", instance.AttributesFlat["id"], "id-1")
	}
	if instance.Attributes != nil {
		t.Error("Attributes should be nil when using flat attributes")
	}
}

func TestParseFilterValues(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"single value", "myid", []string{"myid"}},
		{"colon separated", "id1:id2:id3", []string{"id1", "id2", "id3"}},
		{"quoted value with colon", "'project:dataset'", []string{"project:dataset"}},
		{"mixed", "id1:'a:b':id2", []string{"id1", "a:b", "id2"}},
		{"leading colon", ":myid", []string{"myid"}},
		{"empty", "", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseFilterValues(tc.input)
			if len(got) != len(tc.want) {
				t.Errorf("ParseFilterValues(%q) = %v, want %v", tc.input, got, tc.want)
				return
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("ParseFilterValues(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestContainsResource(t *testing.T) {
	r1 := Resource{InstanceInfo: &tfcompat.InstanceInfo{Id: "aws_vpc.a"}}
	r2 := Resource{InstanceInfo: &tfcompat.InstanceInfo{Id: "aws_vpc.b"}}

	list := []Resource{r1}

	if !ContainsResource(list, r1) {
		t.Error("should contain r1")
	}
	if ContainsResource(list, r2) {
		t.Error("should not contain r2")
	}
}

func TestRefreshResourceWorker_PanicRecovery(t *testing.T) {
	r := NewResource("panic-id", "panic-resource", "aws_backup_framework", "aws",
		map[string]string{}, nil, nil)

	input := make(chan *Resource, 1)
	var wg sync.WaitGroup

	wg.Add(1)
	input <- &r
	close(input)

	// nil provider causes panic in Refresh — recovery must catch it
	RefreshResourceWorker(input, &wg, nil)

	wg.Wait()

	if r.InstanceState != nil {
		t.Error("expected InstanceState to be nil after panic recovery")
	}
}
