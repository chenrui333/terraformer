// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	notificationscontactstypes "github.com/aws/aws-sdk-go-v2/service/notificationscontacts/types"
)

func TestNotificationsContactsImportIDs(t *testing.T) {
	arn := "arn:aws:notificationscontacts::123456789012:emailcontact/contact-123"
	if got := notificationsContactsEmailContactImportID(arn); got != arn {
		t.Fatalf("import ID = %q, want %q", got, arn)
	}
}

func TestNewNotificationsContactsEmailContactResource(t *testing.T) {
	arn := "arn:aws:notificationscontacts::123456789012:emailcontact/contact-123"
	resource, ok := newNotificationsContactsEmailContactResource(notificationscontactstypes.EmailContact{
		Address: aws.String("alerts@example.com"),
		Arn:     aws.String(arn),
		Name:    aws.String("alerts"),
	})
	assertMessagingResource(t, resource, ok, notificationsContactsEmailContactResourceType, arn, map[string]string{
		"arn":           arn,
		"email_address": "alerts@example.com",
		"name":          "alerts",
	})

	if _, ok := newNotificationsContactsEmailContactResource(notificationscontactstypes.EmailContact{Name: aws.String("alerts")}); ok {
		t.Fatal("email contact without ARN/address should be skipped")
	}
}
