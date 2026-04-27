// Copyright 2026 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tfcompat

import (
	"strings"

	"github.com/hashicorp/terraform/configs/hcl2shim"
	"github.com/zclconf/go-cty/cty"
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
	Meta       map[string]interface{}
}

type OutputState struct {
	Sensitive bool
	Type      string
	Value     interface{}
}

func NewInstanceStateShimmedFromValue(state cty.Value, schemaVersion int) *InstanceState {
	attributes := hcl2shim.FlatmapValueFromHCL2(state)
	return &InstanceState{
		ID:         attributes["id"],
		Attributes: attributes,
		Meta: map[string]interface{}{
			"schema_version": schemaVersion,
		},
	}
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
	return hcl2shim.HCL2ValueFromFlatmap(s.Attributes, ty)
}
