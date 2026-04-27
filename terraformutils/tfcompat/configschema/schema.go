// Copyright HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configschema

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

type StringKind int

const (
	StringPlain StringKind = iota
	StringMarkdown
)

type Schema struct {
	Version int64
	Block   *Block
}

type Block struct {
	Attributes map[string]*Attribute
	BlockTypes map[string]*NestedBlock

	Description     string
	DescriptionKind StringKind
	Deprecated      bool
}

type Attribute struct {
	Type        cty.Type
	NestedType  *Object
	Description string

	DescriptionKind StringKind
	Required        bool
	Optional        bool
	Computed        bool
	Sensitive       bool
	Deprecated      bool
}

type Object struct {
	Attributes map[string]*Attribute
	Nesting    NestingMode
	MinItems   int
	MaxItems   int
}

type NestedBlock struct {
	Block
	Nesting  NestingMode
	MinItems int
	MaxItems int
}

type NestingMode int

const (
	nestingModeInvalid NestingMode = iota
	NestingSingle
	NestingGroup
	NestingList
	NestingSet
	NestingMap
)

func EmptyBlock() *Block {
	return &Block{
		Attributes: map[string]*Attribute{},
		BlockTypes: map[string]*NestedBlock{},
	}
}

func (b *Block) ImpliedType() cty.Type {
	if b == nil {
		return cty.EmptyObject
	}

	attrTypes := make(map[string]cty.Type, len(b.Attributes)+len(b.BlockTypes))
	optionalAttrs := make([]string, 0)
	for name, attr := range b.Attributes {
		if attr == nil {
			continue
		}
		attrTypes[name] = attr.impliedType()
		if !attr.Required {
			optionalAttrs = append(optionalAttrs, name)
		}
	}
	for name, block := range b.BlockTypes {
		if block == nil {
			continue
		}
		attrTypes[name] = block.impliedType()
		if block.MinItems == 0 {
			optionalAttrs = append(optionalAttrs, name)
		}
	}
	if len(optionalAttrs) > 0 {
		return cty.ObjectWithOptionalAttrs(attrTypes, optionalAttrs)
	}
	return cty.Object(attrTypes)
}

func (o *Object) ImpliedType() cty.Type {
	if o == nil {
		return cty.EmptyObject
	}

	attrTypes := make(map[string]cty.Type, len(o.Attributes))
	optionalAttrs := make([]string, 0)
	for name, attr := range o.Attributes {
		if attr == nil {
			continue
		}
		attrTypes[name] = attr.impliedType()
		if attr.Optional {
			optionalAttrs = append(optionalAttrs, name)
		}
	}
	var ret cty.Type
	if len(optionalAttrs) > 0 {
		ret = cty.ObjectWithOptionalAttrs(attrTypes, optionalAttrs)
	} else {
		ret = cty.Object(attrTypes)
	}
	switch o.Nesting {
	case NestingSingle:
		return ret
	case NestingList:
		return cty.List(ret)
	case NestingMap:
		return cty.Map(ret)
	case NestingSet:
		return cty.Set(ret)
	default:
		return ret
	}
}

func (a *Attribute) impliedType() cty.Type {
	if a == nil {
		return cty.DynamicPseudoType
	}
	switch {
	case a.NestedType != nil:
		return a.NestedType.ImpliedType()
	case a.Type == cty.NilType:
		return cty.DynamicPseudoType
	default:
		return a.Type
	}
}

func (b *NestedBlock) impliedType() cty.Type {
	if b == nil {
		return cty.DynamicPseudoType
	}
	blockType := b.ImpliedType()
	switch b.Nesting {
	case NestingSingle, NestingGroup:
		return blockType
	case NestingList:
		return cty.List(blockType)
	case NestingSet:
		return cty.Set(blockType)
	case NestingMap:
		return cty.Map(blockType)
	default:
		return cty.DynamicPseudoType
	}
}

