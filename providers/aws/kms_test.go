// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestKmsKeyResourceID(t *testing.T) {
	keyID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	resource := terraformutils.NewResource(
		keyID,
		keyID,
		"aws_kms_key",
		"aws",
		map[string]string{"key_id": keyID},
		kmsAllowEmptyValues,
		map[string]interface{}{},
	)

	if got := resource.InstanceState.ID; got != keyID {
		t.Fatalf("resource ID = %q, want %q", got, keyID)
	}
	if got := resource.InstanceInfo.Type; got != "aws_kms_key" {
		t.Fatalf("resource type = %q, want %q", got, "aws_kms_key")
	}
	if got := resource.InstanceState.Attributes["key_id"]; got != keyID {
		t.Fatalf("key_id attribute = %q, want %q", got, keyID)
	}
}

func TestKmsGrantResourceID(t *testing.T) {
	keyID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	grantID := "grant-123"
	compositeID := keyID + ":" + grantID
	resource := terraformutils.NewSimpleResource(
		compositeID,
		compositeID,
		"aws_kms_grant",
		"aws",
		kmsAllowEmptyValues,
	)

	if got := resource.InstanceState.ID; got != compositeID {
		t.Fatalf("resource ID = %q, want %q", got, compositeID)
	}
	if got := resource.InstanceInfo.Type; got != "aws_kms_grant" {
		t.Fatalf("resource type = %q, want %q", got, "aws_kms_grant")
	}
}
