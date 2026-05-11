// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
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
		{name: "configuration set event destination", got: sesv2ConfigurationSetEventDestinationImportID("config-set", "events"), want: "config-set|events"},
		{name: "contact list", got: sesv2ContactListImportID("contacts"), want: "contacts"},
		{name: "dedicated IP pool", got: sesv2DedicatedIPPoolImportID("pool-a"), want: "pool-a"},
		{name: "email identity", got: sesv2EmailIdentityImportID("sender@example.com"), want: "sender@example.com"},
		{name: "email identity feedback attributes", got: sesv2EmailIdentityFeedbackAttributesImportID("sender@example.com"), want: "sender@example.com"},
		{name: "email identity mail from attributes", got: sesv2EmailIdentityMailFromAttributesImportID("sender@example.com"), want: "sender@example.com"},
		{name: "email identity policy", got: sesv2EmailIdentityPolicyImportID("sender@example.com", "policy-a"), want: "sender@example.com|policy-a"},
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

func TestNewSESV2ConfigurationSetEventDestinationResource(t *testing.T) {
	resource, ok := newSESV2ConfigurationSetEventDestinationResource("config-set", sesv2types.EventDestination{
		Name: aws.String("events"),
	})
	if !ok {
		t.Fatal("newSESV2ConfigurationSetEventDestinationResource() ok = false, want true")
	}
	assertSESV2ResourceAttributes(t, resource, sesv2ConfigurationSetEventDestinationResourceType, "config-set|events",
		[]string{"configuration_set_event_destination", "config-set", "events"},
		map[string]string{
			"configuration_set_name": "config-set",
			"event_destination_name": "events",
		})

	if _, ok := newSESV2ConfigurationSetEventDestinationResource("", sesv2types.EventDestination{Name: aws.String("events")}); ok {
		t.Fatal("newSESV2ConfigurationSetEventDestinationResource() ok = true for empty configuration set name, want false")
	}
	if _, ok := newSESV2ConfigurationSetEventDestinationResource("config-set", sesv2types.EventDestination{}); ok {
		t.Fatal("newSESV2ConfigurationSetEventDestinationResource() ok = true for empty event destination name, want false")
	}
}

