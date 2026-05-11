// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
)

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
