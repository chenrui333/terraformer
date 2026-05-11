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

func TestAppStreamFleetStackAssociationImportID(t *testing.T) {
	got := appStreamFleetStackAssociationImportID("fleetName", "stackName")
	want := "fleetName/stackName"
	if got != want {
		t.Fatalf("association import ID = %q, want %q", got, want)
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

func appStreamString(value string) *string {
	return &value
}