func (b *Block) EmptyValue() cty.Value {
	if b == nil {
		return cty.EmptyObjectVal
	}
	vals := make(map[string]cty.Value, len(b.Attributes)+len(b.BlockTypes))
	for name, attr := range b.Attributes {
		vals[name] = attr.EmptyValue()
	}
	for name, block := range b.BlockTypes {
		vals[name] = block.EmptyValue()
	}
	return cty.ObjectVal(vals)
}

func (a *Attribute) EmptyValue() cty.Value {
	if a == nil {
		return cty.NullVal(cty.DynamicPseudoType)
	}
	if a.NestedType != nil {
		return cty.NullVal(a.NestedType.ImpliedType())
	}
	if a.Type == cty.NilType {
		return cty.NullVal(cty.DynamicPseudoType)
	}
	return cty.NullVal(a.Type)
}

func (b *NestedBlock) EmptyValue() cty.Value {
	if b == nil {
		return cty.NullVal(cty.DynamicPseudoType)
	}
	switch b.Nesting {
	case NestingSingle:
		return cty.NullVal(b.ImpliedType())
	case NestingGroup:
		return b.Block.EmptyValue()
	case NestingList:
		return cty.ListValEmpty(b.ImpliedType())
	case NestingMap:
		return cty.MapValEmpty(b.ImpliedType())
	case NestingSet:
		return cty.SetValEmpty(b.ImpliedType())
	default:
		return cty.NullVal(cty.DynamicPseudoType)
	}
}

func (b *Block) CoerceValue(in cty.Value) (cty.Value, error) {
	var path cty.Path
	return b.coerceValue(in, path)
}

func (b *Block) coerceValue(in cty.Value, path cty.Path) (cty.Value, error) {
	if b == nil {
		b = EmptyBlock()
	}
	switch {
	case in.IsNull():
		return cty.NullVal(b.ImpliedType()), nil
	case !in.IsKnown():
		return cty.UnknownVal(b.ImpliedType()), nil
	}

	ty := in.Type()
	if !ty.IsObjectType() {
		return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("an object is required")
	}

	for name := range ty.AttributeTypes() {
		if _, defined := b.Attributes[name]; defined {
			continue
		}
		if _, defined := b.BlockTypes[name]; defined {
			continue
		}
		return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("unexpected attribute %q", name)
	}

	attrs := make(map[string]cty.Value, len(b.Attributes)+len(b.BlockTypes))
	for name, attr := range b.Attributes {
		if !ty.HasAttribute(name) && attr.Required {
			return cty.UnknownVal(b.ImpliedType()), append(path, cty.GetAttrStep{Name: name}).NewErrorf("attribute is required")
		}
		val, err := attr.coerceValue(attributeValue(in, ty, name, attr), append(path, cty.GetAttrStep{Name: name}))
		if err != nil {
			return cty.UnknownVal(b.ImpliedType()), err
		}
		attrs[name] = val
	}
	for typeName, block := range b.BlockTypes {
		val, err := block.coerceValue(in, ty, typeName, append(path, cty.GetAttrStep{Name: typeName}))
		if err != nil {
			return cty.UnknownVal(b.ImpliedType()), err
		}
		attrs[typeName] = val
	}
	return cty.ObjectVal(attrs), nil
}

func attributeValue(in cty.Value, ty cty.Type, name string, attr *Attribute) cty.Value {
	if ty.HasAttribute(name) {
		return in.GetAttr(name)
	}
	if attr.Computed || attr.Optional {
		return attr.EmptyValue()
	}
	return cty.DynamicVal
}

func (a *Attribute) coerceValue(in cty.Value, path cty.Path) (cty.Value, error) {
	if a == nil {
		return cty.DynamicVal, nil
	}
	if a.NestedType != nil {
		return convert.Convert(in, a.NestedType.ImpliedType())
	}
	if a.Type == cty.NilType {
		return in, nil
	}
	if !in.IsKnown() && in.Type() == cty.DynamicPseudoType {
		return cty.UnknownVal(a.Type), nil
	}
	val, err := convert.Convert(in, a.Type)
	if err != nil {
		return cty.UnknownVal(a.Type), path.NewError(err)
	}
	return val, nil
}

