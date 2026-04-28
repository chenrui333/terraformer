// SPDX-License-Identifier: Apache-2.0

package tfcompat

import (
	"encoding/json"
	"strings"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

const (
	TerraformVersion     = "1.9.0"
	UnknownVariableValue = "74D93920-ED26-11E3-AC10-0800200C9A66"
)

type InstanceInfo struct {
	Type string
	Id   string //nolint:revive
}

type ResourceAddress struct {
	Type string
	Name string
}

func (i *InstanceInfo) ResourceAddress() *ResourceAddress {
	name := strings.TrimPrefix(i.Id, i.Type+".")
	if name == i.Id {
		parts := strings.Split(i.Id, ".")
		if len(parts) > 0 {
			name = parts[len(parts)-1]
		}
	}
	return &ResourceAddress{Type: i.Type, Name: name}
}

func (r *ResourceAddress) String() string {
	if r == nil {
		return ""
	}
	return r.Type + "." + r.Name
}

type InstanceState struct {
	ID         string
	Attributes map[string]string
	// TypedAttributes is serialized only for Terraformer plan/import handoff.
	// Final tfstate output writes this payload as TfInstanceV4.Attributes.
	TypedAttributes json.RawMessage `json:"typed_attributes,omitempty"`
	Meta            map[string]interface{}
}

type OutputState struct {
	Sensitive bool
	Type      string
	Value     interface{}
}

func NewInstanceStateShimmedFromValue(state cty.Value, schemaVersion int) *InstanceState {
	attributes := FlatmapValueFromHCL2(state)
	return &InstanceState{
		ID:              attributes["id"],
		Attributes:      attributes,
		TypedAttributes: TryTypedAttributesFromValue(state),
		Meta: map[string]interface{}{
			"schema_version": schemaVersion,
		},
	}
}

func TryTypedAttributesFromValue(state cty.Value) json.RawMessage {
	raw, err := MarshalTypedAttributesFromValue(state)
	if err != nil {
		return nil
	}
	return raw
}

func MarshalTypedAttributesFromValue(state cty.Value) (json.RawMessage, error) {
	if state == cty.NilVal || state.IsNull() {
		return nil, nil
	}
	unmarked, _ := state.UnmarkDeep()
	raw, err := ctyjson.Marshal(unmarked, unmarked.Type())
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func (s *InstanceState) AttrsAsObjectValue(ty cty.Type) (cty.Value, error) {
	if s == nil {
		s = &InstanceState{}
	}
	if s.Attributes == nil {
		s.Attributes = map[string]string{}
	}
	if s.ID != "" {
		s.Attributes["id"] = s.ID
	}
	return HCL2ValueFromFlatmap(s.Attributes, ty)
}
