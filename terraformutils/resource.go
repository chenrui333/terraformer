// SPDX-License-Identifier: Apache-2.0

package terraformutils

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/chenrui333/terraformer/terraformutils/providerwrapper"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat/providerproto"
	"github.com/zclconf/go-cty/cty"
)

type Resource struct {
	InstanceInfo      *tfcompat.InstanceInfo
	InstanceState     *tfcompat.InstanceState
	Outputs           map[string]*tfcompat.OutputState `json:",omitempty"`
	ResourceName      string
	Provider          string
	Item              map[string]interface{} `json:",omitempty"`
	IgnoreKeys        []string               `json:",omitempty"`
	AllowEmptyValues  []string               `json:",omitempty"`
	AdditionalFields  map[string]interface{} `json:",omitempty"`
	SlowQueryRequired bool
	DataFiles         map[string][]byte
}

type ApplicableFilter interface {
	IsApplicable(resourceName string) bool
}

type ResourceFilter struct {
	ApplicableFilter
	ServiceName      string
	FieldPath        string
	AcceptableValues []string
}

func (rf *ResourceFilter) Filter(resource Resource) bool {
	if !rf.IsApplicable(strings.TrimPrefix(resource.InstanceInfo.Type, resource.Provider+"_")) {
		return true
	}
	var vals []interface{}
	switch {
	case rf.FieldPath == "id":
		vals = []interface{}{resource.InstanceState.ID}
	case rf.AcceptableValues == nil:
		var hasField = WalkAndCheckField(rf.FieldPath, resource.InstanceState.Attributes)
		if hasField {
			return true
		}
		return WalkAndCheckField(rf.FieldPath, resource.Item)
	default:
		vals = WalkAndGet(rf.FieldPath, resource.InstanceState.Attributes)
		if len(vals) == 0 {
			vals = WalkAndGet(rf.FieldPath, resource.Item)
		}
	}
	for _, val := range vals {
		for _, acceptableValue := range rf.AcceptableValues {
			if val == acceptableValue {
				return true
			}
		}
	}
	return false
}

func (rf *ResourceFilter) IsApplicable(serviceName string) bool {
	return rf.ServiceName == "" || rf.ServiceName == serviceName
}

func (rf *ResourceFilter) isInitial() bool {
	return rf.FieldPath == "id"
}

func NewResource(id, resourceName, resourceType, provider string,
	attributes map[string]string,
	allowEmptyValues []string,
	additionalFields map[string]interface{}) Resource {
	return Resource{
		ResourceName: TfSanitize(resourceName),
		Item:         nil,
		Provider:     provider,
		InstanceState: &tfcompat.InstanceState{
			ID:         id,
			Attributes: attributes,
		},
		InstanceInfo: &tfcompat.InstanceInfo{
			Type: resourceType,
			Id:   fmt.Sprintf("%s.%s", resourceType, TfSanitize(resourceName)),
		},
		AdditionalFields: additionalFields,
		AllowEmptyValues: allowEmptyValues,
	}
}

func NewSimpleResource(id, resourceName, resourceType, provider string, allowEmptyValues []string) Resource {
	return NewResource(
		id,
		resourceName,
		resourceType,
		provider,
		map[string]string{},
		allowEmptyValues,
		map[string]interface{}{},
	)
}

func (r *Resource) Refresh(provider *providerwrapper.ProviderWrapper) {
	var err error
	if r.SlowQueryRequired {
		time.Sleep(200 * time.Millisecond)
	}
	r.InstanceState, err = provider.Refresh(r.InstanceInfo, r.InstanceState)
	if err != nil {
		log.Println(err)
	}
}

func (r Resource) GetIDKey() string {
	if _, exist := r.InstanceState.Attributes["self_link"]; exist {
		return "self_link"
	}
	return "id"
}

func (r *Resource) ParseTFstate(parser Flatmapper, impliedType cty.Type) error {
	attributes, err := parser.Parse(impliedType)
	if err != nil {
		return err
	}
	r.setItem(attributes)
	return nil
}

