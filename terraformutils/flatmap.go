// SPDX-License-Identifier: Apache-2.0

package terraformutils

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
	"github.com/zclconf/go-cty/cty"
)

type Flatmapper interface {
	Parse(ty cty.Type) (map[string]interface{}, error)
}

type FlatmapParser struct {
	Flatmapper
	attributes       map[string]string
	ignoreKeys       []*regexp.Regexp
	allowEmptyValues []*regexp.Regexp
}

func NewFlatmapParser(attributes map[string]string, ignoreKeys []*regexp.Regexp, allowEmptyValues []*regexp.Regexp) *FlatmapParser {
	return &FlatmapParser{
		attributes:       attributes,
		ignoreKeys:       ignoreKeys,
		allowEmptyValues: allowEmptyValues,
	}
}

// FromFlatmap converts a map compatible with what would be produced
// by the "flatmap" package to a map[string]interface{} object type.
//
// The intended result type must be provided in order to guide how the
// map contents are decoded. This must be an object type or this function
// will panic.
//
// Flatmap values can only represent maps when they are of primitive types,
// so the given type must not have any maps of complex types or the result
// is undefined.
//
// The result may contain null values if the given map does not contain keys
// for all of the different key paths implied by the given type.
func (p *FlatmapParser) Parse(ty cty.Type) (map[string]interface{}, error) {
	if p.attributes == nil {
		return nil, nil
	}
	if !ty.IsObjectType() {
		return nil, fmt.Errorf("FlatmapParser#Parse called on %#v", ty)
	}
	return p.fromFlatmapObject("", ty.AttributeTypes())
}

func (p *FlatmapParser) fromFlatmapValue(key string, ty cty.Type) (interface{}, error) {
	switch {
	case ty.IsPrimitiveType():
		return p.fromFlatmapPrimitive(key)
	case ty.IsObjectType():
		return p.fromFlatmapObject(key+".", ty.AttributeTypes())
	case ty.IsTupleType():
		return p.fromFlatmapTuple(key+".", ty.TupleElementTypes())
	case ty.IsMapType():
		return p.fromFlatmapMap(key+".", ty.ElementType())
	case ty.IsListType():
		return p.fromFlatmapList(key+".", ty.ElementType())
	case ty.IsSetType():
		return p.fromFlatmapSet(key+".", ty.ElementType())
	default:
		return nil, fmt.Errorf("cannot decode %s from flatmap", ty.FriendlyName())
	}
}

func (p *FlatmapParser) fromFlatmapPrimitive(key string) (interface{}, error) {
	value, ok := p.attributes[key]
	if !ok {
		return nil, nil
	}
	return value, nil
}

func (p *FlatmapParser) fromFlatmapObject(prefix string, tys map[string]cty.Type) (map[string]interface{}, error) {
	values := make(map[string]interface{})
	for name, ty := range tys {
		inAttributes := false
		attributeName := ""
		for k := range p.attributes {
			if k == prefix+name {
				attributeName = k
				inAttributes = true
				break
			}
			if k == name {
				attributeName = k
				inAttributes = true
				break
			}

			if strings.HasPrefix(k, prefix+name+".") {
				attributeName = k
				inAttributes = true
				break
			}
			lastAttribute := (prefix + name)[len(prefix):]
			if lastAttribute == k {
				attributeName = k
				inAttributes = true
				break
			}
		}

		if _, exist := p.attributes[prefix+name+".#"]; exist {
			attributeName = prefix + name + ".#"
			inAttributes = true
		}

		if _, exist := p.attributes[prefix+name+".%"]; exist {
			attributeName = prefix + name + ".%"
			inAttributes = true
		}

		if !inAttributes {
			continue
		}
		if p.isAttributeIgnored(prefix + name) {
			continue
		}
		value, err := p.fromFlatmapValue(prefix+name, ty)
		if err != nil {
			return nil, err
		}
		if p.isValueAllowed(value, attributeName) {
			values[name] = value
		}
	}
	if len(values) == 0 {
		return nil, nil
	}
	return values, nil
}

