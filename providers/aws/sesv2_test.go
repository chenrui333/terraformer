// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	sesv2types "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestSESV2ImportIDs(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "configuration set", got: sesv2ConfigurationSetImportID("config-set"), want: "config-set"},
		{name: "dedicated IP pool", got: sesv2DedicatedIPPoolImportID("pool-a"), want: "pool-a"},
		{name: "email identity", got: sesv2EmailIdentityImportID("sender@example.com"), want: "sender@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("import ID = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestNewSESV2ConfigurationSetResource(t *testing.T) {
	resource, ok := newSESV2ConfigurationSetResource("config-set")
	if !ok {
		t.Fatal("newSESV2ConfigurationSetResource() ok = false, want true")
	}
	assertSESV2Resource(t, resource, sesv2ConfigurationSetResourceType, "config-set", "configuration_set_name", "config-set")

	if _, ok := newSESV2ConfigurationSetResource(""); ok {
		t.Fatal("newSESV2ConfigurationSetResource() ok = true for empty configuration set name, want false")
	}
}

func TestNewSESV2DedicatedIPPoolResource(t *testing.T) {
	resource, ok := newSESV2DedicatedIPPoolResource("pool-a")
	if !ok {
		t.Fatal("newSESV2DedicatedIPPoolResource() ok = false, want true")
	}
	assertSESV2Resource(t, resource, sesv2DedicatedIPPoolResourceType, "pool-a", "pool_name", "pool-a")

	if _, ok := newSESV2DedicatedIPPoolResource(""); ok {
		t.Fatal("newSESV2DedicatedIPPoolResource() ok = true for empty pool name, want false")
	}
}

func TestNewSESV2EmailIdentityResource(t *testing.T) {
	resource, ok := newSESV2EmailIdentityResource(sesv2types.IdentityInfo{
		IdentityName: aws.String("sender@example.com"),
	})
	if !ok {
		t.Fatal("newSESV2EmailIdentityResource() ok = false, want true")
	}
	assertSESV2Resource(t, resource, sesv2EmailIdentityResourceType, "sender@example.com", "email_identity", "sender@example.com")

	if _, ok := newSESV2EmailIdentityResource(sesv2types.IdentityInfo{}); ok {
		t.Fatal("newSESV2EmailIdentityResource() ok = true for empty identity name, want false")
	}
}

func TestSESV2ResourceNameAvoidsSanitizedCollisions(t *testing.T) {
	tests := []struct {
		name   string
		first  []string
		second []string
	}{
		{name: "separator boundary", first: []string{"email_identity", "a_b", "c"}, second: []string{"email_identity", "a", "b_c"}},
		{name: "at sign encoding", first: []string{"email_identity", "a@example.com"}, second: []string{"email_identity", "a-0040-example.com"}},
		{name: "slash encoding", first: []string{"configuration_set", "a/b"}, second: []string{"configuration_set", "a-002F-b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := terraformutils.TfSanitize(sesv2ResourceName(tt.first...))
			second := terraformutils.TfSanitize(sesv2ResourceName(tt.second...))
			if first == second {
				t.Fatalf("sesv2ResourceName() generated duplicate sanitized names %q", first)
			}
		})
	}
}

func assertSESV2Resource(t *testing.T, resource terraformutils.Resource, resourceType, resourceID, attributeName, attributeValue string) {
	t.Helper()

	if resource.InstanceInfo.Type != resourceType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, resourceType)
	}
	if resource.InstanceState.ID != resourceID {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, resourceID)
	}
	if got := resource.InstanceState.Attributes[attributeName]; got != attributeValue {
		t.Fatalf("attribute %q = %q, want %q", attributeName, got, attributeValue)
	}
	wantName := terraformutils.TfSanitize(sesv2ResourceName(stringsForSESV2ResourceName(resourceType, attributeValue)...))
	if resource.ResourceName != wantName {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, wantName)
	}
}

func stringsForSESV2ResourceName(resourceType, name string) []string {
	switch resourceType {
	case sesv2ConfigurationSetResourceType:
		return []string{"configuration_set", name}
	case sesv2DedicatedIPPoolResourceType:
		return []string{"dedicated_ip_pool", name}
	case sesv2EmailIdentityResourceType:
		return []string{"email_identity", name}
	default:
		return []string{name}
	}
}
