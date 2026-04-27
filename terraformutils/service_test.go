package terraformutils

import (
	"reflect"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
)

func TestEmptyFiltersParsing(t *testing.T) {
	service := Service{}
	service.ParseFilters([]string{})

	if !reflect.DeepEqual(service.Filter, []ResourceFilter{}) {
		t.Errorf("failed to parse, got %v", service.Filter)
	}
}

func TestIdFiltersParsing(t *testing.T) {
	service := Service{}
	service.ParseFilters([]string{"aws_vpc=myid"})

	if !reflect.DeepEqual(service.Filter, []ResourceFilter{
		{
			ServiceName:      "aws_vpc",
			FieldPath:        "id",
			AcceptableValues: []string{"myid"},
		}}) {
		t.Errorf("failed to parse, got %v", service.Filter)
	}
}

func TestComplexIdFiltersParsing(t *testing.T) {
	service := Service{}
	service.ParseFilters([]string{"resource=id1:'project:dataset_id'"})

	if !reflect.DeepEqual(service.Filter, []ResourceFilter{
		{
			ServiceName:      "resource",
			FieldPath:        "id",
			AcceptableValues: []string{"id1", "project:dataset_id"},
		}}) {
		t.Errorf("failed to parse, got %v", service.Filter)
	}
}

func TestEdgeIdFiltersParsing(t *testing.T) {
	service := Service{}
	service.ParseFilters([]string{"aws_vpc=:myid"})

	if !reflect.DeepEqual(service.Filter, []ResourceFilter{
		{
			ServiceName:      "aws_vpc",
			FieldPath:        "id",
			AcceptableValues: []string{"myid"},
		}}) {
		t.Errorf("failed to parse, got %v", service.Filter)
	}
}

func TestServiceIdCleanupWithFilter(t *testing.T) {
	service := Service{
		Resources: []Resource{{
			InstanceInfo: &tfcompat.InstanceInfo{
				Type: "type1",
			},
			InstanceState: &tfcompat.InstanceState{
				ID: "myid",
			}}, {
			InstanceInfo: &tfcompat.InstanceInfo{
				Type: "type2",
			},
			InstanceState: &tfcompat.InstanceState{
				ID: "myid",
			}}},
	}
	service.ParseFilters([]string{"type1=:otherId"})
	service.InitialCleanup()

	if !reflect.DeepEqual(len(service.Resources), 1) {
		t.Errorf("failed to cleanup")
	}
}

func TestServiceAttributeCleanupWithFilter(t *testing.T) {
	service := Service{
		Resources: []Resource{
			{
				InstanceInfo: &tfcompat.InstanceInfo{
					Type: "aws_vpc",
				},
				InstanceState: &tfcompat.InstanceState{
					ID: "vpc1",
				},
				Item: mapI("tags", mapI("Name", "some"))},
			{
				InstanceInfo: &tfcompat.InstanceInfo{
					Type: "aws_vpc",
				},
				InstanceState: &tfcompat.InstanceState{
					ID: "vpc2",
				},
				Item: mapI("tags", mapI("Name", "default"))}},
	}
	service.ParseFilters([]string{"Name=tags.Name;Value=default"})
	service.PostRefreshCleanup()

	if !reflect.DeepEqual(len(service.Resources), 1) {
		t.Errorf("failed to cleanup")
	}
}

func TestServiceAttributeNameOnlyCleanupWithFilter(t *testing.T) {
	service := Service{
		Resources: []Resource{
			{
				InstanceInfo: &tfcompat.InstanceInfo{
					Type: "aws_vpc",
				},
				InstanceState: &tfcompat.InstanceState{
					ID: "vpc1",
				},
				Item: mapI("tags", mapI("Abc", nil))},
			{
				InstanceInfo: &tfcompat.InstanceInfo{
					Type: "aws_vpc",
				},
				InstanceState: &tfcompat.InstanceState{
					ID: "vpc2",
				},
				Item: mapI("tags", mapI("Name", "default"))}},
	}
	service.ParseFilters([]string{"Name=tags.Abc"})
	service.PostRefreshCleanup()

	if !reflect.DeepEqual(len(service.Resources), 1) {
		t.Errorf("failed to cleanup")
	}
}
