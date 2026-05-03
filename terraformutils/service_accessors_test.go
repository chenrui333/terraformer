// SPDX-License-Identifier: Apache-2.0

package terraformutils

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
)

func TestServiceSetGetName(t *testing.T) {
	s := &Service{}
	s.SetName("vpc")
	if got := s.GetName(); got != "vpc" {
		t.Errorf("GetName() = %q, want %q", got, "vpc")
	}
}

func TestServiceSetGetProviderName(t *testing.T) {
	s := &Service{}
	s.SetProviderName("aws")
	if got := s.GetProviderName(); got != "aws" {
		t.Errorf("GetProviderName() = %q, want %q", got, "aws")
	}
}

func TestConfigureService(t *testing.T) {
	s := &Service{}

	ConfigureService(s, "vpc", true, "aws")

	if got := s.GetName(); got != "vpc" {
		t.Errorf("GetName() = %q, want %q", got, "vpc")
	}
	if got := s.GetProviderName(); got != "aws" {
		t.Errorf("GetProviderName() = %q, want %q", got, "aws")
	}
	if !s.Verbose {
		t.Error("Verbose should be true")
	}
}

func TestServiceSetGetArgs(t *testing.T) {
	s := &Service{}
	args := map[string]interface{}{"region": "us-east-1"}
	s.SetArgs(args)
	got := s.GetArgs()
	if got["region"] != "us-east-1" {
		t.Errorf("GetArgs()[region] = %v, want %q", got["region"], "us-east-1")
	}
}

func TestServiceSetGetResources(t *testing.T) {
	s := &Service{}
	resources := []Resource{
		NewSimpleResource("id-1", "name-1", "aws_vpc", "aws", nil),
	}
	s.SetResources(resources)
	got := s.GetResources()
	if len(got) != 1 {
		t.Fatalf("GetResources() len = %d, want 1", len(got))
	}
	if got[0].InstanceState.ID != "id-1" {
		t.Errorf("resource ID = %q, want %q", got[0].InstanceState.ID, "id-1")
	}
}

func TestServiceSetVerbose(t *testing.T) {
	s := &Service{}
	s.SetVerbose(true)
	if !s.Verbose {
		t.Error("Verbose should be true")
	}
}

func TestServicePostConvertHookDefault(t *testing.T) {
	s := &Service{}
	if err := s.PostConvertHook(); err != nil {
		t.Errorf("PostConvertHook() = %v, want nil", err)
	}
}

func TestServiceInitResourcesPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("InitResources() did not panic")
		}
	}()
	s := &Service{}
	_ = s.InitResources()
}

func TestServiceParseFilterThreeParts(t *testing.T) {
	s := &Service{}
	filters := s.ParseFilter("Type=vpc;Name=tags.env;Value=prod")
	if len(filters) != 1 {
		t.Fatalf("ParseFilter() len = %d, want 1", len(filters))
	}
	f := filters[0]
	if f.ServiceName != "vpc" {
		t.Errorf("ServiceName = %q, want %q", f.ServiceName, "vpc")
	}
	if f.FieldPath != "tags.env" {
		t.Errorf("FieldPath = %q, want %q", f.FieldPath, "tags.env")
	}
	if len(f.AcceptableValues) != 1 || f.AcceptableValues[0] != "prod" {
		t.Errorf("AcceptableValues = %v, want [prod]", f.AcceptableValues)
	}
}

func TestServiceParseFilterInvalid(t *testing.T) {
	s := &Service{}
	filters := s.ParseFilter("a;b;c;d")
	if len(filters) != 0 {
		t.Errorf("ParseFilter() for invalid input should return empty, got %d", len(filters))
	}
}

func TestServiceInitialCleanupFiltersById(t *testing.T) {
	s := &Service{
		Resources: []Resource{
			{
				InstanceInfo:  &tfcompat.InstanceInfo{Type: "aws_vpc", Id: "aws_vpc.a"},
				InstanceState: &tfcompat.InstanceState{ID: "vpc-111"},
			},
			{
				InstanceInfo:  &tfcompat.InstanceInfo{Type: "aws_vpc", Id: "aws_vpc.b"},
				InstanceState: &tfcompat.InstanceState{ID: "vpc-222"},
			},
		},
	}
	s.ParseFilters([]string{"aws_vpc=vpc-111"})
	s.InitialCleanup()

	if len(s.Resources) != 1 {
		t.Fatalf("Resources len = %d, want 1", len(s.Resources))
	}
	if s.Resources[0].InstanceState.ID != "vpc-111" {
		t.Errorf("kept resource ID = %q, want %q", s.Resources[0].InstanceState.ID, "vpc-111")
	}
}

func TestServicePostRefreshCleanupNoFilter(t *testing.T) {
	s := &Service{
		Resources: []Resource{
			{
				InstanceInfo:  &tfcompat.InstanceInfo{Type: "aws_vpc", Id: "aws_vpc.a"},
				InstanceState: &tfcompat.InstanceState{ID: "vpc-111"},
			},
		},
	}
	s.PostRefreshCleanup()
	if len(s.Resources) != 1 {
		t.Errorf("PostRefreshCleanup with no filters should keep all resources, got %d", len(s.Resources))
	}
}
