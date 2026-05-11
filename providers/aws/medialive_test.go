// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	medialivetypes "github.com/aws/aws-sdk-go-v2/service/medialive/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewMediaLiveMultiplexResource(t *testing.T) {
	for _, state := range []medialivetypes.MultiplexState{
		"",
		medialivetypes.MultiplexStateCreating,
		medialivetypes.MultiplexStateCreateFailed,
		medialivetypes.MultiplexStateIdle,
		medialivetypes.MultiplexStateStarting,
		medialivetypes.MultiplexStateRunning,
		medialivetypes.MultiplexStateRecovering,
		medialivetypes.MultiplexStateStopping,
	} {
		resource, ok := newMediaLiveMultiplexResource(medialivetypes.MultiplexSummary{
			Id:    aws.String("multiplex-123"),
			Name:  aws.String("core-multiplex"),
			State: state,
		})
		assertMediaLiveResource(t, resource, ok, "multiplex-123", mediaLiveResourceName("multiplex", "core-multiplex", "multiplex-123"), mediaLiveMultiplexResourceType)
	}

	if _, ok := newMediaLiveMultiplexResource(medialivetypes.MultiplexSummary{
		Name:  aws.String("core-multiplex"),
		State: medialivetypes.MultiplexStateIdle,
	}); ok {
		t.Fatal("multiplex with empty ID should be skipped")
	}

	for _, state := range []medialivetypes.MultiplexState{
		medialivetypes.MultiplexStateDeleting,
		medialivetypes.MultiplexStateDeleted,
	} {
		if _, ok := newMediaLiveMultiplexResource(medialivetypes.MultiplexSummary{
			Id:    aws.String("multiplex-123"),
			Name:  aws.String("core-multiplex"),
			State: state,
		}); ok {
			t.Fatalf("multiplex in %q state should be skipped", state)
		}
	}
}

func TestNewMediaLiveMultiplexProgramResource(t *testing.T) {
	resource, ok := newMediaLiveMultiplexProgramResource("multiplex-123", medialivetypes.MultiplexProgramSummary{
		ProgramName: aws.String("core-program"),
	})
	assertMediaLiveResource(t, resource, ok, "core-program/multiplex-123", mediaLiveResourceName("multiplex-program", "multiplex-123", "core-program"), mediaLiveMultiplexProgramResourceType)
	assertMediaLiveAttribute(t, resource, "multiplex_id", "multiplex-123")
	assertMediaLiveAttribute(t, resource, "program_name", "core-program")

	if _, ok := newMediaLiveMultiplexProgramResource("", medialivetypes.MultiplexProgramSummary{ProgramName: aws.String("core-program")}); ok {
		t.Fatal("multiplex program with empty multiplex ID should be skipped")
	}
	if _, ok := newMediaLiveMultiplexProgramResource("multiplex-123", medialivetypes.MultiplexProgramSummary{}); ok {
		t.Fatal("multiplex program with empty program name should be skipped")
	}
}

func TestMediaLiveMultiplexImportID(t *testing.T) {
	got := mediaLiveMultiplexImportID("multiplex-123")
	want := "multiplex-123"
	if got != want {
		t.Fatalf("multiplex import ID = %q, want %q", got, want)
	}
}

func TestMediaLiveMultiplexProgramImportID(t *testing.T) {
	got := mediaLiveMultiplexProgramImportID("core-program", "multiplex-123")
	want := "core-program/multiplex-123"
	if got != want {
		t.Fatalf("multiplex program import ID = %q, want %q", got, want)
	}
}

func TestMediaLiveMultiplexStateImportable(t *testing.T) {
	tests := []struct {
		name  string
		state medialivetypes.MultiplexState
		want  bool
	}{
		{name: "empty", want: true},
		{name: "creating", state: medialivetypes.MultiplexStateCreating, want: true},
		{name: "create failed", state: medialivetypes.MultiplexStateCreateFailed, want: true},
		{name: "idle", state: medialivetypes.MultiplexStateIdle, want: true},
		{name: "starting", state: medialivetypes.MultiplexStateStarting, want: true},
		{name: "running", state: medialivetypes.MultiplexStateRunning, want: true},
		{name: "recovering", state: medialivetypes.MultiplexStateRecovering, want: true},
		{name: "stopping", state: medialivetypes.MultiplexStateStopping, want: true},
		{name: "deleting", state: medialivetypes.MultiplexStateDeleting, want: false},
		{name: "deleted", state: medialivetypes.MultiplexStateDeleted, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mediaLiveMultiplexStateImportable(tt.state); got != tt.want {
				t.Fatalf("mediaLiveMultiplexStateImportable(%q) = %t, want %t", tt.state, got, tt.want)
			}
		})
	}
}

func TestMediaLiveResourceNamesPreserveSegmentBoundaries(t *testing.T) {
	left := terraformutils.TfSanitize(mediaLiveResourceName("multiplex-program", "a/b_c", "d"))
	right := terraformutils.TfSanitize(mediaLiveResourceName("multiplex", "program/a", "b_c/d"))
	if left == right {
		t.Fatalf("MediaLive resource names collide: %q", left)
	}
}

func TestMediaLiveResourceNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "not found", err: &medialivetypes.NotFoundException{}, want: true},
		{name: "wrapped not found", err: errors.Join(errors.New("lookup failed"), &medialivetypes.NotFoundException{}), want: true},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mediaLiveResourceNotFound(tt.err); got != tt.want {
				t.Fatalf("mediaLiveResourceNotFound(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func assertMediaLiveResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantName, wantType string) {
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

func assertMediaLiveAttribute(t *testing.T, resource terraformutils.Resource, key, want string) {
	t.Helper()
	if got := resource.InstanceState.Attributes[key]; got != want {
		t.Fatalf("resource attribute %q = %q, want %q", key, got, want)
	}
}
