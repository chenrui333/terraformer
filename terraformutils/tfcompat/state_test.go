// SPDX-License-Identifier: Apache-2.0

package tfcompat

import (
	"encoding/json"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestInstanceInfoResourceAddress(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		info *InstanceInfo
		want string
	}{
		{
			name: "fully qualified import id",
			info: &InstanceInfo{Type: "aws_vpc", Id: "aws_vpc.tfer--main"},
			want: "aws_vpc.tfer--main",
		},
		{
			name: "raw provider id fallback",
			info: &InstanceInfo{Type: "aws_vpc", Id: "vpc-123456"},
			want: "aws_vpc.vpc-123456",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.info.ResourceAddress().String(); got != tc.want {
				t.Fatalf("ResourceAddress() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResourceAddressStringNil(t *testing.T) {
	t.Parallel()

	var address *ResourceAddress
	if got := address.String(); got != "" {
		t.Fatalf("nil ResourceAddress.String() = %q, want empty string", got)
	}
}

func TestNewInstanceStateShimmedFromValue(t *testing.T) {
	t.Parallel()

	state := NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
		"id":   cty.StringVal("vpc-123"),
		"name": cty.StringVal("main"),
	}), 7)

	if state.ID != "vpc-123" {
		t.Fatalf("ID = %q, want %q", state.ID, "vpc-123")
	}
	if got := state.Attributes["name"]; got != "main" {
		t.Fatalf("name attribute = %q, want %q", got, "main")
	}
	if got := state.Meta["schema_version"]; got != 7 {
		t.Fatalf("schema_version = %#v, want 7", got)
	}

	var typed map[string]interface{}
	if err := json.Unmarshal(state.TypedAttributes, &typed); err != nil {
		t.Fatal(err)
	}
	if got := typed["id"].(string); got != "vpc-123" {
		t.Fatalf("typed id attribute = %q, want %q", got, "vpc-123")
	}
	if got := typed["name"].(string); got != "main" {
		t.Fatalf("typed name attribute = %q, want %q", got, "main")
	}
}

func TestInstanceStateAttrsAsObjectValue(t *testing.T) {
	t.Parallel()

	state := &InstanceState{
		ID: "vpc-123",
		Attributes: map[string]string{
			"name": "main",
		},
	}
	objectType := cty.Object(map[string]cty.Type{
		"id":   cty.String,
		"name": cty.String,
	})

	value, err := state.AttrsAsObjectValue(objectType)
	if err != nil {
		t.Fatal(err)
	}
	if got := value.GetAttr("id").AsString(); got != "vpc-123" {
		t.Fatalf("id attribute = %q, want %q", got, "vpc-123")
	}
	if got := value.GetAttr("name").AsString(); got != "main" {
		t.Fatalf("name attribute = %q, want %q", got, "main")
	}
}
