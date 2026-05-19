// SPDX-License-Identifier: Apache-2.0

package terraformutils

import (
	"bytes"
	"encoding/json"
	"log"
	"runtime"
	"sort"
	"sync"

	"github.com/chenrui333/terraformer/terraformutils/providerwrapper"
)

type BaseResource struct {
	Tags map[string]string `json:"tags,omitempty"`
}

const tfStateTerraformVersion = "1.9.0"

type TfStateV4 struct {
	Version          int                   `json:"version"`
	TerraformVersion string                `json:"terraform_version"`
	Serial           int                   `json:"serial"`
	Lineage          string                `json:"lineage"`
	Outputs          map[string]TfOutputV4 `json:"outputs"`
	Resources        []TfResourceV4        `json:"resources"`
}

type TfOutputV4 struct {
	Value     interface{} `json:"value"`
	Type      interface{} `json:"type"`
	Sensitive bool        `json:"sensitive,omitempty"`
}

type TfResourceV4 struct {
	Mode      string         `json:"mode"`
	Type      string         `json:"type"`
	Name      string         `json:"name"`
	Provider  string         `json:"provider"`
	Instances []TfInstanceV4 `json:"instances"`
}

type TfInstanceV4 struct {
	SchemaVersion       uint64            `json:"schema_version"`
	Attributes          json.RawMessage   `json:"attributes,omitempty"`
	AttributesFlat      map[string]string `json:"attributes_flat,omitempty"`
	SensitiveAttributes *[]interface{}    `json:"sensitive_attributes,omitempty"`
}

func NewTfState(resources []Resource) *TfStateV4 {
	tfstate := &TfStateV4{
		Version:          4,
		TerraformVersion: tfStateTerraformVersion,
		Serial:           1,
		Lineage:          "",
		Outputs:          map[string]TfOutputV4{},
		Resources:        []TfResourceV4{},
	}
	for _, r := range resources {
		for k, v := range r.Outputs {
			tfstate.Outputs[k] = TfOutputV4{
				Value:     v.Value,
				Type:      outputType(v.Type, v.Value),
				Sensitive: v.Sensitive,
			}
		}
	}
	for _, resource := range resources {
		instance := newTfInstance(resource)

		tfstate.Resources = append(tfstate.Resources, TfResourceV4{
			Mode:      "managed",
			Type:      resource.InstanceInfo.Type,
			Name:      resource.ResourceName,
			Provider:  ProviderConfigAddress(resource.Provider),
			Instances: []TfInstanceV4{instance},
		})
	}
	sort.Slice(tfstate.Resources, func(i, j int) bool {
		left := tfstate.Resources[i].Type + "." + tfstate.Resources[i].Name
		right := tfstate.Resources[j].Type + "." + tfstate.Resources[j].Name
		return left < right
	})
	return tfstate
}

func newTfInstance(resource Resource) TfInstanceV4 {
	instance := TfInstanceV4{
		SchemaVersion: schemaVersion(resource.InstanceState.Meta),
	}
	if len(resource.InstanceState.TypedAttributes) > 0 {
		emptySensitiveAttributes := []interface{}{}
		instance.Attributes = resource.InstanceState.TypedAttributes
		instance.SensitiveAttributes = &emptySensitiveAttributes
		return instance
	}

	attributes := map[string]string{}
	for k, v := range resource.InstanceState.Attributes {
		attributes[k] = v
	}
	if _, ok := attributes["id"]; !ok && resource.InstanceState.ID != "" {
		attributes["id"] = resource.InstanceState.ID
	}
	instance.AttributesFlat = attributes
	return instance
}

