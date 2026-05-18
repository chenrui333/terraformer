// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestAWSServiceParseFiltersNormalizesAWSResourceTypes(t *testing.T) {
	s := AWSService{}
	s.ParseFilters([]string{"Type=aws_ebs_snapshot;Name=id;Value=snap-123"})

	if len(s.Filter) != 1 {
		t.Fatalf("filters length = %d, want 1", len(s.Filter))
	}
	if got := s.Filter[0].ServiceName; got != "ebs_snapshot" {
		t.Fatalf("filter service name = %q, want ebs_snapshot", got)
	}

	s.Resources = []terraformutils.Resource{
		terraformutils.NewSimpleResource("snap-123", "snap-123", "aws_ebs_snapshot", "aws", nil),
		terraformutils.NewSimpleResource("snap-456", "snap-456", "aws_ebs_snapshot", "aws", nil),
	}
	s.InitialCleanup()

	if len(s.Resources) != 1 {
		t.Fatalf("resources length after cleanup = %d, want 1", len(s.Resources))
	}
	if got := s.Resources[0].InstanceState.ID; got != "snap-123" {
		t.Fatalf("kept resource ID = %q, want snap-123", got)
	}
}

func TestAWSServiceParseFiltersNormalizesTransitGatewayServiceName(t *testing.T) {
	s := AWSService{}
	s.ParseFilters([]string{"transit_gateway=tgw-123"})

	if len(s.Filter) != 1 {
		t.Fatalf("filters length = %d, want 1", len(s.Filter))
	}
	if got := s.Filter[0].ServiceName; got != "ec2_transit_gateway" {
		t.Fatalf("filter service name = %q, want ec2_transit_gateway", got)
	}

	s.Resources = []terraformutils.Resource{
		terraformutils.NewSimpleResource("tgw-123", "tgw-123", "aws_ec2_transit_gateway", "aws", nil),
		terraformutils.NewSimpleResource("tgw-456", "tgw-456", "aws_ec2_transit_gateway", "aws", nil),
	}
	s.InitialCleanup()

	if len(s.Resources) != 1 {
		t.Fatalf("resources length after cleanup = %d, want 1", len(s.Resources))
	}
	if got := s.Resources[0].InstanceState.ID; got != "tgw-123" {
		t.Fatalf("kept resource ID = %q, want tgw-123", got)
	}
}

func TestShouldLoadAWSResourceForTypedFilters(t *testing.T) {
	tests := []struct {
		name         string
		filters      []terraformutils.ResourceFilter
		serviceNames []string
		want         bool
	}{
		{name: "no filters loads resource", serviceNames: []string{"ebs_volume"}, want: true},
		{
			name: "untyped filter loads resource",
			filters: []terraformutils.ResourceFilter{{
				FieldPath:        "id",
				AcceptableValues: []string{"vol-123"},
			}},
			serviceNames: []string{"ebs_volume"},
			want:         true,
		},
		{
			name: "matching typed filter loads resource",
			filters: []terraformutils.ResourceFilter{{
				ServiceName:      "ebs_volume",
				FieldPath:        "id",
				AcceptableValues: []string{"vol-123"},
			}},
			serviceNames: []string{"ebs_volume"},
			want:         true,
		},
		{
			name: "aws prefix typed filter loads resource",
			filters: []terraformutils.ResourceFilter{{
				ServiceName:      "aws_ebs_volume",
				FieldPath:        "id",
				AcceptableValues: []string{"vol-123"},
			}},
			serviceNames: []string{"ebs_volume"},
			want:         true,
		},
		{
			name: "different typed filter skips resource",
			filters: []terraformutils.ResourceFilter{{
				ServiceName:      "ebs_volume",
				FieldPath:        "id",
				AcceptableValues: []string{"vol-123"},
			}},
			serviceNames: []string{"ebs_snapshot"},
			want:         false,
		},
		{
			name: "any matching typed filter loads resource",
			filters: []terraformutils.ResourceFilter{
				{
					ServiceName:      "ebs_volume",
					FieldPath:        "id",
					AcceptableValues: []string{"vol-123"},
				},
				{
					ServiceName:      "ebs_snapshot",
					FieldPath:        "id",
					AcceptableValues: []string{"snap-123"},
				},
			},
			serviceNames: []string{"ebs_snapshot"},
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldLoadAWSResourceForTypedFilters(tt.filters, tt.serviceNames...); got != tt.want {
				t.Fatalf("shouldLoadAWSResourceForTypedFilters() = %t, want %t", got, tt.want)
			}
		})
	}
}
