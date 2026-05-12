// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	mqtypes "github.com/aws/aws-sdk-go-v2/service/mq/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewMQConfigurationResource(t *testing.T) {
	resource, ok := newMQConfigurationResource(mqtypes.Configuration{
		Id:   aws.String("c-12345678-1234-1234-1234-123456789012"),
		Name: aws.String("rabbitmq-config"),
	})
	assertMQResource(t, resource, ok, "c-12345678-1234-1234-1234-123456789012", mqResourceName("configuration", "rabbitmq-config", "c-12345678-1234-1234-1234-123456789012"), mqConfigurationResourceType)

	if _, ok := newMQConfigurationResource(mqtypes.Configuration{Name: aws.String("missing-id")}); ok {
		t.Fatal("configuration with empty ID should be skipped")
	}
}

func TestMQImportIDs(t *testing.T) {
	if got, want := mqConfigurationImportID("c-123"), "c-123"; got != want {
		t.Fatalf("MQ configuration import ID = %q, want %q", got, want)
	}
}

func TestMQResourceNamesPreserveSegmentBoundaries(t *testing.T) {
	left := terraformutils.TfSanitize(mqResourceName("configuration", "a/b_c"))
	right := terraformutils.TfSanitize(mqResourceName("config", "uration/a_b_c"))
	if left == right {
		t.Fatalf("MQ resource names collide: %q", left)
	}
}

func assertMQResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantName, wantType string) {
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
