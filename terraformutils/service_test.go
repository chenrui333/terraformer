package terraformutils

import (
	"reflect"
	"strings"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
)

func TestEmptyFiltersParsing(t *testing.T) {
	service := Service{}
	if err := service.ParseFilters([]string{}); err != nil {
		t.Fatalf("ParseFilters() error = %v", err)
	}

	if !reflect.DeepEqual(service.Filter, []ResourceFilter{}) {
		t.Errorf("failed to parse, got %v", service.Filter)
	}
}

func TestIdFiltersParsing(t *testing.T) {
	service := Service{}
	if err := service.ParseFilters([]string{"aws_vpc=myid"}); err != nil {
		t.Fatalf("ParseFilters() error = %v", err)
	}

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
	if err := service.ParseFilters([]string{"resource=id1:'project:dataset_id'"}); err != nil {
		t.Fatalf("ParseFilters() error = %v", err)
	}

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
	if err := service.ParseFilters([]string{"aws_vpc=:myid"}); err != nil {
		t.Fatalf("ParseFilters() error = %v", err)
	}

	if !reflect.DeepEqual(service.Filter, []ResourceFilter{
		{
			ServiceName:      "aws_vpc",
			FieldPath:        "id",
			AcceptableValues: []string{"myid"},
		}}) {
		t.Errorf("failed to parse, got %v", service.Filter)
	}
}

func TestParseFilterSyntax(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		want        ResourceFilter
		wantErr     bool
		errorValues []string
	}{
		{
			name: "simple ID",
			raw:  "aws_vpc=vpc-123",
			want: ResourceFilter{ServiceName: "aws_vpc", FieldPath: "id", AcceptableValues: []string{"vpc-123"}},
		},
		{
			name: "typed field",
			raw:  "Type=sg;Name=vpc_id;Value=vpc-123",
			want: ResourceFilter{ServiceName: "sg", FieldPath: "vpc_id", AcceptableValues: []string{"vpc-123"}},
		},
		{
			name: "quoted colon",
			raw:  "resource='project:dataset_id'",
			want: ResourceFilter{ServiceName: "resource", FieldPath: "id", AcceptableValues: []string{"project:dataset_id"}},
		},
		{
			name: "value containing equals",
			raw:  "Name=expression;Value=key=value",
			want: ResourceFilter{FieldPath: "expression", AcceptableValues: []string{"key=value"}},
		},
		{name: "missing separator", raw: "aws_vpc", wantErr: true},
		{name: "extra component", raw: "Type=sg;Name=id;Value=sg-1;Extra=bad", wantErr: true},
		{name: "malformed component", raw: "Type=sg;Field=id;Value=sg-1", wantErr: true},
		{name: "malformed field separator", raw: "Name=tags=env;Value=prod", wantErr: true},
		{name: "unbalanced quote", raw: "resource='project:dataset_id", wantErr: true},
		{name: "empty simple service", raw: "=vpc-123", wantErr: true},
		{name: "empty simple value", raw: "aws_vpc=", wantErr: true},
		{name: "empty typed service", raw: "Type=;Name=id;Value=vpc-123", wantErr: true},
		{name: "empty field", raw: "Name=;Value=prod", wantErr: true},
		{name: "empty required value", raw: "Name=tags.env;Value=", wantErr: true},
		{
			name:        "error redacts rejected value",
			raw:         "resource='credential-value",
			wantErr:     true,
			errorValues: []string{"credential-value"},
		},
		{
			name:        "error redacts unstructured segments",
			raw:         "resource=top-secret;still-secret",
			wantErr:     true,
			errorValues: []string{"top-secret", "still-secret"},
		},
		{
			name:        "error redacts malformed Type component",
			raw:         "Type=service=type-secret;Name=id;Value=x",
			wantErr:     true,
			errorValues: []string{"type-secret"},
		},
		{
			name:        "error redacts metadata from malformed typed shape",
			raw:         "Type=service=type-secret;Name=name-secret;Value=x",
			wantErr:     true,
			errorValues: []string{"type-secret", "name-secret"},
		},
		{
			name:        "error redacts malformed Name component",
			raw:         "Name=field=name-secret;Value=x",
			wantErr:     true,
			errorValues: []string{"name-secret"},
		},
		{
			name:        "error redacts Type-like value suffix",
			raw:         "acl=User:producer|*|Write|Allow|Topic|orders;Type=customer-secret|Literal",
			wantErr:     true,
			errorValues: []string{"customer-secret"},
		},
		{
			name:        "error redacts Name-like value suffix",
			raw:         "acl=User:producer|*|Write|Allow|Topic|orders;Name=customer-secret|Literal",
			wantErr:     true,
			errorValues: []string{"customer-secret"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters, err := ParseFilter(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ParseFilter() error = nil, want error")
				}
				if !strings.Contains(err.Error(), "invalid filter") {
					t.Fatalf("ParseFilter() error = %q, want rejected filter context", err)
				}
				for _, errorValue := range tt.errorValues {
					if strings.Contains(err.Error(), errorValue) {
						t.Fatalf("ParseFilter() error exposed filter value %q: %q", errorValue, err)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseFilter() error = %v", err)
			}
			if len(filters) != 1 {
				t.Fatalf("ParseFilter() returned %d filters, want 1", len(filters))
			}
			if !reflect.DeepEqual(filters[0], tt.want) {
				t.Fatalf("ParseFilter() = %#v, want %#v", filters[0], tt.want)
			}
		})
	}
}

func TestParseFiltersRejectsEntireList(t *testing.T) {
	service := Service{}
	err := service.ParseFilters([]string{"aws_vpc=vpc-123", "Type=sg;Name=id;Value="})
	if err == nil {
		t.Fatal("ParseFilters() error = nil, want error")
	}
	if len(service.Filter) != 0 {
		t.Fatalf("ParseFilters() partially applied filters: %#v", service.Filter)
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
