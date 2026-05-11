// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	mediapackagev2types "github.com/aws/aws-sdk-go-v2/service/mediapackagev2/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewMediaPackageV2ChannelGroupResource(t *testing.T) {
	resource, ok := newMediaPackageV2ChannelGroupResource(mediapackagev2types.ChannelGroupListConfiguration{
		ChannelGroupName: aws.String("core-group"),
	})
	assertMediaPackageV2Resource(t, resource, ok, "core-group", mediaPackageV2ResourceName("channel-group", "core-group"), mediaPackageV2ChannelGroupResourceType)

	if _, ok := newMediaPackageV2ChannelGroupResource(mediapackagev2types.ChannelGroupListConfiguration{}); ok {
		t.Fatal("channel group with empty name should be skipped")
	}
}

func TestMediaPackageV2ChannelGroupImportID(t *testing.T) {
	got := mediaPackageV2ChannelGroupImportID("core-group")
	want := "core-group"
	if got != want {
		t.Fatalf("channel group import ID = %q, want %q", got, want)
	}
}

func TestMediaPackageV2ResourceNamesPreserveSegmentBoundaries(t *testing.T) {
	left := terraformutils.TfSanitize(mediaPackageV2ResourceName("channel-group", "a/b_c"))
	right := terraformutils.TfSanitize(mediaPackageV2ResourceName("channel", "group/a_b_c"))
	if left == right {
		t.Fatalf("MediaPackage v2 resource names collide: %q", left)
	}
}

func TestMediaPackageV2ResourceNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "resource not found", err: &mediapackagev2types.ResourceNotFoundException{}, want: true},
		{name: "wrapped resource not found", err: errors.Join(errors.New("lookup failed"), &mediapackagev2types.ResourceNotFoundException{}), want: true},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mediaPackageV2ResourceNotFound(tt.err); got != tt.want {
				t.Fatalf("mediaPackageV2ResourceNotFound(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func assertMediaPackageV2Resource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantName, wantType string) {
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
