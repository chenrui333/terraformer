// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	mediastoretypes "github.com/aws/aws-sdk-go-v2/service/mediastore/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewMediaStoreContainerPolicyResource(t *testing.T) {
	policy := "{\"Version\":\"2012-10-17\"}"
	resource, ok := newMediaStoreContainerPolicyResource("media-container", policy)
	assertMediaStoreResource(t, resource, ok, "media-container", mediaStoreResourceName("container-policy", "media-container"), mediaStoreContainerPolicyResourceType)
	assertMediaStoreAttribute(t, resource, "container_name", "media-container")
	assertMediaStoreAttribute(t, resource, "policy", policy)

	if _, ok := newMediaStoreContainerPolicyResource("", policy); ok {
		t.Fatal("container policy with empty container name should be skipped")
	}
	if _, ok := newMediaStoreContainerPolicyResource("media-container", ""); ok {
		t.Fatal("container policy with empty policy should be skipped")
	}
}

func TestMediaStoreContainerPolicyImportID(t *testing.T) {
	if got := mediaStoreContainerPolicyImportID("media-container"); got != "media-container" {
		t.Fatalf("container policy import ID = %q, want %q", got, "media-container")
	}
}

func TestMediaStoreResourceNamesPreserveSegmentBoundaries(t *testing.T) {
	left := terraformutils.TfSanitize(mediaStoreResourceName("container-policy", "a/b_c"))
	right := terraformutils.TfSanitize(mediaStoreResourceName("container", "policy/a_b_c"))
	if left == right {
		t.Fatalf("MediaStore resource names collide: %q", left)
	}
}

func TestMediaStoreContainerPolicyNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "container not found", err: &mediastoretypes.ContainerNotFoundException{}, want: true},
		{name: "policy not found", err: &mediastoretypes.PolicyNotFoundException{}, want: true},
		{name: "wrapped policy not found", err: errors.Join(errors.New("lookup failed"), &mediastoretypes.PolicyNotFoundException{}), want: true},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mediaStoreContainerPolicyNotFound(tt.err); got != tt.want {
				t.Fatalf("mediaStoreContainerPolicyNotFound(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func TestMediaStorePostConvertHookWrapsContainerPolicy(t *testing.T) {
	policy := "{\"Resource\":\"$" + "{aws:username}\"}"
	resource, ok := newMediaStoreContainerPolicyResource("media-container", policy)
	if !ok {
		t.Fatal("container policy should be importable")
	}
	resource.Item = map[string]interface{}{
		"policy": policy,
	}
	generator := MediaStoreGenerator{}
	generator.Resources = []terraformutils.Resource{resource}
	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}
	got, ok := generator.Resources[0].Item["policy"].(string)
	if !ok {
		t.Fatalf("policy type = %T, want string", generator.Resources[0].Item["policy"])
	}
	want := "<<POLICY\n{\"Resource\":\"$" + "$" + "{aws:username}\"}\nPOLICY"
	if got != want {
		t.Fatalf("wrapped policy = %q, want %q", got, want)
	}
}

func assertMediaStoreResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantName, wantType string) {
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

func assertMediaStoreAttribute(t *testing.T, resource terraformutils.Resource, key, want string) {
	t.Helper()
	if got := resource.InstanceState.Attributes[key]; got != want {
		t.Fatalf("resource attribute %q = %q, want %q", key, got, want)
	}
}