func (p *FlatmapParser) fromFlatmapTuple(prefix string, tys []cty.Type) ([]interface{}, error) {
	// if the container is unknown, there is no count string
	listName := strings.TrimRight(prefix, ".")
	if p.attributes[listName] == tfcompat.UnknownVariableValue {
		return nil, nil
	}

	countStr, exists := p.attributes[prefix+"#"]
	if !exists {
		return nil, nil
	}
	if countStr == tfcompat.UnknownVariableValue {
		return nil, nil
	}

	count, err := strconv.Atoi(countStr)
	if err != nil {
		return nil, fmt.Errorf("invalid count value for %q in state: %w", prefix, err)
	}
	if count != len(tys) {
		return nil, fmt.Errorf("wrong number of values for %q in state: got %d, but need %d", prefix, count, len(tys))
	}

	var values []interface{}
	for i, ty := range tys {
		key := prefix + strconv.Itoa(i)
		value, err := p.fromFlatmapValue(key, ty)
		if err != nil {
			return nil, err
		}
		if p.isValueAllowed(value, prefix) {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return nil, nil
	}
	return values, nil
}

func (p *FlatmapParser) fromFlatmapMap(prefix string, ty cty.Type) (map[string]interface{}, error) {
	// if the container is unknown, there is no count string
	listName := strings.TrimRight(prefix, ".")
	if p.attributes[listName] == tfcompat.UnknownVariableValue {
		return nil, nil
	}

	// We actually don't really care about the "count" of a map for our
	// purposes here, but we do need to check if it _exists_ in order to
	// recognize the difference between null (not set at all) and empty.
	strCount, exists := p.attributes[prefix+"%"]
	if !exists {
		return nil, nil
	}
	if strCount == tfcompat.UnknownVariableValue {
		return nil, nil
	}

	values := make(map[string]interface{})
	for fullKey := range p.attributes {
		if !strings.HasPrefix(fullKey, prefix) {
			continue
		}

		// The flatmap format doesn't allow us to distinguish between keys
		// that contain periods and nested objects, so by convention a
		// map is only ever of primitive type in flatmap, and we just assume
		// that the remainder of the raw key (dots and all) is the key we
		// want in the result value.
		key := fullKey[len(prefix):]
		if key == "%" {
			// Ignore the "count" key
			continue
		}
		if p.isAttributeIgnored(fullKey) {
			continue
		}

		valueKey := fullKey
		if !ty.IsPrimitiveType() {
			key, valueKey = p.fromFlatmapMapElementKey(prefix, key, ty)
		}
		if _, exists := values[key]; exists {
			continue
		}

		value, err := p.fromFlatmapValue(valueKey, ty)
		if err != nil {
			return nil, err
		}
		if p.isValueAllowed(value, prefix) {
			values[key] = value
		}
	}
	if len(values) == 0 {
		return nil, nil
	}
	return values, nil
}

func (p *FlatmapParser) fromFlatmapMapElementKey(prefix, key string, ty cty.Type) (string, string) {
	candidates := flatmapMapKeyCandidates(key)
	if ty.IsObjectType() {
		candidates = flatmapObjectMapKeyCandidates(key)
	}
	for _, candidate := range candidates {
		valueKey := prefix + candidate
		if !flatmapMapElementPathMatches(key, candidate, ty) {
			continue
		}
		if p.flatmapValueExists(valueKey, ty) {
			return candidate, valueKey
		}
	}

	if dot := strings.IndexByte(key, '.'); dot != -1 {
		key = key[:dot]
	}
	return key, prefix + key
}

func flatmapMapKeyCandidates(key string) []string {
	candidates := []string{key}
	for dot := strings.LastIndexByte(key, '.'); dot != -1; dot = strings.LastIndexByte(key[:dot], '.') {
		if dot == 0 {
			continue
		}
		candidates = append(candidates, key[:dot])
	}
	return candidates
}

func flatmapObjectMapKeyCandidates(key string) []string {
	var candidates []string
	for dot := strings.IndexByte(key, '.'); dot != -1; {
		candidates = append(candidates, key[:dot])
		next := strings.IndexByte(key[dot+1:], '.')
		if next == -1 {
			break
		}
		dot += next + 1
	}
	candidates = append(candidates, key)
	return candidates
}

func flatmapMapElementPathMatches(key, candidate string, ty cty.Type) bool {
	if ty.IsObjectType() {
		return flatmapMapObjectElementPathMatches(key, candidate, ty.AttributeTypes())
	}
	if !strings.HasPrefix(key, candidate+".") {
		return false
	}
	return flatmapMapValuePathMatches(key[len(candidate)+1:], ty)
}

func flatmapMapObjectElementPathMatches(key, candidate string, tys map[string]cty.Type) bool {
	if !strings.HasPrefix(key, candidate+".") {
		return false
	}
	return flatmapObjectAttributePathMatches(key[len(candidate)+1:], tys)
}

func flatmapObjectAttributePathMatches(path string, tys map[string]cty.Type) bool {
	for name, ty := range tys {
		if path == name {
			return true
		}
		if strings.HasPrefix(path, name+".") {
			return flatmapMapValuePathMatches(path[len(name)+1:], ty)
		}
	}
	return false
}

func flatmapMapValuePathMatches(path string, ty cty.Type) bool {
	switch {
	case ty.IsPrimitiveType(), ty == cty.DynamicPseudoType:
		return path == ""
	case ty.IsObjectType():
		return flatmapObjectAttributePathMatches(path, ty.AttributeTypes())
	case ty.IsTupleType():
		return flatmapTuplePathMatches(path, ty.TupleElementTypes())
	case ty.IsListType():
		return flatmapSeqPathMatches(path, ty.ElementType(), true)
	case ty.IsSetType():
		return flatmapSeqPathMatches(path, ty.ElementType(), false)
	case ty.IsMapType():
		return flatmapNestedMapPathMatches(path, ty.ElementType())
	default:
		return false
	}
}

func flatmapTuplePathMatches(path string, tys []cty.Type) bool {
	if path == "#" {
		return true
	}
	segment, rest, hasRest := strings.Cut(path, ".")
	index, err := strconv.Atoi(segment)
	if err != nil || index < 0 || index >= len(tys) {
		return false
	}
	if !hasRest {
		return tys[index].IsPrimitiveType() || tys[index] == cty.DynamicPseudoType
	}
	return flatmapMapValuePathMatches(rest, tys[index])
}

func flatmapSeqPathMatches(path string, ty cty.Type, requireIndex bool) bool {
	if path == "#" {
		return true
	}
	segment, rest, hasRest := strings.Cut(path, ".")
	if segment == "" {
		return false
	}
	if requireIndex {
		if _, err := strconv.Atoi(segment); err != nil {
			return false
		}
	}
	if !hasRest {
		return ty.IsPrimitiveType() || ty == cty.DynamicPseudoType
	}
	return flatmapMapValuePathMatches(rest, ty)
}

func flatmapNestedMapPathMatches(path string, ty cty.Type) bool {
	if path == "%" {
		return true
	}
	if path == "" {
		return false
	}
	if ty.IsPrimitiveType() || ty == cty.DynamicPseudoType {
		return true
	}
	_, rest, hasRest := strings.Cut(path, ".")
	if !hasRest {
		return false
	}
	return flatmapMapValuePathMatches(rest, ty)
}

func (p *FlatmapParser) flatmapValueExists(key string, ty cty.Type) bool {
	switch {
	case ty.IsPrimitiveType():
		if p.attributes[key] == tfcompat.UnknownVariableValue {
			return true
		}
		_, exists := p.attributes[key]
		return exists
	case ty.IsObjectType():
		return p.flatmapObjectValueExists(key+".", ty.AttributeTypes())
	case ty.IsTupleType(), ty.IsListType(), ty.IsSetType():
		if p.attributes[key] == tfcompat.UnknownVariableValue {
			return true
		}
		_, exists := p.attributes[key+".#"]
		return exists
	case ty.IsMapType():
		if p.attributes[key] == tfcompat.UnknownVariableValue {
			return true
		}
		_, exists := p.attributes[key+".%"]
		return exists
	default:
		return false
	}
}

func (p *FlatmapParser) flatmapObjectValueExists(prefix string, tys map[string]cty.Type) bool {
	for name, ty := range tys {
		key := prefix + name
		if p.flatmapValueExists(key, ty) {
			return true
		}
		for attributeKey := range p.attributes {
			if strings.HasPrefix(attributeKey, key+".") {
				return true
			}
		}
	}
	return false
}

func (p *FlatmapParser) fromFlatmapList(prefix string, ty cty.Type) ([]interface{}, error) {
	// if the container is unknown, there is no count string
	listName := strings.TrimRight(prefix, ".")
	if p.attributes[listName] == tfcompat.UnknownVariableValue {
		return nil, nil
	}

	countStr, exists := p.attributes[prefix+"#"]
	if !exists {
		return nil, nil
	}
	if countStr == tfcompat.UnknownVariableValue {
		return nil, nil
	}

	count, err := strconv.Atoi(countStr)
	if err != nil {
		return nil, fmt.Errorf("invalid count value for %q in state: %w", prefix, err)
	}

	if count == 0 {
		return nil, nil
	}

	var values []interface{}
	for i := 0; i < count; i++ {
		key := prefix + strconv.Itoa(i)

		if p.isAttributeIgnored(key) {
			continue
		}

		value, err := p.fromFlatmapValue(key, ty)
		if err != nil {
			return nil, err
		}
		if p.isValueAllowed(value, prefix) {
			values = append(values, value)
		}
	}
	return values, nil
}

func (p *FlatmapParser) fromFlatmapSet(prefix string, ty cty.Type) ([]interface{}, error) {
	// if the container is unknown, there is no count string
	listName := strings.TrimRight(prefix, ".")
	if p.attributes[listName] == tfcompat.UnknownVariableValue {
		return nil, nil
	}

	strCount, exists := p.attributes[prefix+"#"]
	if !exists {
		return nil, nil
	}
	if strCount == tfcompat.UnknownVariableValue {
		return nil, nil
	}

	// Keep track of keys we've seen, se we don't add the same set value
	// multiple times. The cty.Set will normally de-duplicate values, but we may
	// have unknown values that would not show as equivalent.
	seen := map[string]bool{}

	var values []interface{}
	for fullKey := range p.attributes {
		if !strings.HasPrefix(fullKey, prefix) {
			continue
		}

		subKey := fullKey[len(prefix):]
		if subKey == "#" {
			// Ignore the "count" key
			continue
		}

		key := fullKey

		if p.isAttributeIgnored(fullKey) {
			continue
		}

		if dot := strings.IndexByte(subKey, '.'); dot != -1 {
			key = fullKey[:dot+len(prefix)]
		}

		if seen[key] {
			continue
		}
		seen[key] = true

		// The flatmap format doesn't allow us to distinguish between keys
		// that contain periods and nested objects, so by convention a
		// map is only ever of primitive type in flatmap, and we just assume
		// that the remainder of the raw key (dots and all) is the key we
		// want in the result value.

		value, err := p.fromFlatmapValue(key, ty)
		if err != nil {
			return nil, err
		}
		if p.isValueAllowed(value, prefix) {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return nil, nil
	}
	return values, nil
}

func (p *FlatmapParser) isAttributeIgnored(name string) bool {
	ignored := false
	for _, pattern := range p.ignoreKeys {
		if pattern.MatchString(name) {
			ignored = true
			break
		}
	}
	return ignored
}

func (p *FlatmapParser) isValueAllowed(value interface{}, prefix string) bool {
	if !reflect.ValueOf(value).IsValid() {
		return false
	}
	switch reflect.ValueOf(value).Kind() {
	case reflect.Slice:
		if reflect.ValueOf(value).Len() == 0 {
			return false
		}

		for i := 0; i < reflect.ValueOf(value).Len(); i++ {
			if !reflect.ValueOf(value).Index(i).IsZero() {
				return true
			}
		}
	case reflect.Map:
		if reflect.ValueOf(value).Len() == 0 {
			return false
		}
	}
	if !reflect.ValueOf(value).IsZero() {
		return true
	}

	allowed := false
	for _, pattern := range p.allowEmptyValues {
		if pattern.MatchString(prefix) {
			allowed = true
			break
		}
	}
	return allowed
}