func PrintTfState(resources []Resource) ([]byte, error) {
	state := NewTfState(resources)
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(state); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func outputType(typeName string, value interface{}) interface{} {
	if typeName != "" {
		return typeName
	}
	switch value.(type) {
	case bool:
		return "bool"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return "number"
	default:
		return "string"
	}
}

func schemaVersion(meta map[string]interface{}) uint64 {
	version, ok := meta["schema_version"]
	if !ok {
		return 0
	}
	switch v := version.(type) {
	case int:
		if v < 0 {
			return 0
		}
		return uint64(v)
	case int64:
		if v < 0 {
			return 0
		}
		return uint64(v)
	case uint64:
		return v
	case float64:
		if v < 0 {
			return 0
		}
		return uint64(v)
	default:
		return 0
	}
}

func RefreshResources(resources []*Resource, provider *providerwrapper.ProviderWrapper, slowProcessingResources [][]*Resource) ([]*Resource, error) {
	refreshedResources := []*Resource{}
	input := make(chan *Resource, len(resources))
	var wg sync.WaitGroup
	poolSize := 15
	for i := range resources {
		wg.Add(1)
		input <- resources[i]
	}
	close(input)

	for i := 0; i < poolSize; i++ {
		go RefreshResourceWorker(input, &wg, provider)
	}

	spInputs := []chan *Resource{}
	for i, resourceGroup := range slowProcessingResources {
		spInputs = append(spInputs, make(chan *Resource, len(resourceGroup)))
		for j := range resourceGroup {
			spInputs[i] <- resourceGroup[j]
		}
		close(spInputs[i])
	}

	for i := 0; i < len(spInputs); i++ {
		wg.Add(len(slowProcessingResources[i]))
		go RefreshResourceWorker(spInputs[i], &wg, provider)
	}

	wg.Wait()
	for _, r := range resources {
		if r.InstanceState != nil && r.InstanceState.ID != "" {
			refreshedResources = append(refreshedResources, r)
		} else {
			log.Printf("ERROR: Unable to refresh resource %s", r.ResourceName)
		}
	}

	for _, resourceGroup := range slowProcessingResources {
		for i := range resourceGroup {
			r := resourceGroup[i]
			if r.InstanceState != nil && r.InstanceState.ID != "" {
				refreshedResources = append(refreshedResources, r)
			} else {
				log.Printf("ERROR: Unable to refresh resource %s", r.ResourceName)
			}
		}
	}
	return refreshedResources, nil
}

func RefreshResourcesByProvider(providersMapping *ProvidersMapping, providerWrapper *providerwrapper.ProviderWrapper) error {
	allResources := providersMapping.ShuffleResources()
	slowProcessingResources := make(map[ProviderGenerator][]*Resource)
	regularResources := []*Resource{}
	for i := range allResources {
		resource := allResources[i]
		if resource.SlowQueryRequired {
			provider := providersMapping.MatchProvider(resource)
			if slowProcessingResources[provider] == nil {
				slowProcessingResources[provider] = []*Resource{}
			}
			slowProcessingResources[provider] = append(slowProcessingResources[provider], resource)
		} else {
			regularResources = append(regularResources, resource)
		}
	}

	var spResourcesList [][]*Resource
	for p := range slowProcessingResources {
		spResourcesList = append(spResourcesList, slowProcessingResources[p])
	}

	refreshedResources, err := RefreshResources(regularResources, providerWrapper, spResourcesList)
	if err != nil {
		return err
	}

	providersMapping.SetResources(refreshedResources)
	return nil
}

func RefreshResourceWorker(input chan *Resource, wg *sync.WaitGroup, provider *providerwrapper.ProviderWrapper) {
	for r := range input {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					buf := make([]byte, 4096)
					n := runtime.Stack(buf, false)
					log.Printf("PANIC: Refresh failed for resource %s: %v\n%s",
						r.InstanceInfo.Id, rec, buf[:n])
					r.InstanceState = nil
				}
				wg.Done()
			}()
			log.Println("Refreshing state...", r.InstanceInfo.Id)
			r.Refresh(provider)
		}()
	}
}

func IgnoreKeys(resourcesTypes []string, p *providerwrapper.ProviderWrapper) map[string][]string {
	readOnlyAttributes, err := p.GetReadOnlyAttributes(resourcesTypes)
	if err != nil {
		log.Println("plugin error 2:", err)
		return map[string][]string{}
	}
	return readOnlyAttributes
}

func ParseFilterValues(value string) []string {
	var values []string

	valueBuffering := true
	wrapped := false
	var valueBuffer []byte
	for i := 0; i < len(value); i++ {
		if value[i] == '\'' {
			wrapped = !wrapped
			continue
		} else if value[i] == ':' {
			if len(valueBuffer) == 0 {
				continue
			} else if valueBuffering && !wrapped {
				values = append(values, string(valueBuffer))
				valueBuffering = false
				valueBuffer = []byte{}
				continue
			}
		}
		valueBuffering = true
		valueBuffer = append(valueBuffer, value[i])
	}
	if len(valueBuffer) > 0 {
		values = append(values, string(valueBuffer))
	}

	return values
}

func FilterCleanup(s *Service, isInitial bool) {
	if len(s.Filter) == 0 {
		return
	}
	var newListOfResources []Resource
	for _, resource := range s.Resources {
		allPredicatesTrue := true
		for _, filter := range s.Filter {
			if filter.isInitial() == isInitial {
				allPredicatesTrue = allPredicatesTrue && filter.Filter(resource)
			}
		}
		if allPredicatesTrue && !ContainsResource(newListOfResources, resource) {
			newListOfResources = append(newListOfResources, resource)
		}
	}
	s.Resources = newListOfResources
}

func ContainsResource(s []Resource, e Resource) bool {
	for _, a := range s {
		if a.InstanceInfo.Id == e.InstanceInfo.Id {
			return true
		}
	}
	return false
}
