// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	elasticachetypes "github.com/aws/aws-sdk-go-v2/service/elasticache/types"
)

func TestElastiCacheStatusImportable(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{name: "available", status: "available", want: true},
		{name: "active", status: "active", want: true},
		{name: "modifying", status: "modifying", want: true},
		{name: "creating", status: "creating", want: true},
		{name: "empty", status: "", want: false},
		{name: "deleting", status: "deleting", want: false},
		{name: "delete failed wording", status: "DELETE-FAILED", want: false},
		{name: "create failed", status: "CREATE-FAILED", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := elastiCacheStatusImportable(tt.status)
			if got != tt.want {
				t.Fatalf("elastiCacheStatusImportable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestElastiCacheUserImportable(t *testing.T) {
	tests := []struct {
		name string
		user elasticachetypes.User
		want bool
	}{
		{
			name: "no password user",
			user: elasticachetypes.User{
				Status: aws.String("active"),
				Authentication: &elasticachetypes.Authentication{
					Type: elasticachetypes.AuthenticationTypeNoPassword,
				},
			},
			want: true,
		},
		{
			name: "iam user",
			user: elasticachetypes.User{
				Status: aws.String("active"),
				Authentication: &elasticachetypes.Authentication{
					Type: elasticachetypes.AuthenticationTypeIam,
				},
			},
			want: true,
		},
		{
			name: "password user",
			user: elasticachetypes.User{
				Status: aws.String("active"),
				Authentication: &elasticachetypes.Authentication{
					Type: elasticachetypes.AuthenticationTypePassword,
				},
			},
			want: false,
		},
		{
			name: "deleting user",
			user: elasticachetypes.User{
				Status: aws.String("deleting"),
				Authentication: &elasticachetypes.Authentication{
					Type: elasticachetypes.AuthenticationTypeNoPassword,
				},
			},
			want: false,
		},
		{
			name: "missing auth",
			user: elasticachetypes.User{Status: aws.String("active")},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := elastiCacheUserImportable(tt.user)
			if got != tt.want {
				t.Fatalf("elastiCacheUserImportable() = %v, want %v", got, tt.want)
			}
		})
	}
}
