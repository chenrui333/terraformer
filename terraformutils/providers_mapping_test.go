// SPDX-License-Identifier: Apache-2.0

package terraformutils

import (
	"sort"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils/providerwrapper"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
	"github.com/zclconf/go-cty/cty"
)

type mockProvider struct {
	Provider
	name     string
	services map[string]ServiceGenerator
}

func (m *mockProvider) Init(_ []string) error                                  { return nil }
func (m *mockProvider) InitService(svc string, _ bool) error                   { m.Service = m.services[svc]; return nil }
func (m *mockProvider) GetName() string                                        { return m.name }
func (m *mockProvider) GetConfig() cty.Value                                   { return cty.EmptyObjectVal }
func (m *mockProvider) GetBasicConfig() cty.Value                              { return cty.EmptyObjectVal }
func (m *mockProvider) GetSupportedService() map[string]ServiceGenerator       { return m.services }
func (m *mockProvider) GenerateFiles()                                         {}
func (m *mockProvider) GetProviderData(_ ...string) map[string]interface{}     { return nil }
func (m *mockProvider) GenerateOutputPath() error                              { return nil }
func (m *mockProvider) GetResourceConnections() map[string]map[string][]string { return nil }
func (m *mockProvider) PopulateIgnoreKeys(_ *providerwrapper.ProviderWrapper)  {}

func newMockProvider(serviceNames ...string) *mockProvider {
	services := make(map[string]ServiceGenerator, len(serviceNames))
	for _, s := range serviceNames {
		services[s] = &Service{Name: s}
	}
	return &mockProvider{name: "aws", services: services}
}

func TestNewProvidersMapping(t *testing.T) {
	base := newMockProvider("vpc", "ec2")
	pm := NewProvidersMapping(base)

	if pm == nil {
		t.Fatal("NewProvidersMapping returned nil")
	}
	if len(pm.Resources) != 0 {
		t.Errorf("Resources should be empty, got %d", len(pm.Resources))
	}
	if len(pm.Services) != 0 {
		t.Errorf("Services should be empty, got %d", len(pm.Services))
	}
	if len(pm.Providers) != 0 {
		t.Errorf("Providers should be empty, got %d", len(pm.Providers))
	}
}

func TestGetBaseProvider(t *testing.T) {
	base := newMockProvider()
	pm := NewProvidersMapping(base)
	if pm.GetBaseProvider() != base {
		t.Error("GetBaseProvider did not return the original provider")
	}
}

func TestAddServiceToProvider(t *testing.T) {
	base := newMockProvider("vpc")
	pm := NewProvidersMapping(base)

	newProv := pm.AddServiceToProvider("vpc")

	if newProv == nil {
		t.Fatal("AddServiceToProvider returned nil")
	}
	if newProv == base {
		t.Error("AddServiceToProvider should return a deep copy, not the base")
	}
	if !pm.Services["vpc"] {
		t.Error("service 'vpc' not added to Services map")
	}
	if !pm.Providers[newProv] {
		t.Error("new provider not added to Providers map")
	}
}

func TestAddMultipleServices(t *testing.T) {
	base := newMockProvider("vpc", "ec2", "s3")
	pm := NewProvidersMapping(base)

	p1 := pm.AddServiceToProvider("vpc")
	p2 := pm.AddServiceToProvider("ec2")
	p3 := pm.AddServiceToProvider("s3")

	if p1 == p2 || p2 == p3 || p1 == p3 {
		t.Error("each AddServiceToProvider call should return a distinct provider")
	}
	if len(pm.Providers) != 3 {
		t.Errorf("expected 3 providers, got %d", len(pm.Providers))
	}
	if len(pm.Services) != 3 {
		t.Errorf("expected 3 services, got %d", len(pm.Services))
	}
}

func TestGetServices(t *testing.T) {
	base := newMockProvider("vpc", "ec2")
	pm := NewProvidersMapping(base)
	pm.AddServiceToProvider("vpc")
	pm.AddServiceToProvider("ec2")

	services := pm.GetServices()
	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d: %v", len(services), services)
	}
	found := map[string]bool{}
	for _, s := range services {
		if s == "" {
			t.Fatalf("GetServices returned empty service: %v", services)
		}
		found[s] = true
	}
	if !found["vpc"] || !found["ec2"] {
		t.Errorf("GetServices missing expected services, got %v", services)
	}
}

func TestRemoveServices(t *testing.T) {
	base := newMockProvider("vpc", "ec2")
	pm := NewProvidersMapping(base)
	pm.AddServiceToProvider("vpc")
	pm.AddServiceToProvider("ec2")

	pm.RemoveServices([]string{"vpc"})

	if pm.Services["vpc"] {
		t.Error("vpc should have been removed from Services")
	}
	if !pm.Services["ec2"] {
		t.Error("ec2 should still be in Services")
	}
	if len(pm.Providers) != 1 {
		t.Errorf("expected 1 provider after removal, got %d", len(pm.Providers))
	}
}

func TestRemoveServicesNonExistent(t *testing.T) {
	base := newMockProvider("vpc")
	pm := NewProvidersMapping(base)
	pm.AddServiceToProvider("vpc")

	pm.RemoveServices([]string{"nonexistent"})

	if !pm.Services["vpc"] {
		t.Error("vpc should still be in Services")
	}
}

func TestShuffleResources(t *testing.T) {
	base := newMockProvider("vpc")
	pm := NewProvidersMapping(base)

	r1 := &Resource{InstanceInfo: &tfcompat.InstanceInfo{Id: "a"}}
	r2 := &Resource{InstanceInfo: &tfcompat.InstanceInfo{Id: "b"}}
	r3 := &Resource{InstanceInfo: &tfcompat.InstanceInfo{Id: "c"}}
	pm.Resources[r1] = true
	pm.Resources[r2] = true
	pm.Resources[r3] = true

	shuffled := pm.ShuffleResources()
	if len(shuffled) != 3 {
		t.Errorf("ShuffleResources returned %d resources, want 3", len(shuffled))
	}
}

