// SPDX-License-Identifier: Apache-2.0

package terraformutils

import (
	"fmt"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils/providerwrapper"
)

type ServiceGenerator interface {
	InitResources() error
	GetResources() []Resource
	SetResources(resources []Resource)
	ParseFilter(rawFilter string) ([]ResourceFilter, error)
	ParseFilters(rawFilters []string) error
	PostConvertHook() error
	GetArgs() map[string]interface{}
	SetArgs(args map[string]interface{})
	SetName(name string)
	SetVerbose(bool)
	SetProviderName(name string)
	GetProviderName() string
	GetName() string
	InitialCleanup()
	PopulateIgnoreKeys(*providerwrapper.ProviderWrapper)
	PostRefreshCleanup()
}

type Service struct {
	Name         string
	Resources    []Resource
	ProviderName string
	Args         map[string]interface{}
	Filter       []ResourceFilter
	Verbose      bool
}

func ConfigureService(service ServiceGenerator, name string, verbose bool, providerName string) {
	service.SetName(name)
	service.SetVerbose(verbose)
	service.SetProviderName(providerName)
}

func (s *Service) SetProviderName(providerName string) {
	s.ProviderName = providerName
}

func (s *Service) GetProviderName() string {
	return s.ProviderName
}

func (s *Service) SetVerbose(verbose bool) {
	s.Verbose = verbose
}

func ParseFilters(rawFilters []string) ([]ResourceFilter, error) {
	filters := make([]ResourceFilter, 0, len(rawFilters))
	for _, rawFilter := range rawFilters {
		parsed, err := ParseFilter(rawFilter)
		if err != nil {
			return nil, err
		}
		filters = append(filters, parsed...)
	}
	return filters, nil
}

func ParseFilter(rawFilter string) ([]ResourceFilter, error) {
	if rawFilter == "" {
		return nil, invalidFilter(rawFilter, "filter is empty")
	}
	if _, err := ParseFilterValues(rawFilter); err != nil {
		return nil, invalidFilter(rawFilter, err.Error())
	}

	parts := strings.Split(rawFilter, ";")
	switch len(parts) {
	case 1:
		if strings.HasPrefix(rawFilter, "Name=") {
			fieldPath, err := parseFilterComponent(parts[0], "Name")
			if err != nil {
				return nil, invalidFilter(rawFilter, err.Error())
			}
			return []ResourceFilter{{FieldPath: fieldPath}}, nil
		}

		simple := strings.SplitN(rawFilter, "=", 2)
		if len(simple) != 2 {
			return nil, invalidFilter(rawFilter, "expected service=value")
		}
		serviceName, rawValues := simple[0], simple[1]
		if strings.TrimSpace(serviceName) == "" {
			return nil, invalidFilter(rawFilter, "service name is empty")
		}
		if serviceName == "Type" || serviceName == "Value" {
			return nil, invalidFilter(rawFilter, "expected service=value or a complete named filter")
		}
		values, err := ParseFilterValues(rawValues)
		if err != nil {
			return nil, invalidFilter(rawFilter, err.Error())
		}
		if len(values) == 0 {
			return nil, invalidFilter(rawFilter, "filter value is empty")
		}
		return []ResourceFilter{{ServiceName: serviceName, FieldPath: "id", AcceptableValues: values}}, nil
	case 2:
		fieldPath, err := parseFilterComponent(parts[0], "Name")
		if err != nil {
			return nil, invalidFilter(rawFilter, err.Error())
		}
		rawValues, err := parseFilterComponent(parts[1], "Value")
		if err != nil {
			return nil, invalidFilter(rawFilter, err.Error())
		}
		values, err := ParseFilterValues(rawValues)
		if err != nil {
			return nil, invalidFilter(rawFilter, err.Error())
		}
		if len(values) == 0 {
			return nil, invalidFilter(rawFilter, "filter value is empty")
		}
		return []ResourceFilter{{FieldPath: fieldPath, AcceptableValues: values}}, nil
	case 3:
		serviceName, err := parseFilterComponent(parts[0], "Type")
		if err != nil {
			return nil, invalidFilter(rawFilter, err.Error())
		}
		fieldPath, err := parseFilterComponent(parts[1], "Name")
		if err != nil {
			return nil, invalidFilter(rawFilter, err.Error())
		}
		rawValues, err := parseFilterComponent(parts[2], "Value")
		if err != nil {
			return nil, invalidFilter(rawFilter, err.Error())
		}
		values, err := ParseFilterValues(rawValues)
		if err != nil {
			return nil, invalidFilter(rawFilter, err.Error())
		}
		if len(values) == 0 {
			return nil, invalidFilter(rawFilter, "filter value is empty")
		}
		return []ResourceFilter{{ServiceName: serviceName, FieldPath: fieldPath, AcceptableValues: values}}, nil
	default:
		return nil, invalidFilter(rawFilter, "unexpected semicolon component")
	}
}

