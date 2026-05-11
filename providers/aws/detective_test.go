// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	detectivetypes "github.com/aws/aws-sdk-go-v2/service/detective/types"
)

const testDetectiveGraphARN = "arn:aws:detective:us-east-1:123456789012:graph:abc123"

func TestDetectiveMemberResourceID(t *testing.T) {
	got := detectiveMemberResourceID(testDetectiveGraphARN, "210987654321")
	want := testDetectiveGraphARN + "/210987654321"
	if got != want {
		t.Fatalf("detectiveMemberResourceID() = %q, want %q", got, want)
	}
}

func TestDetectiveGraphResource(t *testing.T) {
	resource := newDetectiveGraphResource(testDetectiveGraphARN)
	if got := resource.InstanceInfo.Type; got != detectiveGraphResourceType {
		t.Fatalf("resource type = %q, want %q", got, detectiveGraphResourceType)
	}
	if got := resource.InstanceState.ID; got != testDetectiveGraphARN {
		t.Fatalf("resource ID = %q, want %q", got, testDetectiveGraphARN)
	}
	if got := resource.InstanceState.Attributes["graph_arn"]; got != testDetectiveGraphARN {
		t.Fatalf("graph_arn = %q, want %q", got, testDetectiveGraphARN)
	}
	if resource.ResourceName == newDetectiveOrganizationAdminAccountResource("123456789012").ResourceName {
		t.Fatal("expected graph and organization admin account resource names to differ")
	}
}

func TestDetectiveMemberResource(t *testing.T) {
	resource, ok := newDetectiveMemberResource(testDetectiveGraphARN, detectivetypes.MemberDetail{
		AccountId:    aws.String("210987654321"),
		EmailAddress: aws.String("member@example.com"),
	})
	if !ok {
		t.Fatal("expected member resource")
	}
	if got := resource.InstanceInfo.Type; got != detectiveMemberResourceType {
		t.Fatalf("resource type = %q, want %q", got, detectiveMemberResourceType)
	}
	wantID := testDetectiveGraphARN + "/210987654321"
	if got := resource.InstanceState.ID; got != wantID {
		t.Fatalf("resource ID = %q, want %q", got, wantID)
	}
	if got := resource.InstanceState.Attributes["email_address"]; got != "member@example.com" {
		t.Fatalf("email_address = %q", got)
	}
}

func TestDetectiveMemberResourceSkipsEmptyIdentifiers(t *testing.T) {
	if _, ok := newDetectiveMemberResource("", detectivetypes.MemberDetail{
		AccountId:    aws.String("210987654321"),
		EmailAddress: aws.String("member@example.com"),
	}); ok {
		t.Fatal("expected missing graph ARN to skip")
	}
	if _, ok := newDetectiveMemberResource(testDetectiveGraphARN, detectivetypes.MemberDetail{
		EmailAddress: aws.String("member@example.com"),
	}); ok {
		t.Fatal("expected missing account ID to skip")
	}
	if _, ok := newDetectiveMemberResource(testDetectiveGraphARN, detectivetypes.MemberDetail{
		AccountId: aws.String("210987654321"),
	}); ok {
		t.Fatal("expected missing email address to skip")
	}
}

func TestDetectiveOrganizationAdminAccountResource(t *testing.T) {
	resource := newDetectiveOrganizationAdminAccountResource("123456789012")
	if got := resource.InstanceInfo.Type; got != detectiveOrganizationAdminAccountResourceType {
		t.Fatalf("resource type = %q, want %q", got, detectiveOrganizationAdminAccountResourceType)
	}
	if got := resource.InstanceState.ID; got != "123456789012" {
		t.Fatalf("resource ID = %q", got)
	}
	if got := resource.InstanceState.Attributes["account_id"]; got != "123456789012" {
		t.Fatalf("account_id = %q", got)
	}
}

func TestDetectiveOptionalResourceUnavailable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "resource not found",
			err:  &detectivetypes.ResourceNotFoundException{Message: aws.String("not found")},
			want: true,
		},
		{
			name: "access denied",
			err:  &detectivetypes.AccessDeniedException{Message: aws.String("denied")},
			want: true,
		},
		{
			name: "not organization member",
			err:  &detectivetypes.ValidationException{Message: aws.String("This account is not a member of an organization")},
			want: true,
		},
		{
			name: "not administrator",
			err:  &detectivetypes.ValidationException{Message: aws.String("The account is not an administrator account")},
			want: true,
		},
		{
			name: "other validation",
			err:  &detectivetypes.ValidationException{Message: aws.String("invalid request")},
			want: false,
		},
		{
			name: "other error",
			err:  errors.New("boom"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectiveOptionalResourceUnavailable(tt.err); got != tt.want {
				t.Fatalf("detectiveOptionalResourceUnavailable() = %t, want %t", got, tt.want)
			}
		})
	}
}
