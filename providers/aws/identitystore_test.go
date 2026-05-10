// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	identitystoretypes "github.com/aws/aws-sdk-go-v2/service/identitystore/types"
)

func TestIdentityStoreResourceID(t *testing.T) {
	got := identityStoreResourceID("d-1234567890", "g-123")
	want := "d-1234567890/g-123"
	if got != want {
		t.Fatalf("identityStoreResourceID() = %q, want %q", got, want)
	}
}

func TestIdentityStoreGroupResource(t *testing.T) {
	resource := newIdentityStoreGroupResource("d-1234567890", identitystoretypes.Group{
		Description: aws.String("Engineering team"),
		DisplayName: aws.String("Engineering"),
		GroupId:     aws.String("g-123"),
	})

	if got, want := resource.InstanceState.ID, "d-1234567890/g-123"; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceInfo.Type, identityStoreGroupResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["identity_store_id"], "d-1234567890"; got != want {
		t.Fatalf("identity_store_id = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["group_id"], "g-123"; got != want {
		t.Fatalf("group_id = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["display_name"], "Engineering"; got != want {
		t.Fatalf("display_name = %q, want %q", got, want)
	}
}

func TestIdentityStoreGroupMembershipResource(t *testing.T) {
	resource := newIdentityStoreGroupMembershipResource("d-1234567890", "g-123", "u-456", "m-789")

	if got, want := resource.InstanceState.ID, "d-1234567890/m-789"; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceInfo.Type, identityStoreGroupMembershipResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"group_id":          "g-123",
		"identity_store_id": "d-1234567890",
		"member_id":         "u-456",
		"membership_id":     "m-789",
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestIdentityStoreUserResource(t *testing.T) {
	resource := newIdentityStoreUserResource("d-1234567890", identitystoretypes.User{
		DisplayName: aws.String("Jane Doe"),
		UserId:      aws.String("u-456"),
		UserName:    aws.String("jane@example.com"),
	})

	if got, want := resource.InstanceState.ID, "d-1234567890/u-456"; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceInfo.Type, identityStoreUserResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	attributes := resource.InstanceState.Attributes
	if got, want := attributes["identity_store_id"], "d-1234567890"; got != want {
		t.Fatalf("identity_store_id = %q, want %q", got, want)
	}
	if got, want := attributes["user_id"], "u-456"; got != want {
		t.Fatalf("user_id = %q, want %q", got, want)
	}
	if got, want := attributes["user_name"], "jane@example.com"; got != want {
		t.Fatalf("user_name = %q, want %q", got, want)
	}
	if _, ok := attributes["use_name"]; ok {
		t.Fatalf("unexpected legacy use_name attribute in %#v", attributes)
	}
}

func TestIdentityStoreResourceNamesDoNotCollapseJoinedParts(t *testing.T) {
	left := newIdentityStoreGroupMembershipResource("store", "a_b", "c", "membership-left")
	right := newIdentityStoreGroupMembershipResource("store", "a", "b_c", "membership-right")
	if left.ResourceName == right.ResourceName {
		t.Fatalf("group membership resource names collide: %q", left.ResourceName)
	}
}

func TestIdentityStoreResourceNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "resource not found", err: &identitystoretypes.ResourceNotFoundException{}, want: true},
		{name: "wrapped resource not found", err: errors.Join(errors.New("lookup failed"), &identitystoretypes.ResourceNotFoundException{}), want: true},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := identityStoreResourceNotFound(tt.err); got != tt.want {
				t.Fatalf("identityStoreResourceNotFound(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}
