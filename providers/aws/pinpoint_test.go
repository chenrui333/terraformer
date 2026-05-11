// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	pinpointtypes "github.com/aws/aws-sdk-go-v2/service/pinpoint/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestPinpointImportIDs(t *testing.T) {
	if got, want := pinpointApplicationImportID("app-123"), "app-123"; got != want {
		t.Fatalf("pinpointApplicationImportID() = %q, want %q", got, want)
	}
}

func TestPinpointResourceNameAvoidsSanitizedCollisions(t *testing.T) {
	first := terraformutils.TfSanitize(pinpointResourceName("email_channel", "a_b", "c"))
	second := terraformutils.TfSanitize(pinpointResourceName("email_channel", "a", "b_c"))
	if first == second {
		t.Fatalf("pinpointResourceName() generated duplicate sanitized name %q", first)
	}
}

func TestNewPinpointAppResource(t *testing.T) {
	resource, ok := newPinpointAppResource(pinpointtypes.ApplicationResponse{
		Id:   aws.String("app-123"),
		Name: aws.String("engagement"),
	})
	assertPinpointResource(t, resource, ok, pinpointAppResourceType, map[string]string{
		"application_id": "app-123",
		"name":           "engagement",
	})
	if _, ok := newPinpointAppResource(pinpointtypes.ApplicationResponse{}); ok {
		t.Fatal("app with empty ID should be skipped")
	}
}

func TestNewPinpointEmailChannelResource(t *testing.T) {
	resource, ok := newPinpointEmailChannelResource("app-123", "engagement", pinpointtypes.EmailChannelResponse{
		Enabled:     aws.Bool(true),
		FromAddress: aws.String("sender@example.com"),
		Identity:    aws.String("arn:aws:ses:us-east-1:123456789012:identity/example.com"),
	})
	assertPinpointResource(t, resource, ok, pinpointEmailChannelResourceType, map[string]string{
		"application_id": "app-123",
		"enabled":        "true",
		"from_address":   "sender@example.com",
		"identity":       "arn:aws:ses:us-east-1:123456789012:identity/example.com",
	})
	if _, ok := newPinpointEmailChannelResource("app-123", "engagement", pinpointtypes.EmailChannelResponse{}); ok {
		t.Fatal("email channel without from address or identity should be skipped")
	}
	if _, ok := newPinpointEmailChannelResource("", "engagement", pinpointtypes.EmailChannelResponse{
		FromAddress: aws.String("sender@example.com"),
		Identity:    aws.String("arn:aws:ses:us-east-1:123456789012:identity/example.com"),
	}); ok {
		t.Fatal("email channel with empty application ID should be skipped")
	}
}

func TestNewPinpointSMSChannelResource(t *testing.T) {
	resource, ok := newPinpointSMSChannelResource("app-123", "engagement", pinpointtypes.SMSChannelResponse{
		Enabled:   aws.Bool(false),
		SenderId:  aws.String("Example"),
		ShortCode: aws.String("12345"),
	})
	assertPinpointResource(t, resource, ok, pinpointSMSChannelResourceType, map[string]string{
		"application_id": "app-123",
		"enabled":        "false",
		"sender_id":      "Example",
		"short_code":     "12345",
	})
	if _, ok := newPinpointSMSChannelResource("", "engagement", pinpointtypes.SMSChannelResponse{}); ok {
		t.Fatal("SMS channel with empty application ID should be skipped")
	}
}

func TestNewPinpointEventStreamResource(t *testing.T) {
	resource, ok := newPinpointEventStreamResource("app-123", "engagement", pinpointtypes.EventStream{
		DestinationStreamArn: aws.String("arn:aws:kinesis:us-east-1:123456789012:stream/pinpoint-events"),
		RoleArn:              aws.String("arn:aws:iam::123456789012:role/pinpoint-events"),
	})
	assertPinpointResource(t, resource, ok, pinpointEventStreamResourceType, map[string]string{
		"application_id":         "app-123",
		"destination_stream_arn": "arn:aws:kinesis:us-east-1:123456789012:stream/pinpoint-events",
		"role_arn":               "arn:aws:iam::123456789012:role/pinpoint-events",
	})
	if _, ok := newPinpointEventStreamResource("app-123", "engagement", pinpointtypes.EventStream{}); ok {
		t.Fatal("event stream without destination or role should be skipped")
	}
}

func TestPinpointNotFound(t *testing.T) {
	if !pinpointNotFound(&pinpointtypes.NotFoundException{}) {
		t.Fatal("pinpointNotFound() = false, want true")
	}
	if pinpointNotFound(errors.New("other")) {
		t.Fatal("pinpointNotFound() = true for generic error, want false")
	}
}

func assertPinpointResource(t *testing.T, resource terraformutils.Resource, ok bool, wantType string, wantAttrs map[string]string) {
	t.Helper()
	if !ok {
		t.Fatal("resource constructor returned ok=false, want true")
	}
	if resource.InstanceInfo.Type != wantType {
		t.Fatalf("type = %q, want %q", resource.InstanceInfo.Type, wantType)
	}
	if resource.InstanceState.ID != "app-123" {
		t.Fatalf("ID = %q, want %q", resource.InstanceState.ID, "app-123")
	}
	for key, want := range wantAttrs {
		if got := resource.InstanceState.Attributes[key]; got != want {
			t.Fatalf("attribute %q = %q, want %q", key, got, want)
		}
	}
	if len(resource.AllowEmptyValues) != 1 || resource.AllowEmptyValues[0] != "tags." {
		t.Fatalf("AllowEmptyValues = %#v, want tags", resource.AllowEmptyValues)
	}
}
