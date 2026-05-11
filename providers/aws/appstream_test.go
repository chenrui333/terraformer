// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	appstreamtypes "github.com/aws/aws-sdk-go-v2/service/appstream/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewAppStreamFleetResource(t *testing.T) {
	for _, state := range []appstreamtypes.FleetState{
		"",
		appstreamtypes.FleetStateRunning,
		appstreamtypes.FleetStateStopped,
		appstreamtypes.FleetStateStarting,
		appstreamtypes.FleetStateStopping,
	} {
		resource, ok := newAppStreamFleetResource(appstreamtypes.Fleet{
			Name:  appStreamString("core-fleet"),
			State: state,
		})
		assertAppStreamResource(t, resource, ok, "core-fleet", appStreamResourceName("fleet", "core-fleet"), appStreamFleetResourceType)
	}

	if _, ok := newAppStreamFleetResource(appstreamtypes.Fleet{State: appstreamtypes.FleetStateRunning}); ok {
		t.Fatal("fleet with empty name should be skipped")
	}
}

func TestNewAppStreamStackResource(t *testing.T) {
	resource, ok := newAppStreamStackResource(appstreamtypes.Stack{Name: appStreamString("core-stack")})
	assertAppStreamResource(t, resource, ok, "core-stack", appStreamResourceName("stack", "core-stack"), appStreamStackResourceType)

	if _, ok := newAppStreamStackResource(appstreamtypes.Stack{}); ok {
		t.Fatal("stack with empty name should be skipped")
	}
}

func TestNewAppStreamFleetStackAssociationResource(t *testing.T) {
	resource, ok := newAppStreamFleetStackAssociationResource("core-fleet", "core-stack")
	assertAppStreamResource(t, resource, ok, "core-fleet/core-stack", appStreamResourceName("fleet-stack-association", "core-fleet", "core-stack"), appStreamFleetStackAssociationResourceType)

	if got := resource.InstanceInfo.Type; got != appStreamFleetStackAssociationResourceType {
		t.Fatalf("resource type = %q, want %q", got, appStreamFleetStackAssociationResourceType)
	}
	if _, ok := newAppStreamFleetStackAssociationResource("", "core-stack"); ok {
		t.Fatal("association with empty fleet name should be skipped")
	}
	if _, ok := newAppStreamFleetStackAssociationResource("core-fleet", ""); ok {
		t.Fatal("association with empty stack name should be skipped")
	}
}

func TestNewAppStreamUserResource(t *testing.T) {
	resource, ok := newAppStreamUserResource(appstreamtypes.User{
		AuthenticationType: appstreamtypes.AuthenticationTypeUserpool,
		UserName:           appStreamString("user@example.com"),
	})
	assertAppStreamResource(t, resource, ok, "user@example.com/USERPOOL", appStreamResourceName("user", "USERPOOL", "user@example.com"), appStreamUserResourceType)
	assertAppStreamAttribute(t, resource, "authentication_type", "USERPOOL")
	assertAppStreamAttribute(t, resource, "user_name", "user@example.com")

	if _, ok := newAppStreamUserResource(appstreamtypes.User{AuthenticationType: appstreamtypes.AuthenticationTypeUserpool}); ok {
		t.Fatal("user with empty user name should be skipped")
	}
	if _, ok := newAppStreamUserResource(appstreamtypes.User{UserName: appStreamString("user@example.com")}); ok {
		t.Fatal("user with empty authentication type should be skipped")
	}
}

func TestNewAppStreamUserStackAssociationResource(t *testing.T) {
	resource, ok := newAppStreamUserStackAssociationResource(appstreamtypes.UserStackAssociation{
		AuthenticationType: appstreamtypes.AuthenticationTypeUserpool,
		StackName:          appStreamString("core-stack"),
		UserName:           appStreamString("user@example.com"),
	})
	assertAppStreamResource(t, resource, ok, "user@example.com/USERPOOL/core-stack", appStreamResourceName("user-stack-association", "USERPOOL", "user@example.com", "core-stack"), appStreamUserStackAssociationResourceType)
	assertAppStreamAttribute(t, resource, "authentication_type", "USERPOOL")
	assertAppStreamAttribute(t, resource, "stack_name", "core-stack")
	assertAppStreamAttribute(t, resource, "user_name", "user@example.com")

	if _, ok := newAppStreamUserStackAssociationResource(appstreamtypes.UserStackAssociation{
		AuthenticationType: appstreamtypes.AuthenticationTypeUserpool,
		StackName:          appStreamString("core-stack"),
	}); ok {
		t.Fatal("user-stack association with empty user name should be skipped")
	}
	if _, ok := newAppStreamUserStackAssociationResource(appstreamtypes.UserStackAssociation{
		AuthenticationType: appstreamtypes.AuthenticationTypeUserpool,
		UserName:           appStreamString("user@example.com"),
	}); ok {
		t.Fatal("user-stack association with empty stack name should be skipped")
	}
	if _, ok := newAppStreamUserStackAssociationResource(appstreamtypes.UserStackAssociation{
		StackName: appStreamString("core-stack"),
		UserName:  appStreamString("user@example.com"),
	}); ok {
		t.Fatal("user-stack association with empty authentication type should be skipped")
	}
}

