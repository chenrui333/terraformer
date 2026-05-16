// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestKmsKeyPolicyResourceID(t *testing.T) {
	keyID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	resource := terraformutils.NewResource(
		keyID,
		keyID+"_policy",
		"aws_kms_key_policy",
		"aws",
		map[string]string{"key_id": keyID},
		kmsAllowEmptyValues,
		map[string]interface{}{},
	)

	if got := resource.InstanceState.ID; got != keyID {
		t.Fatalf("resource ID = %q, want %q", got, keyID)
	}
	if got := resource.ResourceName; got != "tfer--"+keyID+"_policy" {
		t.Fatalf("resource name = %q, want %q", got, "tfer--"+keyID+"_policy")
	}
	if got := resource.InstanceInfo.Type; got != "aws_kms_key_policy" {
		t.Fatalf("resource type = %q, want %q", got, "aws_kms_key_policy")
	}
	if got := resource.InstanceState.Attributes["key_id"]; got != keyID {
		t.Fatalf("key_id attribute = %q, want %q", got, keyID)
	}
}

func TestKmsKeyAndPolicyResourceNamesDoNotCollide(t *testing.T) {
	keyID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	keyResource := terraformutils.NewResource(
		keyID, keyID, "aws_kms_key", "aws",
		map[string]string{"key_id": keyID}, kmsAllowEmptyValues, map[string]interface{}{},
	)
	policyResource := terraformutils.NewResource(
		keyID, keyID+"_policy", "aws_kms_key_policy", "aws",
		map[string]string{"key_id": keyID}, kmsAllowEmptyValues, map[string]interface{}{},
	)

	if keyResource.ResourceName == policyResource.ResourceName {
		t.Fatalf("key and policy resource names collide: %q", keyResource.ResourceName)
	}
}
