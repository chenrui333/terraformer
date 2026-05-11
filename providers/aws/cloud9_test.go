// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloud9/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewCloud9EnvironmentEC2Resource(t *testing.T) {
	resource := newCloud9EnvironmentEC2Resource("env-1234567890abcdef0")
	assertCloud9Resource(t, resource, "env-1234567890abcdef0", "env-1234567890abcdef0", cloud9EnvironmentEC2ResourceType)
}

func TestNewCloud9EnvironmentMembershipResource(t *testing.T) {
	userArn := "arn:aws:iam::123456789012:user/alice"
	resource, ok := newCloud9EnvironmentMembershipResource(types.EnvironmentMember{
		EnvironmentId: aws.String("env-1234567890abcdef0"),
		Permissions:   types.PermissionsReadWrite,
		UserArn:       aws.String(userArn),
	})
	assertCloud9ResourceOK(t, resource, ok, cloud9EnvironmentMembershipImportID("env-1234567890abcdef0", userArn), cloud9ResourceName("membership", "env-1234567890abcdef0", userArn), cloud9EnvironmentMembershipResourceType)
	assertCloud9Attribute(t, resource, "environment_id", "env-1234567890abcdef0")
	assertCloud9Attribute(t, resource, "permissions", "read-write")
	assertCloud9Attribute(t, resource, "user_arn", userArn)
}

func TestCloud9EnvironmentMembershipImportID(t *testing.T) {
	got := cloud9EnvironmentMembershipImportID("env-123", "arn:aws:iam::123456789012:user/alice")
	want := "env-123#arn:aws:iam::123456789012:user/alice"
	if got != want {
		t.Fatalf("Cloud9 membership import ID = %q, want %q", got, want)
	}
}

func TestCloud9EnvironmentMembershipImportable(t *testing.T) {
	tests := []struct {
		name        string
		permissions types.Permissions
		want        bool
	}{
		{name: "read only", permissions: types.PermissionsReadOnly, want: true},
		{name: "read write", permissions: types.PermissionsReadWrite, want: true},
		{name: "owner", permissions: types.PermissionsOwner, want: false},
		{name: "empty", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cloud9EnvironmentMembershipImportable(tt.permissions); got != tt.want {
				t.Fatalf("cloud9EnvironmentMembershipImportable(%q) = %t, want %t", tt.permissions, got, tt.want)
			}
		})
	}
}

func TestCloud9MembershipSkipsEmptyIdentifiers(t *testing.T) {
	if _, ok := newCloud9EnvironmentMembershipResource(types.EnvironmentMember{
		Permissions: types.PermissionsReadOnly,
		UserArn:     aws.String("arn:aws:iam::123456789012:user/alice"),
	}); ok {
		t.Fatal("membership with empty environment ID should be skipped")
	}

	if _, ok := newCloud9EnvironmentMembershipResource(types.EnvironmentMember{
		EnvironmentId: aws.String("env-1234567890abcdef0"),
		Permissions:   types.PermissionsReadOnly,
	}); ok {
		t.Fatal("membership with empty user ARN should be skipped")
	}
}

func TestCloud9ResourceNamesPreserveSegmentBoundaries(t *testing.T) {
	left := terraformutils.TfSanitize(cloud9ResourceName("membership", "a/b_c"))
	right := terraformutils.TfSanitize(cloud9ResourceName("member", "ship/a_b_c"))
	if left == right {
		t.Fatalf("Cloud9 resource names collide: %q", left)
	}
}

func assertCloud9ResourceOK(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantName, wantType string) {
	t.Helper()
	if !ok {
		t.Fatal("resource was skipped")
	}
	assertCloud9Resource(t, resource, wantID, wantName, wantType)
}

func assertCloud9Resource(t *testing.T, resource terraformutils.Resource, wantID, wantName, wantType string) {
	t.Helper()
	if got := resource.InstanceState.ID; got != wantID {
		t.Fatalf("resource ID = %q, want %q", got, wantID)
	}
	if got := resource.ResourceName; got != terraformutils.TfSanitize(wantName) {
		t.Fatalf("resource name = %q, want %q", got, terraformutils.TfSanitize(wantName))
	}
	if got := resource.InstanceInfo.Type; got != wantType {
		t.Fatalf("resource type = %q, want %q", got, wantType)
	}
}

func assertCloud9Attribute(t *testing.T, resource terraformutils.Resource, key, want string) {
	t.Helper()
	if got := resource.InstanceState.Attributes[key]; got != want {
		t.Fatalf("resource attribute %q = %q, want %q", key, got, want)
	}
}