func (b *NestedBlock) coerceValue(in cty.Value, ty cty.Type, typeName string, path cty.Path) (cty.Value, error) {
	if b == nil {
		return cty.NullVal(cty.DynamicPseudoType), nil
	}
	if !ty.HasAttribute(typeName) {
		return b.EmptyValue(), nil
	}

	coll := in.GetAttr(typeName)
	switch b.Nesting {
	case NestingSingle, NestingGroup:
		return b.Block.coerceValue(coll, path)
	case NestingList:
		return b.coerceList(coll, path)
	case NestingSet:
		return b.coerceSet(coll, path)
	case NestingMap:
		return b.coerceMap(coll, path)
	default:
		return cty.UnknownVal(b.impliedType()), fmt.Errorf("unsupported nesting mode %#v", b.Nesting)
	}
}

func (b *NestedBlock) coerceList(coll cty.Value, path cty.Path) (cty.Value, error) {
	blockType := b.ImpliedType()
	switch {
	case coll.IsNull():
		return cty.NullVal(cty.List(blockType)), nil
	case !coll.IsKnown():
		return cty.UnknownVal(cty.List(blockType)), nil
	case !coll.CanIterateElements():
		return cty.UnknownVal(cty.List(blockType)), path.NewErrorf("must be a list")
	case coll.LengthInt() == 0:
		return cty.ListValEmpty(blockType), nil
	}
	elems := make([]cty.Value, 0, coll.LengthInt())
	for it := coll.ElementIterator(); it.Next(); {
		idx, val := it.Element()
		coerced, err := b.Block.coerceValue(val, append(path, cty.IndexStep{Key: idx}))
		if err != nil {
			return cty.UnknownVal(cty.List(blockType)), err
		}
		elems = append(elems, coerced)
	}
	return cty.ListVal(elems), nil
}

func (b *NestedBlock) coerceSet(coll cty.Value, path cty.Path) (cty.Value, error) {
	blockType := b.ImpliedType()
	switch {
	case coll.IsNull():
		return cty.NullVal(cty.Set(blockType)), nil
	case !coll.IsKnown():
		return cty.UnknownVal(cty.Set(blockType)), nil
	case !coll.CanIterateElements():
		return cty.UnknownVal(cty.Set(blockType)), path.NewErrorf("must be a set")
	case coll.LengthInt() == 0:
		return cty.SetValEmpty(blockType), nil
	}
	elems := make([]cty.Value, 0, coll.LengthInt())
	for it := coll.ElementIterator(); it.Next(); {
		idx, val := it.Element()
		coerced, err := b.Block.coerceValue(val, append(path, cty.IndexStep{Key: idx}))
		if err != nil {
			return cty.UnknownVal(cty.Set(blockType)), err
		}
		elems = append(elems, coerced)
	}
	return cty.SetVal(elems), nil
}

func (b *NestedBlock) coerceMap(coll cty.Value, path cty.Path) (cty.Value, error) {
	blockType := b.ImpliedType()
	switch {
	case coll.IsNull():
		return cty.NullVal(cty.Map(blockType)), nil
	case !coll.IsKnown():
		return cty.UnknownVal(cty.Map(blockType)), nil
	case !coll.CanIterateElements():
		return cty.UnknownVal(cty.Map(blockType)), path.NewErrorf("must be a map")
	case coll.LengthInt() == 0:
		return cty.MapValEmpty(blockType), nil
	}
	elems := make(map[string]cty.Value, coll.LengthInt())
	for it := coll.ElementIterator(); it.Next(); {
		key, val := it.Element()
		if key.Type() != cty.String || key.IsNull() || !key.IsKnown() {
			return cty.UnknownVal(cty.Map(blockType)), path.NewErrorf("must be a map")
		}
		coerced, err := b.Block.coerceValue(val, append(path, cty.IndexStep{Key: key}))
		if err != nil {
			return cty.UnknownVal(cty.Map(blockType)), err
		}
		elems[key.AsString()] = coerced
	}
	return cty.MapVal(elems), nil
}
