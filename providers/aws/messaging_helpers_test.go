// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestMessagingResourceNameWithLengthPrefixesAvoidsSanitizedCollisions(t *testing.T) {
	tests := []struct {
		name   string
		first  []string
		second []string
	}{
		{name: "separator boundary", first: []string{"resource", "a_b", "c"}, second: []string{"resource", "a", "b_c"}},
		{name: "slash encoding", first: []string{"resource", "a/b"}, second: []string{"resource", "a-002F-b"}},
		{name: "colon encoding", first: []string{"resource", "a:b"}, second: []string{"resource", "a-003A-b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := terraformutils.TfSanitize(resourceNameWithLengthPrefixes(tt.first...))
			second := terraformutils.TfSanitize(resourceNameWithLengthPrefixes(tt.second...))
			if first == second {
				t.Fatalf("resourceNameWithLengthPrefixes() generated duplicate sanitized name %q", first)
			}
		})
	}
}

func TestStringSliceToInterfaceSlice(t *testing.T) {
	got := stringSliceToInterfaceSlice([]string{"us-east-1", "", "us-west-2"})
	want := []interface{}{"us-east-1", "us-west-2"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("value %d = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func assertMessagingResource(t *testing.T, resource terraformutils.Resource, ok bool, wantType, wantID string, wantAttrs map[string]string) {
	t.Helper()
	if !ok {
		t.Fatal("resource ok = false, want true")
	}
	if resource.InstanceInfo == nil {
		t.Fatal("resource InstanceInfo is nil")
	}
	if resource.InstanceInfo.Type != wantType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, wantType)
	}
	if resource.InstanceState == nil {
		t.Fatal("resource InstanceState is nil")
	}
	if resource.InstanceState.ID != wantID {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, wantID)
	}
	for key, want := range wantAttrs {
		if got := resource.InstanceState.Attributes[key]; got != want {
			t.Fatalf("attribute %s = %q, want %q", key, got, want)
		}
	}
}