func TestProcessResources(t *testing.T) {
	base := newMockProvider("vpc")
	pm := NewProvidersMapping(base)

	prov := pm.AddServiceToProvider("vpc")
	svc := &Service{
		Name: "vpc",
		Resources: []Resource{
			NewSimpleResource("vpc-1", "main", "aws_vpc", "aws", nil),
			NewSimpleResource("vpc-2", "dev", "aws_vpc", "aws", nil),
		},
	}
	prov.(*mockProvider).Service = svc

	pm.ProcessResources(false)

	if len(pm.Resources) != 2 {
		t.Errorf("expected 2 resources, got %d", len(pm.Resources))
	}
}

func TestProcessResourcesCleanup(t *testing.T) {
	base := newMockProvider("vpc")
	pm := NewProvidersMapping(base)

	prov := pm.AddServiceToProvider("vpc")
	svc := &Service{
		Name: "vpc",
		Resources: []Resource{
			NewSimpleResource("vpc-1", "main", "aws_vpc", "aws", nil),
		},
	}
	prov.(*mockProvider).Service = svc

	pm.ProcessResources(false)
	if len(pm.Resources) != 1 {
		t.Fatalf("expected 1 resource after initial load, got %d", len(pm.Resources))
	}

	svc.Resources = append(svc.Resources,
		NewSimpleResource("vpc-2", "dev", "aws_vpc", "aws", nil),
	)
	pm.ProcessResources(true)

	if len(pm.Resources) != 2 {
		t.Errorf("expected 2 resources after cleanup reload, got %d", len(pm.Resources))
	}
}

func TestMatchProvider(t *testing.T) {
	base := newMockProvider("vpc")
	pm := NewProvidersMapping(base)
	prov := pm.AddServiceToProvider("vpc")

	r := &Resource{InstanceInfo: &tfcompat.InstanceInfo{Id: "aws_vpc.main"}}
	pm.Resources[r] = true
	pm.resourceToProvider[r] = prov

	if pm.MatchProvider(r) != prov {
		t.Error("MatchProvider did not return the correct provider")
	}
}

func TestSetResources(t *testing.T) {
	base := newMockProvider("vpc")
	pm := NewProvidersMapping(base)
	prov := pm.AddServiceToProvider("vpc")
	svc := &Service{Name: "vpc"}
	prov.(*mockProvider).Service = svc

	r1 := &Resource{InstanceInfo: &tfcompat.InstanceInfo{Id: "aws_vpc.a"}}
	r2 := &Resource{InstanceInfo: &tfcompat.InstanceInfo{Id: "aws_vpc.b"}}
	pm.Resources[r1] = true
	pm.Resources[r2] = true
	pm.resourceToProvider[r1] = prov
	pm.resourceToProvider[r2] = prov

	pm.SetResources([]*Resource{r1})

	if len(pm.Resources) != 1 {
		t.Errorf("expected 1 resource after SetResources, got %d", len(pm.Resources))
	}
	if !pm.Resources[r1] {
		t.Error("r1 should be in Resources")
	}
}

func TestGetResourcesByService(t *testing.T) {
	base := newMockProvider("vpc", "ec2")
	pm := NewProvidersMapping(base)

	provVpc := pm.AddServiceToProvider("vpc")
	provEc2 := pm.AddServiceToProvider("ec2")

	r1 := &Resource{InstanceInfo: &tfcompat.InstanceInfo{Id: "aws_vpc.main"}}
	r2 := &Resource{InstanceInfo: &tfcompat.InstanceInfo{Id: "aws_instance.web"}}
	pm.Resources[r1] = true
	pm.Resources[r2] = true
	pm.resourceToProvider[r1] = provVpc
	pm.resourceToProvider[r2] = provEc2

	byService := pm.GetResourcesByService()

	if len(byService) != 2 {
		t.Errorf("expected 2 service groups, got %d", len(byService))
	}
	if len(byService["vpc"]) != 1 {
		t.Errorf("vpc should have 1 resource, got %d", len(byService["vpc"]))
	}
	if len(byService["ec2"]) != 1 {
		t.Errorf("ec2 should have 1 resource, got %d", len(byService["ec2"]))
	}
}

func TestCleanupProviders(t *testing.T) {
	base := newMockProvider("vpc")
	pm := NewProvidersMapping(base)

	prov := pm.AddServiceToProvider("vpc")
	svc := &Service{
		Name: "vpc",
		Resources: []Resource{
			NewSimpleResource("vpc-1", "main", "aws_vpc", "aws", nil),
		},
	}
	prov.(*mockProvider).Service = svc

	pm.ProcessResources(false)
	pm.CleanupProviders()

	if len(pm.Resources) != 1 {
		t.Errorf("expected 1 resource after cleanup, got %d", len(pm.Resources))
	}
}

func TestGetServicesSorted(t *testing.T) {
	base := newMockProvider("s3", "ec2", "vpc")
	pm := NewProvidersMapping(base)
	pm.AddServiceToProvider("s3")
	pm.AddServiceToProvider("ec2")
	pm.AddServiceToProvider("vpc")

	services := pm.GetServices()
	sort.Strings(services)

	want := []string{"ec2", "s3", "vpc"}
	if len(services) != len(want) {
		t.Fatalf("expected %d services, got %d: %v", len(want), len(services), services)
	}
	for i, s := range want {
		if services[i] != s {
			t.Errorf("sorted service[%d] = %q, want %q", i, services[i], s)
		}
	}
}
