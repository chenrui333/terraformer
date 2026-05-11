// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	mediaconverttypes "github.com/aws/aws-sdk-go-v2/service/mediaconvert/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewMediaConvertQueueResource(t *testing.T) {
	resource, ok := newMediaConvertQueueResource(mediaconverttypes.Queue{
		Name: aws.String("transcode"),
		Type: mediaconverttypes.Type("CUSTOM"),
	})
	assertMediaConvertResource(t, resource, ok, "transcode", mediaConvertResourceName("queue", "transcode"), mediaConvertQueueResourceType)
	assertMediaConvertAttribute(t, resource, "name", "transcode")

	if _, ok := newMediaConvertQueueResource(mediaconverttypes.Queue{}); ok {
		t.Fatal("queue with empty name should be skipped")
	}
}

func TestMediaConvertQueueImportable(t *testing.T) {
	tests := []struct {
		name  string
		queue mediaconverttypes.Queue
		want  bool
	}{
		{name: "custom", queue: mediaconverttypes.Queue{Type: mediaconverttypes.Type("CUSTOM")}, want: true},
		{name: "empty type", queue: mediaconverttypes.Queue{}, want: true},
		{name: "system", queue: mediaconverttypes.Queue{Type: mediaconverttypes.TypeSystem}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mediaConvertQueueImportable(tt.queue); got != tt.want {
				t.Fatalf("mediaConvertQueueImportable(%#v) = %t, want %t", tt.queue, got, tt.want)
			}
		})
	}
}

func TestMediaConvertQueueImportID(t *testing.T) {
	if got, want := mediaConvertQueueImportID("transcode"), "transcode"; got != want {
		t.Fatalf("MediaConvert queue import ID = %q, want %q", got, want)
	}
}

func TestMediaConvertResourceNamesPreserveSegmentBoundaries(t *testing.T) {
	left := terraformutils.TfSanitize(mediaConvertResourceName("queue", "a/b_c"))
	right := terraformutils.TfSanitize(mediaConvertResourceName("que", "ue/a_b_c"))
	if left == right {
		t.Fatalf("MediaConvert resource names collide: %q", left)
	}
}

func TestMediaConvertQueueNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "not found", err: &mediaconverttypes.NotFoundException{}, want: true},
		{name: "wrapped not found", err: errors.Join(errors.New("lookup failed"), &mediaconverttypes.NotFoundException{}), want: true},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mediaConvertQueueNotFound(tt.err); got != tt.want {
				t.Fatalf("mediaConvertQueueNotFound(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func TestMediaConvertQueueDiscoverySkippable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "not found", err: &mediaconverttypes.NotFoundException{}, want: true},
		{name: "customer endpoint bad request", err: &mediaconverttypes.BadRequestException{Message: aws.String("You must use the customer-specific endpoint")}, want: true},
		{name: "account endpoint bad request", err: &mediaconverttypes.BadRequestException{Message: aws.String("account endpoint is not available")}, want: true},
		{name: "wrapped endpoint bad request", err: errors.Join(errors.New("list queues failed"), &mediaconverttypes.BadRequestException{Message: aws.String("use the account-specific endpoint")}), want: true},
		{name: "other bad request", err: &mediaconverttypes.BadRequestException{Message: aws.String("invalid maxResults")}, want: false},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mediaConvertQueueDiscoverySkippable(tt.err); got != tt.want {
				t.Fatalf("mediaConvertQueueDiscoverySkippable(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func assertMediaConvertResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantName, wantType string) {
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

func assertMediaConvertAttribute(t *testing.T, resource terraformutils.Resource, key, want string) {
	t.Helper()
	if got := resource.InstanceState.Attributes[key]; got != want {
		t.Fatalf("resource attribute %q = %q, want %q", key, got, want)
	}
}