func parseFilterComponent(component, name string) (string, error) {
	parts := strings.SplitN(component, "=", 2)
	if len(parts) != 2 || parts[0] != name {
		return "", fmt.Errorf("expected %s= component", name)
	}
	if strings.TrimSpace(parts[1]) == "" {
		return "", fmt.Errorf("%s value is empty", name)
	}
	if name != "Value" && strings.Contains(parts[1], "=") {
		return "", fmt.Errorf("%s value contains an unexpected separator", name)
	}
	return parts[1], nil
}

func invalidFilter(rawFilter, reason string) error {
	return NewFilterParseError(rawFilter, reason)
}

// NewFilterParseError reports a rejected filter while redacting user-supplied values.
func NewFilterParseError(rawFilter, reason string) error {
	return fmt.Errorf("invalid filter %q: %s", filterForError(rawFilter), reason)
}

func filterForError(rawFilter string) string {
	originalParts := strings.Split(rawFilter, ";")
	parts := append([]string(nil), originalParts...)
	for i, part := range parts {
		keyValue := strings.SplitN(part, "=", 2)
		if len(keyValue) != 2 {
			parts[i] = "<redacted>"
			continue
		}
		if (keyValue[0] == "Type" || keyValue[0] == "Name") && filterMetadataIsStructural(originalParts, i) {
			continue
		}
		key := keyValue[0]
		if key == "" {
			key = "<empty>"
		}
		parts[i] = key + "=<redacted>"
	}
	return strings.Join(parts, ";")
}

func filterMetadataIsStructural(parts []string, index int) bool {
	switch len(parts) {
	case 1:
		return index == 0 && validFilterComponent(parts[0], "Name")
	case 2:
		return index == 0 &&
			validFilterComponent(parts[0], "Name") &&
			validFilterComponent(parts[1], "Value")
	case 3:
		return (index == 0 || index == 1) &&
			validFilterComponent(parts[0], "Type") &&
			validFilterComponent(parts[1], "Name") &&
			validFilterComponent(parts[2], "Value")
	default:
		return false
	}
}

func validFilterComponent(component, name string) bool {
	_, err := parseFilterComponent(component, name)
	return err == nil
}

func (s *Service) ParseFilters(rawFilters []string) error {
	filters, err := ParseFilters(rawFilters)
	if err != nil {
		return err
	}
	s.Filter = filters
	return nil
}

func (s *Service) ParseFilter(rawFilter string) ([]ResourceFilter, error) {
	return ParseFilter(rawFilter)
}

func (s *Service) SetName(name string) {
	s.Name = name
}
func (s *Service) GetName() string {
	return s.Name
}

func (s *Service) InitialCleanup() {
	FilterCleanup(s, true)
}

func (s *Service) PostRefreshCleanup() {
	if len(s.Filter) != 0 {
		FilterCleanup(s, false)
	}
}

func (s *Service) GetArgs() map[string]interface{} {
	return s.Args
}
func (s *Service) SetArgs(args map[string]interface{}) {
	s.Args = args
}

func (s *Service) GetResources() []Resource {
	return s.Resources
}
func (s *Service) SetResources(resources []Resource) {
	s.Resources = resources
}

func (s *Service) InitResources() error {
	panic("implement me")
}

func (s *Service) PostConvertHook() error {
	return nil
}

func (s *Service) PopulateIgnoreKeys(providerWrapper *providerwrapper.ProviderWrapper) {
	var resourcesTypes []string
	for _, r := range s.Resources {
		resourcesTypes = append(resourcesTypes, r.InstanceInfo.Type)
	}
	keys := IgnoreKeys(resourcesTypes, providerWrapper)
	for k, v := range keys {
		for i := range s.Resources {
			if s.Resources[i].InstanceInfo.Type == k {
				s.Resources[i].IgnoreKeys = append(s.Resources[i].IgnoreKeys, v...)
			}
		}
	}
}
