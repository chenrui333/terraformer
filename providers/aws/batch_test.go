// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	batchtypes "github.com/aws/aws-sdk-go-v2/service/batch/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestBatchSchedulingPolicyImportID(t *testing.T) {
	arn := "arn:aws:batch:us-east-1:123456789012:scheduling-policy/core"
	if got := batchSchedulingPolicyImportID(arn); got != arn {
		t.Fatalf("batchSchedulingPolicyImportID() = %q, want %q", got, arn)
	}
}

func TestNewBatchSchedulingPolicyResource(t *testing.T) {
	arn := "arn:aws:batch:us-east-1:123456789012:scheduling-policy/core"
	resource, ok := newBatchSchedulingPolicyResource(batchtypes.SchedulingPolicyListingDetail{
		Arn: aws.String(arn),
	})
	if !ok {
		t.Fatal("scheduling policy should be importable")
	}
	if resource.InstanceState.ID != arn {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, arn)
	}
	if got, want := resource.ResourceName, terraformutils.TfSanitize(batchResourceName("scheduling_policy", "core")); got != want {
		t.Fatalf("resource name = %q, want %q", got, want)
	}
	if resource.InstanceInfo.Type != batchSchedulingPolicyResourceType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, batchSchedulingPolicyResourceType)
	}
	if got := resource.InstanceState.Attributes["name"]; got != "core" {
		t.Fatalf("name = %q, want core", got)
	}
	if got := resource.InstanceState.Attributes["arn"]; got != arn {
		t.Fatalf("arn = %q, want %q", got, arn)
	}

	if _, ok := newBatchSchedulingPolicyResource(batchtypes.SchedulingPolicyListingDetail{}); ok {
		t.Fatal("scheduling policy with empty ARN should be skipped")
	}
	if _, ok := newBatchSchedulingPolicyResource(batchtypes.SchedulingPolicyListingDetail{
		Arn: aws.String("arn:aws:batch:us-east-1:123456789012:scheduling-policy/"),
	}); ok {
		t.Fatal("scheduling policy with empty name should be skipped")
	}
}
