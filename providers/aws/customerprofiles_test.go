// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/customerprofiles"
)

func TestCustomerProfilesImportIDs(t *testing.T) {
	if got := customerProfilesDomainImportID("customers"); got != "customers" {
		t.Fatalf("import ID = %q, want customers", got)
	}
}

func TestNewCustomerProfilesDomainResource(t *testing.T) {
	resource, ok := newCustomerProfilesDomainResource(&customerprofiles.GetDomainOutput{
		DeadLetterQueueUrl:    aws.String("https://sqs.us-east-1.amazonaws.com/123456789012/dlq"),
		DefaultEncryptionKey:  aws.String("arn:aws:kms:us-east-1:123456789012:key/key-123"),
		DefaultExpirationDays: aws.Int32(365),
		DomainName:            aws.String("customers"),
	})
	assertMessagingResource(t, resource, ok, customerProfilesDomainResourceType, "customers", map[string]string{
		"dead_letter_queue_url":   "https://sqs.us-east-1.amazonaws.com/123456789012/dlq",
		"default_encryption_key":  "arn:aws:kms:us-east-1:123456789012:key/key-123",
		"default_expiration_days": "365",
		"domain_name":             "customers",
	})

	if _, ok := newCustomerProfilesDomainResource(&customerprofiles.GetDomainOutput{DomainName: aws.String("customers")}); ok {
		t.Fatal("domain without default expiration should be skipped")
	}
	if _, ok := newCustomerProfilesDomainResource(nil); ok {
		t.Fatal("nil domain output should be skipped")
	}
}