func TestNewSESV2ContactListResource(t *testing.T) {
	resource, ok := newSESV2ContactListResource(&sesv2.GetContactListOutput{
		ContactListName: aws.String("contacts"),
		Description:     aws.String("primary contact list"),
	})
	if !ok {
		t.Fatal("newSESV2ContactListResource() ok = false, want true")
	}
	assertSESV2ResourceAttributes(t, resource, sesv2ContactListResourceType, "contacts",
		[]string{"contact_list", "contacts"},
		map[string]string{
			"contact_list_name": "contacts",
			"description":       "primary contact list",
		})

	resource, ok = newSESV2ContactListResource(&sesv2.GetContactListOutput{
		ContactListName: aws.String("contacts"),
	})
	if !ok {
		t.Fatal("newSESV2ContactListResource() ok = false without description, want true")
	}
	if _, ok := resource.InstanceState.Attributes["description"]; ok {
		t.Fatal("newSESV2ContactListResource() seeded empty description, want omitted")
	}
	if _, ok := resource.AdditionalFields["topic"]; ok {
		t.Fatal("newSESV2ContactListResource() seeded empty topics, want omitted")
	}

	resource, ok = newSESV2ContactListResource(&sesv2.GetContactListOutput{
		ContactListName: aws.String("contacts"),
		Topics: []sesv2types.Topic{
			{
				DefaultSubscriptionStatus: sesv2types.SubscriptionStatusOptOut,
				Description:               aws.String("announcements"),
				DisplayName:               aws.String("Announcements"),
				TopicName:                 aws.String("announcements"),
			},
			{
				DefaultSubscriptionStatus: sesv2types.SubscriptionStatusOptIn,
				DisplayName:               aws.String("News"),
				TopicName:                 aws.String("news"),
			},
		},
	})
	if !ok {
		t.Fatal("newSESV2ContactListResource() ok = false with topics, want true")
	}
	assertSESV2ContactListTopics(t, resource, []map[string]interface{}{
		{
			"default_subscription_status": "OPT_OUT",
			"description":                 "announcements",
			"display_name":                "Announcements",
			"topic_name":                  "announcements",
		},
		{
			"default_subscription_status": "OPT_IN",
			"display_name":                "News",
			"topic_name":                  "news",
		},
	})

	if _, ok := newSESV2ContactListResource(&sesv2.GetContactListOutput{
		ContactListName: aws.String("contacts"),
		Topics: []sesv2types.Topic{
			{
				DefaultSubscriptionStatus: sesv2types.SubscriptionStatusOptIn,
				DisplayName:               aws.String("News"),
			},
		},
	}); ok {
		t.Fatal("newSESV2ContactListResource() ok = true for topic without topic name, want false")
	}

	if _, ok := newSESV2ContactListResource(&sesv2.GetContactListOutput{}); ok {
		t.Fatal("newSESV2ContactListResource() ok = true for empty contact list name, want false")
	}
	if _, ok := newSESV2ContactListResource(nil); ok {
		t.Fatal("newSESV2ContactListResource() ok = true for nil contact list output, want false")
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
	resource, ok := newSESV2EmailIdentityResource("sender@example.com", &sesv2.GetEmailIdentityOutput{})
	if !ok {
		t.Fatal("newSESV2EmailIdentityResource() ok = false, want true")
	}
	assertSESV2Resource(t, resource, sesv2EmailIdentityResourceType, "sender@example.com", "email_identity", "sender@example.com")

	if _, ok := newSESV2EmailIdentityResource("", &sesv2.GetEmailIdentityOutput{}); ok {
		t.Fatal("newSESV2EmailIdentityResource() ok = true for empty identity name, want false")
	}

	if _, ok := newSESV2EmailIdentityResource("example.com", &sesv2.GetEmailIdentityOutput{
		DkimAttributes: &sesv2types.DkimAttributes{
			SigningAttributesOrigin: sesv2types.DkimSigningAttributesOriginExternal,
		},
	}); ok {
		t.Fatal("newSESV2EmailIdentityResource() ok = true for BYODKIM identity, want false")
	}
}

func TestNewSESV2EmailIdentityFeedbackAttributesResource(t *testing.T) {
	resource, ok := newSESV2EmailIdentityFeedbackAttributesResource("sender@example.com", &sesv2.GetEmailIdentityOutput{
		FeedbackForwardingStatus: true,
	})
	if !ok {
		t.Fatal("newSESV2EmailIdentityFeedbackAttributesResource() ok = false, want true")
	}
	assertSESV2ResourceAttributes(t, resource, sesv2EmailIdentityFeedbackAttributesResourceType, "sender@example.com",
		[]string{"email_identity_feedback_attributes", "sender@example.com"},
		map[string]string{
			"email_identity":           "sender@example.com",
			"email_forwarding_enabled": "true",
		})

	if _, ok := newSESV2EmailIdentityFeedbackAttributesResource("", &sesv2.GetEmailIdentityOutput{}); ok {
		t.Fatal("newSESV2EmailIdentityFeedbackAttributesResource() ok = true for empty identity name, want false")
	}
	if _, ok := newSESV2EmailIdentityFeedbackAttributesResource("sender@example.com", nil); ok {
		t.Fatal("newSESV2EmailIdentityFeedbackAttributesResource() ok = true for nil identity output, want false")
	}
}

func TestNewSESV2EmailIdentityMailFromAttributesResource(t *testing.T) {
	resource, ok := newSESV2EmailIdentityMailFromAttributesResource("example.com", &sesv2.GetEmailIdentityOutput{
		MailFromAttributes: &sesv2types.MailFromAttributes{
			BehaviorOnMxFailure: sesv2types.BehaviorOnMxFailureRejectMessage,
			MailFromDomain:      aws.String("bounce.example.com"),
		},
	})
	if !ok {
		t.Fatal("newSESV2EmailIdentityMailFromAttributesResource() ok = false, want true")
	}
	assertSESV2ResourceAttributes(t, resource, sesv2EmailIdentityMailFromAttributesResourceType, "example.com",
		[]string{"email_identity_mail_from_attributes", "example.com"},
		map[string]string{
			"behavior_on_mx_failure": "REJECT_MESSAGE",
			"email_identity":         "example.com",
			"mail_from_domain":       "bounce.example.com",
		})

	if _, ok := newSESV2EmailIdentityMailFromAttributesResource("", &sesv2.GetEmailIdentityOutput{}); ok {
		t.Fatal("newSESV2EmailIdentityMailFromAttributesResource() ok = true for empty identity name, want false")
	}
	if _, ok := newSESV2EmailIdentityMailFromAttributesResource("example.com", &sesv2.GetEmailIdentityOutput{}); ok {
		t.Fatal("newSESV2EmailIdentityMailFromAttributesResource() ok = true without mail-from attributes, want false")
	}
	if _, ok := newSESV2EmailIdentityMailFromAttributesResource("example.com", &sesv2.GetEmailIdentityOutput{
		MailFromAttributes: &sesv2types.MailFromAttributes{
			BehaviorOnMxFailure: sesv2types.BehaviorOnMxFailureRejectMessage,
		},
	}); ok {
		t.Fatal("newSESV2EmailIdentityMailFromAttributesResource() ok = true without mail-from domain, want false")
	}
}

func TestNewSESV2EmailIdentityPolicyResource(t *testing.T) {
	policy := `{"Version":"2012-10-17"}`
	resource, ok := newSESV2EmailIdentityPolicyResource("sender@example.com", "policy-a", policy)
	if !ok {
		t.Fatal("newSESV2EmailIdentityPolicyResource() ok = false, want true")
	}
	assertSESV2ResourceAttributes(t, resource, sesv2EmailIdentityPolicyResourceType, "sender@example.com|policy-a",
		[]string{"email_identity_policy", "sender@example.com", "policy-a"},
		map[string]string{
			"email_identity": "sender@example.com",
			"policy":         policy,
			"policy_name":    "policy-a",
		})

	if _, ok := newSESV2EmailIdentityPolicyResource("", "policy-a", policy); ok {
		t.Fatal("newSESV2EmailIdentityPolicyResource() ok = true for empty identity name, want false")
	}
	if _, ok := newSESV2EmailIdentityPolicyResource("sender@example.com", "", policy); ok {
		t.Fatal("newSESV2EmailIdentityPolicyResource() ok = true for empty policy name, want false")
	}
	if _, ok := newSESV2EmailIdentityPolicyResource("sender@example.com", "policy-a", ""); ok {
		t.Fatal("newSESV2EmailIdentityPolicyResource() ok = true for empty policy, want false")
	}
}

func TestSESV2EmailIdentityImportable(t *testing.T) {
	tests := []struct {
		name   string
		output *sesv2.GetEmailIdentityOutput
		want   bool
	}{
		{name: "nil output", output: nil, want: true},
		{name: "no DKIM attributes", output: &sesv2.GetEmailIdentityOutput{}, want: true},
		{name: "easy DKIM", output: &sesv2.GetEmailIdentityOutput{
			DkimAttributes: &sesv2types.DkimAttributes{
				SigningAttributesOrigin: sesv2types.DkimSigningAttributesOriginAwsSes,
			},
		}, want: true},
		{name: "BYODKIM", output: &sesv2.GetEmailIdentityOutput{
			DkimAttributes: &sesv2types.DkimAttributes{
				SigningAttributesOrigin: sesv2types.DkimSigningAttributesOriginExternal,
			},
		}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sesv2EmailIdentityImportable(tt.output); got != tt.want {
				t.Fatalf("sesv2EmailIdentityImportable() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestSESV2NotFound(t *testing.T) {
	if !sesv2NotFound(&sesv2types.NotFoundException{}) {
		t.Fatal("sesv2NotFound() = false for NotFoundException, want true")
	}
	if sesv2NotFound(errors.New("boom")) {
		t.Fatal("sesv2NotFound() = true for generic error, want false")
	}
	if sesv2NotFound(nil) {
		t.Fatal("sesv2NotFound() = true for nil error, want false")
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
		{name: "contact list separator", first: []string{"contact_list", "a_b", "c"}, second: []string{"contact_list", "a", "b_c"}},
		{name: "event destination composite", first: []string{"configuration_set_event_destination", "a_b", "c"}, second: []string{"configuration_set_event_destination", "a", "b_c"}},
		{name: "policy composite", first: []string{"email_identity_policy", "a|b", "c"}, second: []string{"email_identity_policy", "a", "b|c"}},
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

func TestSESV2ContactListResourceHCLIncludesTopics(t *testing.T) {
	resource, ok := newSESV2ContactListResource(&sesv2.GetContactListOutput{
		ContactListName: aws.String("contacts"),
		Topics: []sesv2types.Topic{
			{
				DefaultSubscriptionStatus: sesv2types.SubscriptionStatusOptIn,
				DisplayName:               aws.String("News"),
				TopicName:                 aws.String("news"),
			},
		},
	})
	if !ok {
		t.Fatal("newSESV2ContactListResource() ok = false, want true")
	}
	resource.Item = map[string]interface{}{
		"contact_list_name": resource.InstanceState.Attributes["contact_list_name"],
		"topic":             resource.AdditionalFields["topic"],
	}

	data, err := terraformutils.HclPrintResource([]terraformutils.Resource{resource}, map[string]interface{}{}, "hcl", true)
	if err != nil {
		t.Fatalf("HclPrintResource() error = %v", err)
	}
	output := string(data)
	normalizedOutput := strings.Join(strings.Fields(output), " ")
	for _, want := range []string{
		"topic {",
		"default_subscription_status = \"OPT_IN\"",
		"display_name = \"News\"",
		"topic_name = \"news\"",
	} {
		outputToSearch := output
		if strings.Contains(want, " = ") {
			outputToSearch = normalizedOutput
		}
		if !strings.Contains(outputToSearch, want) {
			t.Fatalf("output does not contain %q:\n%s", want, output)
		}
	}
}

func TestSESV2PostConvertHookWrapsPolicy(t *testing.T) {
	resource, ok := newSESV2EmailIdentityPolicyResource("sender@example.com", "policy-a", `{"Resource":"${aws:username}"}`)
	if !ok {
		t.Fatal("newSESV2EmailIdentityPolicyResource() ok = false, want true")
	}
	resource.Item = map[string]interface{}{
		"policy": `{"Resource":"${aws:username}"}`,
	}
	generator := SesV2Generator{
		AWSService: AWSService{
			Service: terraformutils.Service{
				Resources: []terraformutils.Resource{resource},
			},
		},
	}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}

	want := "<<POLICY\n{\"Resource\":\"$${aws:username}\"}\nPOLICY"
	if got := generator.Resources[0].Item["policy"]; got != want {
		t.Fatalf("policy = %q, want %q", got, want)
	}
}

func assertSESV2Resource(t *testing.T, resource terraformutils.Resource, resourceType, resourceID, attributeName, attributeValue string) {
	t.Helper()

	assertSESV2ResourceAttributes(t, resource, resourceType, resourceID, stringsForSESV2ResourceName(resourceType, attributeValue), map[string]string{
		attributeName: attributeValue,
	})
}

func assertSESV2ResourceAttributes(t *testing.T, resource terraformutils.Resource, resourceType, resourceID string, nameParts []string, attributes map[string]string) {
	t.Helper()

	if resource.InstanceInfo.Type != resourceType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, resourceType)
	}
	if resource.InstanceState.ID != resourceID {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, resourceID)
	}
	for name, want := range attributes {
		if got := resource.InstanceState.Attributes[name]; got != want {
			t.Fatalf("attribute %q = %q, want %q", name, got, want)
		}
	}
	wantName := terraformutils.TfSanitize(sesv2ResourceName(nameParts...))
	if resource.ResourceName != wantName {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, wantName)
	}
}

func assertSESV2ContactListTopics(t *testing.T, resource terraformutils.Resource, wants []map[string]interface{}) {
	t.Helper()

	topics, ok := resource.AdditionalFields["topic"].([]interface{})
	if !ok {
		t.Fatalf("topic additional field type = %T, want []interface{}", resource.AdditionalFields["topic"])
	}
	if len(topics) != len(wants) {
		t.Fatalf("topic additional field length = %d, want %d", len(topics), len(wants))
	}
	for i, want := range wants {
		got, ok := topics[i].(map[string]interface{})
		if !ok {
			t.Fatalf("topic[%d] type = %T, want map[string]interface{}", i, topics[i])
		}
		if len(got) != len(want) {
			t.Fatalf("topic[%d] = %#v, want %#v", i, got, want)
		}
		for key, wantValue := range want {
			if got[key] != wantValue {
				t.Fatalf("topic[%d][%q] = %q, want %q", i, key, got[key], wantValue)
			}
		}
	}
}

func stringsForSESV2ResourceName(resourceType, name string) []string {
	switch resourceType {
	case sesv2ConfigurationSetResourceType:
		return []string{"configuration_set", name}
	case sesv2ContactListResourceType:
		return []string{"contact_list", name}
	case sesv2DedicatedIPPoolResourceType:
		return []string{"dedicated_ip_pool", name}
	case sesv2EmailIdentityResourceType:
		return []string{"email_identity", name}
	default:
		return []string{name}
	}
}