func TestAppStreamFleetStackAssociationImportID(t *testing.T) {
	got := appStreamFleetStackAssociationImportID("fleetName", "stackName")
	want := "fleetName/stackName"
	if got != want {
		t.Fatalf("association import ID = %q, want %q", got, want)
	}
}

func TestAppStreamUserImportID(t *testing.T) {
	got := appStreamUserImportID("user@example.com", appstreamtypes.AuthenticationTypeUserpool)
	want := "user@example.com/USERPOOL"
	if got != want {
		t.Fatalf("user import ID = %q, want %q", got, want)
	}
}

func TestAppStreamUserStackAssociationImportID(t *testing.T) {
	got := appStreamUserStackAssociationImportID("user@example.com", appstreamtypes.AuthenticationTypeUserpool, "stackName")
	want := "user@example.com/USERPOOL/stackName"
	if got != want {
		t.Fatalf("user-stack association import ID = %q, want %q", got, want)
	}
}

func TestAppStreamUserStackAssociationsInput(t *testing.T) {
	input, ok := appStreamUserStackAssociationsInput("core-stack")
	if !ok {
		t.Fatal("stack-filtered association input should be built")
	}
	if got := input.AuthenticationType; got != appstreamtypes.AuthenticationTypeUserpool {
		t.Fatalf("authentication type = %q, want %q", got, appstreamtypes.AuthenticationTypeUserpool)
	}
	if got := StringValue(input.StackName); got != "core-stack" {
		t.Fatalf("stack name = %q, want %q", got, "core-stack")
	}
	if input.UserName != nil {
		t.Fatalf("user name = %v, want nil", input.UserName)
	}

	if _, ok := appStreamUserStackAssociationsInput(""); ok {
		t.Fatal("association input with empty stack name should be skipped")
	}
}

func TestAppStreamResourceNamesPreserveSegmentBoundaries(t *testing.T) {
	left, ok := newAppStreamFleetStackAssociationResource("a/b_c", "d")
	if !ok {
		t.Fatal("left association should be importable")
	}
	right, ok := newAppStreamFleetStackAssociationResource("a", "b_c/d")
	if !ok {
		t.Fatal("right association should be importable")
	}
	if left.ResourceName == right.ResourceName {
		t.Fatalf("association resource names collide: %q", left.ResourceName)
	}

	left, ok = newAppStreamUserStackAssociationResource(appstreamtypes.UserStackAssociation{
		AuthenticationType: appstreamtypes.AuthenticationTypeUserpool,
		StackName:          appStreamString("d"),
		UserName:           appStreamString("a/b_c"),
	})
	if !ok {
		t.Fatal("left user-stack association should be importable")
	}
	right, ok = newAppStreamUserStackAssociationResource(appstreamtypes.UserStackAssociation{
		AuthenticationType: appstreamtypes.AuthenticationTypeUserpool,
		StackName:          appStreamString("b_c/d"),
		UserName:           appStreamString("a"),
	})
	if !ok {
		t.Fatal("right user-stack association should be importable")
	}
	if left.ResourceName == right.ResourceName {
		t.Fatalf("user-stack association resource names collide: %q", left.ResourceName)
	}
}

func TestAppStreamNextToken(t *testing.T) {
	if got := appStreamNextToken(nil); got != nil {
		t.Fatalf("nil token = %v, want nil", got)
	}
	if got := appStreamNextToken(appStreamString("")); got != nil {
		t.Fatalf("empty token = %v, want nil", got)
	}
	token := appStreamString("next")
	if got := appStreamNextToken(token); got != token {
		t.Fatalf("next token pointer = %p, want %p", got, token)
	}
}

func TestAppStreamResourceNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "resource not found", err: &appstreamtypes.ResourceNotFoundException{}, want: true},
		{name: "wrapped resource not found", err: errors.Join(errors.New("lookup failed"), &appstreamtypes.ResourceNotFoundException{}), want: true},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appStreamResourceNotFound(tt.err); got != tt.want {
				t.Fatalf("appStreamResourceNotFound(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func assertAppStreamResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantName, wantType string) {
	t.Helper()
	if !ok {
		t.Fatal("resource was skipped")
	}
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

func assertAppStreamAttribute(t *testing.T, resource terraformutils.Resource, key, want string) {
	t.Helper()
	if got := resource.InstanceState.Attributes[key]; got != want {
		t.Fatalf("resource attribute %q = %q, want %q", key, got, want)
	}
}

func appStreamString(value string) *string {
	return &value
}
