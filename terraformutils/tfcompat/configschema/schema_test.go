// Copyright HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configschema

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestNestedBlockImpliedTypeDynamicCollections(t *testing.T) {
	tests := map[string]struct {
		block *NestedBlock
		want  cty.Type
	}{
		"list with dynamic attributes": {
			block: dynamicNestedBlock(NestingList),
			want:  cty.DynamicPseudoType,
		},
		"map with dynamic attributes": {
			block: dynamicNestedBlock(NestingMap),
			want:  cty.DynamicPseudoType,
		},
		"list with static attributes": {
			block: staticNestedBlock(NestingList),
			want: cty.List(cty.Object(map[string]cty.Type{
				"bar": cty.String,
			})),
		},
		"map with static attributes": {
			block: staticNestedBlock(NestingMap),
			want: cty.Map(cty.Object(map[string]cty.Type{
				"bar": cty.String,
			})),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := test.block.impliedType()
			if !got.Equals(test.want) {
				t.Fatalf("wrong implied type\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}

func TestNestedBlockEmptyValueDynamicCollections(t *testing.T) {
	schema := &Block{
		BlockTypes: map[string]*NestedBlock{
			"list_dynamic": dynamicNestedBlock(NestingList),
			"map_dynamic":  dynamicNestedBlock(NestingMap),
			"list_static":  staticNestedBlock(NestingList),
			"map_static":   staticNestedBlock(NestingMap),
		},
	}
	want := cty.ObjectVal(map[string]cty.Value{
		"list_dynamic": cty.EmptyTupleVal,
		"map_dynamic":  cty.EmptyObjectVal,
		"list_static": cty.ListValEmpty(cty.Object(map[string]cty.Type{
			"bar": cty.String,
		})),
		"map_static": cty.MapValEmpty(cty.Object(map[string]cty.Type{
			"bar": cty.String,
		})),
	})

	got := schema.EmptyValue()
	if !got.RawEquals(want) {
		t.Fatalf("wrong empty value\ngot:  %s\nwant: %s", got.GoString(), want.GoString())
	}
}

func TestCoerceValueDynamicMapUsesObjectShape(t *testing.T) {
	schema := &Block{
		BlockTypes: map[string]*NestedBlock{
			"foo": dynamicNestedBlock(NestingMap),
		},
	}
	input := cty.ObjectVal(map[string]cty.Value{
		"foo": cty.MapVal(map[string]cty.Value{
			"a": cty.ObjectVal(map[string]cty.Value{
				"bar": cty.StringVal("beep"),
			}),
			"b": cty.ObjectVal(map[string]cty.Value{
				"bar": cty.StringVal("boop"),
			}),
		}),
	})
	want := cty.ObjectVal(map[string]cty.Value{
		"foo": cty.ObjectVal(map[string]cty.Value{
			"a": cty.ObjectVal(map[string]cty.Value{
				"bar": cty.StringVal("beep"),
				"baz": cty.NullVal(cty.DynamicPseudoType),
			}),
			"b": cty.ObjectVal(map[string]cty.Value{
				"bar": cty.StringVal("boop"),
				"baz": cty.NullVal(cty.DynamicPseudoType),
			}),
		}),
	})

	got, err := schema.CoerceValue(input)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !got.RawEquals(want) {
		t.Fatalf("wrong coerced value\ngot:  %s\nwant: %s", got.GoString(), want.GoString())
	}
}

func TestCoerceValueDynamicListUsesTupleShape(t *testing.T) {
	schema := &Block{
		BlockTypes: map[string]*NestedBlock{
			"foo": dynamicNestedBlock(NestingList),
		},
	}
	input := cty.ObjectVal(map[string]cty.Value{
		"foo": cty.TupleVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"bar": cty.StringVal("beep"),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"bar": cty.StringVal("boop"),
				"baz": cty.NumberIntVal(8),
			}),
		}),
	})
	want := cty.ObjectVal(map[string]cty.Value{
		"foo": cty.TupleVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"bar": cty.StringVal("beep"),
				"baz": cty.NullVal(cty.DynamicPseudoType),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"bar": cty.StringVal("boop"),
				"baz": cty.NumberIntVal(8),
			}),
		}),
	})

	got, err := schema.CoerceValue(input)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !got.RawEquals(want) {
		t.Fatalf("wrong coerced value\ngot:  %s\nwant: %s", got.GoString(), want.GoString())
	}
}

func dynamicNestedBlock(nesting NestingMode) *NestedBlock {
	return &NestedBlock{
		Nesting: nesting,
		Block: Block{
			Attributes: map[string]*Attribute{
				"bar": {
					Type:     cty.String,
					Optional: true,
					Computed: true,
				},
				"baz": {
					Type:     cty.DynamicPseudoType,
					Optional: true,
					Computed: true,
				},
			},
		},
	}
}

func staticNestedBlock(nesting NestingMode) *NestedBlock {
	return &NestedBlock{
		Nesting: nesting,
		Block: Block{
			Attributes: map[string]*Attribute{
				"bar": {
					Type:     cty.String,
					Required: true,
				},
			},
		},
	}
}
