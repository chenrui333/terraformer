// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	pinpointsmsvoicev2types "github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestPinpointSMSVoiceV2ImportIDs(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "configuration set", got: pinpointSMSVoiceV2NameImportID("config-set"), want: "config-set"},
		{name: "opt-out list", got: pinpointSMSVoiceV2NameImportID("opt-outs"), want: "opt-outs"},
		{name: "phone number", got: pinpointSMSVoiceV2PhoneNumberImportID("phone-123"), want: "phone-123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("import ID = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestPinpointSMSVoiceV2ResourceNameAvoidsSanitizedCollisions(t *testing.T) {
	first := terraformutils.TfSanitize(pinpointSMSVoiceV2ResourceName("configuration_set", "a_b", "c"))
	second := terraformutils.TfSanitize(pinpointSMSVoiceV2ResourceName("configuration_set", "a", "b_c"))
	if first == second {
		t.Fatalf("pinpointSMSVoiceV2ResourceName() generated duplicate sanitized name %q", first)
	}
}

func TestPinpointSMSVoiceV2OwnedResourceInputs(t *testing.T) {
	if got := pinpointSMSVoiceV2DescribeOptOutListsInput().Owner; got != pinpointsmsvoicev2types.OwnerSelf {
		t.Fatalf("opt-out list owner = %q, want %q", got, pinpointsmsvoicev2types.OwnerSelf)
	}
	if got := pinpointSMSVoiceV2DescribePhoneNumbersInput().Owner; got != pinpointsmsvoicev2types.OwnerSelf {
		t.Fatalf("phone number owner = %q, want %q", got, pinpointsmsvoicev2types.OwnerSelf)
	}
}

func TestNewPinpointSMSVoiceV2ConfigurationSetResource(t *testing.T) {
	resource, ok := newPinpointSMSVoiceV2ConfigurationSetResource(pinpointsmsvoicev2types.ConfigurationSetInformation{
		ConfigurationSetArn:  aws.String("arn:aws:sms-voice:us-east-1:123456789012:configuration-set/config-set"),
		ConfigurationSetName: aws.String("config-set"),
	})
	assertPinpointSMSVoiceV2Resource(t, resource, ok, pinpointSMSVoiceV2ConfigurationSetResourceType, "config-set", map[string]string{
		"arn":  "arn:aws:sms-voice:us-east-1:123456789012:configuration-set/config-set",
		"id":   "config-set",
		"name": "config-set",
	})
	if _, ok := newPinpointSMSVoiceV2ConfigurationSetResource(pinpointsmsvoicev2types.ConfigurationSetInformation{}); ok {
		t.Fatal("configuration set with empty name should be skipped")
	}
}

func TestNewPinpointSMSVoiceV2OptOutListResource(t *testing.T) {
	resource, ok := newPinpointSMSVoiceV2OptOutListResource(pinpointsmsvoicev2types.OptOutListInformation{
		OptOutListArn:  aws.String("arn:aws:sms-voice:us-east-1:123456789012:opt-out-list/opt-outs"),
		OptOutListName: aws.String("opt-outs"),
	})
	assertPinpointSMSVoiceV2Resource(t, resource, ok, pinpointSMSVoiceV2OptOutListResourceType, "opt-outs", map[string]string{
		"arn":  "arn:aws:sms-voice:us-east-1:123456789012:opt-out-list/opt-outs",
		"id":   "opt-outs",
		"name": "opt-outs",
	})
	if _, ok := newPinpointSMSVoiceV2OptOutListResource(pinpointsmsvoicev2types.OptOutListInformation{}); ok {
		t.Fatal("opt-out list with empty name should be skipped")
	}
}

func TestNewPinpointSMSVoiceV2PhoneNumberResource(t *testing.T) {
	resource, ok := newPinpointSMSVoiceV2PhoneNumberResource(pinpointsmsvoicev2types.PhoneNumberInformation{
		IsoCountryCode: aws.String("US"),
		MessageType:    pinpointsmsvoicev2types.MessageTypeTransactional,
		NumberCapabilities: []pinpointsmsvoicev2types.NumberCapability{
			pinpointsmsvoicev2types.NumberCapabilitySms,
		},
		NumberType:     pinpointsmsvoicev2types.NumberTypeTollFree,
		OptOutListName: aws.String("opt-outs"),
		PhoneNumber:    aws.String("+12065550100"),
		PhoneNumberArn: aws.String("arn:aws:sms-voice:us-east-1:123456789012:phone-number/phone-123"),
		PhoneNumberId:  aws.String("phone-123"),
		Status:         pinpointsmsvoicev2types.NumberStatusActive,
	})
	assertPinpointSMSVoiceV2Resource(t, resource, ok, pinpointSMSVoiceV2PhoneNumberResourceType, "phone-123", map[string]string{
		"arn":               "arn:aws:sms-voice:us-east-1:123456789012:phone-number/phone-123",
		"id":                "phone-123",
		"iso_country_code":  "US",
		"message_type":      "TRANSACTIONAL",
		"number_type":       "TOLL_FREE",
		"opt_out_list_name": "opt-outs",
		"phone_number":      "+12065550100",
	})
	if _, ok := newPinpointSMSVoiceV2PhoneNumberResource(pinpointsmsvoicev2types.PhoneNumberInformation{
		PhoneNumberId: aws.String("phone-123"),
		Status:        pinpointsmsvoicev2types.NumberStatusPending,
	}); ok {
		t.Fatal("pending phone number should be skipped")
	}
	if _, ok := newPinpointSMSVoiceV2PhoneNumberResource(pinpointsmsvoicev2types.PhoneNumberInformation{
		IsoCountryCode: aws.String("US"),
		MessageType:    pinpointsmsvoicev2types.MessageTypeTransactional,
		NumberCapabilities: []pinpointsmsvoicev2types.NumberCapability{
			pinpointsmsvoicev2types.NumberCapabilitySms,
		},
		NumberType:    pinpointsmsvoicev2types.NumberTypeShortCode,
		PhoneNumber:   aws.String("12345"),
		PhoneNumberId: aws.String("phone-123"),
		Status:        pinpointsmsvoicev2types.NumberStatusActive,
	}); ok {
		t.Fatal("short-code phone number should be skipped")
	}
	if _, ok := newPinpointSMSVoiceV2PhoneNumberResource(pinpointsmsvoicev2types.PhoneNumberInformation{}); ok {
		t.Fatal("phone number with empty ID should be skipped")
	}
}

func TestPinpointSMSVoiceV2NotFound(t *testing.T) {
	if !pinpointSMSVoiceV2NotFound(&pinpointsmsvoicev2types.ResourceNotFoundException{}) {
		t.Fatal("pinpointSMSVoiceV2NotFound() = false, want true")
	}
	if pinpointSMSVoiceV2NotFound(errors.New("other")) {
		t.Fatal("pinpointSMSVoiceV2NotFound() = true for generic error, want false")
	}
}

func assertPinpointSMSVoiceV2Resource(t *testing.T, resource terraformutils.Resource, ok bool, wantType, wantID string, wantAttrs map[string]string) {
	t.Helper()
	if !ok {
		t.Fatal("resource constructor returned ok=false, want true")
	}
	if resource.InstanceInfo.Type != wantType {
		t.Fatalf("type = %q, want %q", resource.InstanceInfo.Type, wantType)
	}
	if resource.InstanceState.ID != wantID {
		t.Fatalf("ID = %q, want %q", resource.InstanceState.ID, wantID)
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