func (r *Resource) setItem(attributes map[string]interface{}) {
	if attributes == nil {
		attributes = map[string]interface{}{} // ensure HCL can represent empty resource correctly
	}

	// add Additional Fields to resource
	for key, value := range r.AdditionalFields {
		attributes[key] = value
	}

	r.Item = attributes
}

func (r *Resource) ConvertTFstate(provider *providerwrapper.ProviderWrapper) error {
	return r.convertTFstate(provider.GetSchema())
}

func (r *Resource) convertTFstate(schema *providerproto.GetProviderSchemaResponse) error {
	ignoreKeys := []*regexp.Regexp{}
	for _, pattern := range r.IgnoreKeys {
		ignoreKeys = append(ignoreKeys, regexp.MustCompile(pattern))
	}
	allowEmptyValues := []*regexp.Regexp{}
	for _, pattern := range r.AllowEmptyValues {
		if pattern != "" {
			allowEmptyValues = append(allowEmptyValues, regexp.MustCompile(pattern))
		}
	}
	parser := NewFlatmapParser(r.InstanceState.Attributes, ignoreKeys, allowEmptyValues)
	impliedType := schema.ResourceTypes[r.InstanceInfo.Type].Block.ImpliedType()
	err := r.ParseTFstate(parser, impliedType)
	if err == nil && !needsTypedManifestAttributes(r.InstanceInfo.Type, r.Item) {
		return nil
	}

	attributes, typedErr := typedAttributesAsMap(r.InstanceState.TypedAttributes, ignoreKeys)
	if typedErr != nil {
		if err == nil {
			return nil
		}
		return err
	}
	attributes = resourceTypeConfigAttributes(r.InstanceInfo.Type, attributes)
	if err == nil && needsTypedManifestAttributes(r.InstanceInfo.Type, attributes) {
		return nil
	}
	r.setItem(attributes)
	return nil
}

func needsTypedManifestAttributes(resourceType string, attributes map[string]interface{}) bool {
	if resourceType != "kubernetes_manifest" {
		return false
	}
	return !manifestAttributeHasValue(attributes["manifest"])
}

func manifestAttributeHasValue(value interface{}) bool {
	switch value := value.(type) {
	case nil:
		return false
	case map[string]interface{}:
		return len(value) > 0
	default:
		return true
	}
}

func resourceTypeConfigAttributes(resourceType string, attributes map[string]interface{}) map[string]interface{} {
	if resourceType != "kubernetes_manifest" {
		return attributes
	}
	if !manifestAttributeHasValue(attributes["manifest"]) {
		if object, ok := attributes["object"].(map[string]interface{}); ok && len(object) > 0 {
			attributes["manifest"] = object
		}
	}
	delete(attributes, "object")
	return attributes
}

func typedAttributesAsMap(raw json.RawMessage, ignoreKeys []*regexp.Regexp) (map[string]interface{}, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("typed attributes are empty")
	}

	attributes := map[string]interface{}{}
	if err := json.Unmarshal(raw, &attributes); err != nil {
		return nil, err
	}
	for key := range attributes {
		for _, pattern := range ignoreKeys {
			if pattern.MatchString(key) {
				delete(attributes, key)
				break
			}
		}
	}
	return attributes, nil
}

func (r *Resource) ConvertTypedState(provider *providerwrapper.ProviderWrapper) error {
	if r.InstanceState.HasCurrentTypedAttributes() {
		return nil
	}

	schema := provider.GetSchema()
	resourceSchema, ok := schema.ResourceTypes[r.InstanceInfo.Type]
	if !ok {
		return fmt.Errorf("missing schema for resource type %q", r.InstanceInfo.Type)
	}
	value, err := r.InstanceState.AttrsAsObjectValue(resourceSchema.Block.ImpliedType())
	if err != nil {
		return err
	}
	typedAttributes, err := tfcompat.MarshalTypedAttributesFromValue(value)
	if err != nil {
		return err
	}
	r.InstanceState.SetTypedAttributes(typedAttributes)
	return nil
}

func (r *Resource) ServiceName() string {
	return strings.TrimPrefix(r.InstanceInfo.Type, r.Provider+"_")
}
