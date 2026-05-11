// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
)

func TestEBSShouldLoadResource(t *testing.T) {
	g := EbsGenerator{}
	if !g.shouldLoadEBSResource("ebs_snapshot") {
		t.Fatal("should load snapshots without typed filters")
	}

	g.Filter = []terraformutils.ResourceFilter{{
		ServiceName:      "ebs_volume",
		FieldPath:        "id",
		AcceptableValues: []string{"vol-123"},
	}}
	if !g.shouldLoadEBSResource("ebs_volume") {
		t.Fatal("should load typed EBS volume resource")
	}
	if !g.shouldLoadEBSResource("ebs_volume", "volume_attachment") {
		t.Fatal("should load shared volume loader for typed EBS volume resource")
	}
	if g.shouldLoadEBSResource("ebs_snapshot") {
		t.Fatal("should not load snapshots for typed EBS volume filter")
	}
	if g.shouldLoadEBSResource("ebs_default_kms_key") {
		t.Fatal("should not load default KMS key for typed EBS volume filter")
	}

	g.Filter = []terraformutils.ResourceFilter{{
		ServiceName:      "volume_attachment",
		FieldPath:        "id",
		AcceptableValues: []string{"i-123:/dev/sdf"},
	}}
	if !g.shouldLoadEBSResource("ebs_volume", "volume_attachment") {
		t.Fatal("should load shared volume loader for typed volume attachment resource")
	}
	if g.shouldLoadEBSResource("ebs_volume") {
		t.Fatal("should not append EBS volumes for typed volume attachment filter")
	}
}

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
	if got, want := resource.ResourceName, terraformutils.TfSanitize(ebsEncryptionByDefaultResourceName); got != want {
		t.Fatalf("resource name = %q, want %q", got, want)
	}
	if got := resource.InstanceState.Attributes["enabled"]; got != "true" {
		t.Fatalf("enabled = %q, want true", got)
	}
	assertEBSPreservesID(t, resource)

	resource, ok = newEBSEncryptionByDefaultResource(false)
	if !ok {
		t.Fatal("disabled EBS encryption by default should still be imported")
	}
	if got := resource.InstanceState.ID; got != ebsEncryptionByDefaultImportID {
		t.Fatalf("disabled resource ID = %q, want %q", got, ebsEncryptionByDefaultImportID)
	}
	if got := resource.InstanceState.Attributes["enabled"]; got != "false" {
		t.Fatalf("disabled enabled = %q, want false", got)
	}
}

func TestNewEBSDefaultKMSKeyResource(t *testing.T) {
	keyARN := "arn:aws:kms:us-east-1:123456789012:key/key-123"
	resource, ok := newEBSDefaultKMSKeyResource(keyARN, kmstypes.KeyManagerTypeCustomer)
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

	if _, ok := newEBSDefaultKMSKeyResource("", kmstypes.KeyManagerTypeCustomer); ok {
		t.Fatal("default KMS key with empty ARN should be skipped")
	}
	if _, ok := newEBSDefaultKMSKeyResource("arn:aws:kms:us-east-1:123456789012:alias/aws/ebs", kmstypes.KeyManagerTypeCustomer); ok {
		t.Fatal("AWS-managed default EBS KMS alias should be skipped")
	}
	if _, ok := newEBSDefaultKMSKeyResource("arn:aws:kms:us-east-1:123456789012:key/aws-managed-key", kmstypes.KeyManagerTypeAws); ok {
		t.Fatal("AWS-managed default EBS KMS key ARN should be skipped")
	}
}

func TestEBSDefaultKMSKeyDescribeSkippable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "typed not found", err: &kmstypes.NotFoundException{}, want: true},
		{name: "generic access denied", err: &smithy.GenericAPIError{Code: "AccessDeniedException"}, want: true},
		{name: "generic not found", err: &smithy.GenericAPIError{Code: "NotFoundException"}, want: true},
		{name: "generic internal", err: &smithy.GenericAPIError{Code: "InternalError"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ebsDefaultKMSKeyDescribeSkippable(tt.err); got != tt.want {
				t.Fatalf("ebsDefaultKMSKeyDescribeSkippable() = %t, want %t", got, tt.want)
			}
		})
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
