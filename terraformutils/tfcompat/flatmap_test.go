// Copyright HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfcompat

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestHCL2ValueFromFlatmapList(t *testing.T) {
	tests := []struct {
		name    string
		m       map[string]string
		ty      cty.Type
		wantLen int
	}{
		{
			"two string elements",
			map[string]string{"tags.#": "2", "tags.0": "web", "tags.1": "prod"},
			cty.Object(map[string]cty.Type{"tags": cty.List(cty.String)}),
			2,
		},
		{
			"empty list",
			map[string]string{"tags.#": "0"},
			cty.Object(map[string]cty.Type{"tags": cty.List(cty.String)}),
			0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val, err := HCL2ValueFromFlatmap(tc.m, tc.ty)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			tags := val.GetAttr("tags")
			if !tags.IsKnown() {
				t.Fatal("tags is unknown")
			}
			if tags.IsNull() && tc.wantLen > 0 {
				t.Fatal("tags is null")
			}
			if !tags.IsNull() && tags.LengthInt() != tc.wantLen {
				t.Errorf("tags length = %d, want %d", tags.LengthInt(), tc.wantLen)
			}
		})
	}
}

func TestHCL2ValueFromFlatmapListUnknown(t *testing.T) {
	m := map[string]string{"tags.#": UnknownVariableValue}
	ty := cty.Object(map[string]cty.Type{"tags": cty.List(cty.String)})

	val, err := HCL2ValueFromFlatmap(m, ty)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	tags := val.GetAttr("tags")
	if tags.IsKnown() {
		t.Error("tags should be unknown")
	}
}

func TestHCL2ValueFromFlatmapMap(t *testing.T) {
	tests := []struct {
		name    string
		m       map[string]string
		ty      cty.Type
		wantLen int
	}{
		{
			"two entries",
			map[string]string{"labels.%": "2", "labels.env": "prod", "labels.team": "infra"},
			cty.Object(map[string]cty.Type{"labels": cty.Map(cty.String)}),
			2,
		},
		{
			"empty map",
			map[string]string{"labels.%": "0"},
			cty.Object(map[string]cty.Type{"labels": cty.Map(cty.String)}),
			0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val, err := HCL2ValueFromFlatmap(tc.m, tc.ty)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			labels := val.GetAttr("labels")
			if !labels.IsKnown() {
				t.Fatal("labels is unknown")
			}
			if labels.IsNull() && tc.wantLen > 0 {
				t.Fatal("labels is null")
			}
			if !labels.IsNull() && labels.LengthInt() != tc.wantLen {
				t.Errorf("labels length = %d, want %d", labels.LengthInt(), tc.wantLen)
			}
		})
	}
}

func TestHCL2ValueFromFlatmapMapUnknown(t *testing.T) {
	m := map[string]string{"labels.%": UnknownVariableValue}
	ty := cty.Object(map[string]cty.Type{"labels": cty.Map(cty.String)})

	val, err := HCL2ValueFromFlatmap(m, ty)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	labels := val.GetAttr("labels")
	if labels.IsKnown() {
		t.Error("labels should be unknown")
	}
}

func TestHCL2ValueFromFlatmapSet(t *testing.T) {
	m := map[string]string{"ids.#": "2", "ids.12345": "a", "ids.67890": "b"}
	ty := cty.Object(map[string]cty.Type{"ids": cty.Set(cty.String)})

	val, err := HCL2ValueFromFlatmap(m, ty)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	ids := val.GetAttr("ids")
	if !ids.IsKnown() {
		t.Fatal("ids is unknown")
	}
	if ids.LengthInt() != 2 {
		t.Errorf("ids length = %d, want 2", ids.LengthInt())
	}
}

func TestHCL2ValueFromFlatmapTuple(t *testing.T) {
	m := map[string]string{"pair.#": "2", "pair.0": "hello", "pair.1": "world"}
	ty := cty.Object(map[string]cty.Type{
		"pair": cty.Tuple([]cty.Type{cty.String, cty.String}),
	})

	val, err := HCL2ValueFromFlatmap(m, ty)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	pair := val.GetAttr("pair")
	if !pair.IsKnown() {
		t.Fatal("pair is unknown")
	}
	if pair.LengthInt() != 2 {
		t.Errorf("pair length = %d, want 2", pair.LengthInt())
	}
}

func TestHCL2ValueFromFlatmapTupleUnknown(t *testing.T) {
	m := map[string]string{"pair.#": UnknownVariableValue}
	ty := cty.Object(map[string]cty.Type{
		"pair": cty.Tuple([]cty.Type{cty.String, cty.String}),
	})

	val, err := HCL2ValueFromFlatmap(m, ty)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	pair := val.GetAttr("pair")
	if pair.IsKnown() {
		t.Error("pair should be unknown")
	}
}
