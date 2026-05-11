// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ivschat"
	ivschattypes "github.com/aws/aws-sdk-go-v2/service/ivschat/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	testIvsChatLoggingConfigurationARN = "arn:aws:ivschat:us-east-1:123456789012:logging-configuration/abc123"
	testIvsChatRoomARN                 = "arn:aws:ivschat:us-east-1:123456789012:room/def456"
)

func TestNewIvsChatLoggingConfigurationResource(t *testing.T) {
	configuration := validIvsChatLoggingConfiguration("chat-logs")
	resource, ok := newIvsChatLoggingConfigurationResource(configuration)
	assertIvsChatResource(t, resource, ok, testIvsChatLoggingConfigurationARN, ivsChatResourceName("logging-configuration", "chat-logs", testIvsChatLoggingConfigurationARN), ivsChatLoggingConfigurationResourceType)
}

func TestNewIvsChatLoggingConfigurationResourceSkipsIncompleteConfigurations(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ivschat.GetLoggingConfigurationOutput)
	}{
		{name: "empty ARN", mutate: func(configuration *ivschat.GetLoggingConfigurationOutput) { configuration.Arn = nil }},
		{name: "empty state", mutate: func(configuration *ivschat.GetLoggingConfigurationOutput) { configuration.State = "" }},
		{name: "deleting state", mutate: func(configuration *ivschat.GetLoggingConfigurationOutput) {
			configuration.State = ivschattypes.LoggingConfigurationStateDeleting
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configuration := validIvsChatLoggingConfiguration("chat-logs")
			tt.mutate(configuration)
			if _, ok := newIvsChatLoggingConfigurationResource(configuration); ok {
				t.Fatal("incomplete logging configuration should be skipped")
			}
		})
	}

	if _, ok := newIvsChatLoggingConfigurationResource(nil); ok {
		t.Fatal("nil logging configuration should be skipped")
	}
}

func TestNewIvsChatRoomResource(t *testing.T) {
	resource, ok := newIvsChatRoomResource(&ivschat.GetRoomOutput{
		Arn:  aws.String(testIvsChatRoomARN),
		Name: aws.String("support-room"),
	})
	assertIvsChatResource(t, resource, ok, testIvsChatRoomARN, ivsChatResourceName("room", "support-room", testIvsChatRoomARN), ivsChatRoomResourceType)

	if _, ok := newIvsChatRoomResource(nil); ok {
		t.Fatal("nil room should be skipped")
	}
	if _, ok := newIvsChatRoomResource(&ivschat.GetRoomOutput{Name: aws.String("support-room")}); ok {
		t.Fatal("room with empty ARN should be skipped")
	}
}

func TestIvsChatARNImportID(t *testing.T) {
	if got := ivsChatARNImportID(testIvsChatRoomARN); got != testIvsChatRoomARN {
		t.Fatalf("IVS Chat ARN import ID = %q, want %q", got, testIvsChatRoomARN)
	}
}

func TestIvsChatLoggingConfigurationStateImportable(t *testing.T) {
	tests := []struct {
		name  string
		state ivschattypes.LoggingConfigurationState
		want  bool
	}{
		{name: "creating", state: ivschattypes.LoggingConfigurationStateCreating, want: true},
		{name: "create failed", state: ivschattypes.LoggingConfigurationStateCreateFailed, want: true},
		{name: "delete failed", state: ivschattypes.LoggingConfigurationStateDeleteFailed, want: true},
		{name: "updating", state: ivschattypes.LoggingConfigurationStateUpdating, want: true},
		{name: "update failed", state: ivschattypes.LoggingConfigurationStateUpdateFailed, want: true},
		{name: "active", state: ivschattypes.LoggingConfigurationStateActive, want: true},
		{name: "empty", want: false},
		{name: "deleting", state: ivschattypes.LoggingConfigurationStateDeleting, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ivsChatLoggingConfigurationStateImportable(tt.state); got != tt.want {
				t.Fatalf("ivsChatLoggingConfigurationStateImportable(%q) = %t, want %t", tt.state, got, tt.want)
			}
		})
	}
}

func TestIvsChatResourceNamesPreserveSegmentBoundaries(t *testing.T) {
	left := terraformutils.TfSanitize(ivsChatResourceName("room", "a/b_c", "d"))
	right := terraformutils.TfSanitize(ivsChatResourceName("room", "a", "b_c/d"))
	if left == right {
		t.Fatalf("IVS Chat resource names collide: %q", left)
	}
}

func TestIvsChatResourceNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "resource not found", err: &ivschattypes.ResourceNotFoundException{}, want: true},
		{name: "wrapped resource not found", err: errors.Join(errors.New("lookup failed"), &ivschattypes.ResourceNotFoundException{}), want: true},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ivsChatResourceNotFound(tt.err); got != tt.want {
				t.Fatalf("ivsChatResourceNotFound(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func validIvsChatLoggingConfiguration(name string) *ivschat.GetLoggingConfigurationOutput {
	return &ivschat.GetLoggingConfigurationOutput{
		Arn:   aws.String(testIvsChatLoggingConfigurationARN),
		Name:  aws.String(name),
		State: ivschattypes.LoggingConfigurationStateActive,
	}
}

func assertIvsChatResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantName, wantType string) {
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
