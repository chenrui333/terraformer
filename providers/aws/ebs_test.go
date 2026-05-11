// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
)

func TestNewEBSSnapshotResource(t *testing.T) {
	resource, ok := newEBSSnapshotResource(types.Snapshot{
		SnapshotId: aws.String("snap-123"),
		State:      types.SnapshotStateCompleted,
		VolumeId:   aws.String("vol-123"),
	})
	if !ok {
		t.Fatal("newEBSSnapshotResource() ok = false, want true")
	}
	if got := resource.InstanceInfo.Type; got != ebsSnapshotResourceType {
		t.Fatalf("resource type = %q, want %q", got, ebsSnapshotResourceType)
	}
	if got := resource.InstanceState.ID; got != "snap-123" {
		t.Fatalf("resource ID = %q, want snap-123", got)
	}

	if _, ok := newEBSSnapshotResource(types.Snapshot{State: types.SnapshotStateCompleted}); ok {
		t.Fatal("snapshot with empty ID should be skipped")
	}
	if _, ok := newEBSSnapshotResource(types.Snapshot{
		SnapshotId: aws.String("snap-123"),
		State:      types.SnapshotStateCompleted,
	}); ok {
		t.Fatal("snapshot with empty source volume should be skipped")
	}
	if _, ok := newEBSSnapshotResource(types.Snapshot{
		SnapshotId: aws.String("snap-123"),
		State:      types.SnapshotStateError,
	}); ok {
		t.Fatal("error snapshot should be skipped")
	}
	if _, ok := newEBSSnapshotResource(types.Snapshot{
		SnapshotId: aws.String("snap-123"),
		State:      types.SnapshotStatePending,
	}); ok {
		t.Fatal("pending snapshot should be skipped")
	}
	if _, ok := newEBSSnapshotResource(types.Snapshot{
		SnapshotId:   aws.String("snap-123"),
		State:        types.SnapshotStateCompleted,
		TransferType: types.TransferTypeStandard,
		VolumeId:     aws.String("vol-copy-derived"),
	}); ok {
		t.Fatal("copy-derived snapshot should be skipped")
	}
}

func TestNewEBSFastSnapshotRestoreResource(t *testing.T) {
	resource, ok := newEBSFastSnapshotRestoreResource(types.DescribeFastSnapshotRestoreSuccessItem{
		AvailabilityZone: aws.String("us-east-1a"),
		SnapshotId:       aws.String("snap-123"),
		State:            types.FastSnapshotRestoreStateCodeEnabled,
	})
	if !ok {
		t.Fatal("newEBSFastSnapshotRestoreResource() ok = false, want true")
	}
	if got := resource.InstanceInfo.Type; got != ebsFastSnapshotRestoreResourceType {
		t.Fatalf("resource type = %q, want %q", got, ebsFastSnapshotRestoreResourceType)
	}
	if got, want := resource.InstanceState.ID, "us-east-1a,snap-123"; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}

	if _, ok := newEBSFastSnapshotRestoreResource(types.DescribeFastSnapshotRestoreSuccessItem{
		SnapshotId: aws.String("snap-123"),
		State:      types.FastSnapshotRestoreStateCodeEnabled,
	}); ok {
		t.Fatal("fast snapshot restore with empty AZ should be skipped")
	}
	if _, ok := newEBSFastSnapshotRestoreResource(types.DescribeFastSnapshotRestoreSuccessItem{
		AvailabilityZone: aws.String("us-east-1a"),
		SnapshotId:       aws.String("snap-123"),
		State:            types.FastSnapshotRestoreStateCodeDisabled,
	}); ok {
		t.Fatal("disabled fast snapshot restore should be skipped")
	}
	if _, ok := newEBSFastSnapshotRestoreResource(types.DescribeFastSnapshotRestoreSuccessItem{
		AvailabilityZone: aws.String("us-east-1a"),
		SnapshotId:       aws.String("snap-123"),
		State:            types.FastSnapshotRestoreStateCodeEnabling,
	}); ok {
		t.Fatal("enabling fast snapshot restore should be skipped")
	}
}

func TestEBSFastSnapshotRestoreImportID(t *testing.T) {
	if got, want := ebsFastSnapshotRestoreImportID("us-east-1a", "snap-123"), "us-east-1a,snap-123"; got != want {
		t.Fatalf("ebsFastSnapshotRestoreImportID() = %q, want %q", got, want)
	}
	if got := ebsFastSnapshotRestoreImportID("", "snap-123"); got != "" {
		t.Fatalf("ebsFastSnapshotRestoreImportID(empty AZ) = %q, want empty", got)
	}
}

func TestNewEBSEncryptionByDefaultResource(t *testing.T) {
	resource, ok := newEBSEncryptionByDefaultResource(true)
	if !ok {
		t.Fatal("newEBSEncryptionByDefaultResource() ok = false, want true")
	}
	if got := resource.InstanceInfo.Type; got != ebsEncryptionByDefaultResourceType {
		t.Fatalf("resource type = %q, want %q", got, ebsEncryptionByDefaultResourceType)
	}
	if got := resource.InstanceState.ID; got != ebsEncryptionByDefaultImportID {
		t.Fatalf("resource ID = %q, want %q", got, ebsEncryptionByDefaultImportID)
	}
	if got := resource.InstanceState.Attributes["enabled"]; got != "true" {
		t.Fatalf("enabled = %q, want true", got)
	}
	assertEBSPreservesID(t, resource)

	if _, ok := newEBSEncryptionByDefaultResource(false); ok {
		t.Fatal("disabled EBS encryption by default should be skipped")
	}
}

func TestNewEBSDefaultKMSKeyResource(t *testing.T) {
	keyARN := "arn:aws:kms:us-east-1:123456789012:key/key-123"
	resource, ok := newEBSDefaultKMSKeyResource(keyARN)
	if !ok {
		t.Fatal("newEBSDefaultKMSKeyResource() ok = false, want true")
	}
	if got := resource.InstanceInfo.Type; got != ebsDefaultKMSKeyResourceType {
		t.Fatalf("resource type = %q, want %q", got, ebsDefaultKMSKeyResourceType)
	}
	if got := resource.InstanceState.ID; got != keyARN {
		t.Fatalf("resource ID = %q, want %q", got, keyARN)
	}
	if got := resource.InstanceState.Attributes["key_arn"]; got != keyARN {
		t.Fatalf("key_arn = %q, want %q", got, keyARN)
	}
	assertEBSPreservesID(t, resource)

	if _, ok := newEBSDefaultKMSKeyResource(""); ok {
		t.Fatal("default KMS key with empty ARN should be skipped")
	}
	if _, ok := newEBSDefaultKMSKeyResource("arn:aws:kms:us-east-1:123456789012:alias/aws/ebs"); ok {
		t.Fatal("AWS-managed default EBS KMS key should be skipped")
	}
}

func TestEBSResourceNamesPreservePartBoundaries(t *testing.T) {
	left := terraformutils.TfSanitize(ebsResourceName("snapshot", "ab", "c"))
	right := terraformutils.TfSanitize(ebsResourceName("snapshot", "a", "bc"))
	if left == right {
		t.Fatalf("resource names collide: %q", left)
	}
}

func assertEBSPreservesID(t *testing.T, resource terraformutils.Resource) {
	t.Helper()

	preserveID, ok := resource.InstanceState.Meta[tfcompat.MetaKeyPreserveIDAfterRefresh].(bool)
	if !ok || !preserveID {
		t.Fatalf("preserve ID metadata = %#v, want true", resource.InstanceState.Meta[tfcompat.MetaKeyPreserveIDAfterRefresh])
	}
}
