// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

const (
	ebsVolumeResourceType                   = "aws_ebs_volume"
	ebsVolumeAttachmentResourceType         = "aws_volume_attachment"
	ebsSnapshotResourceType                 = "aws_ebs_snapshot"
	ebsFastSnapshotRestoreResourceType      = "aws_ebs_fast_snapshot_restore"
	ebsEncryptionByDefaultResourceType      = "aws_ebs_encryption_by_default"
	ebsDefaultKMSKeyResourceType            = "aws_ebs_default_kms_key"
	ebsEncryptionByDefaultImportID          = "ebs-encryption-by-default"
	ebsFastSnapshotRestoreImportIDSeparator = ","
	ebsSelfOwnerID                          = "self"
)

var ebsAllowEmptyValues = []string{"tags."}

type EbsGenerator struct {
	AWSService
}

func (g *EbsGenerator) volumeAttachmentID(device, volumeID, instanceID string) string {
	return fmt.Sprintf("vai-%d", terraformutils.HashString(fmt.Sprintf("%s-%s-%s-", device, instanceID, volumeID)))
}

func (g *EbsGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := ec2.NewFromConfig(config)
	loadVolumes := g.shouldLoadEBSResource("ebs_volume")
	loadVolumeAttachments := g.shouldLoadEBSResource("volume_attachment")
	if loadVolumes || loadVolumeAttachments {
		var filters []types.Filter
		for _, filter := range g.Filter {
			if strings.HasPrefix(filter.FieldPath, "tags.") && filter.IsApplicable("ebs_volume") {
				filters = append(filters, types.Filter{
					Name:   aws.String("tag:" + strings.TrimPrefix(filter.FieldPath, "tags.")),
					Values: filter.AcceptableValues,
				})
			}
		}
		p := ec2.NewDescribeVolumesPaginator(svc, &ec2.DescribeVolumesInput{
			Filters: filters,
		})
		for p.HasMorePages() {
			page, e := p.NextPage(context.TODO())
			if e != nil {
				return e
			}
			for _, volume := range page.Volumes {
				isRootDevice := false // Let's leave root device configuration to be done in ec2_instance resources

				for _, attachment := range volume.Attachments {
					instances, err := svc.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{
						InstanceIds: []string{StringValue(attachment.InstanceId)},
					})
					if err != nil {
						return fmt.Errorf(
							"describe EC2 instance %s for EBS volume %s: %w",
							StringValue(attachment.InstanceId),
							StringValue(volume.VolumeId),
							err,
						)
					}
					for _, reservation := range instances.Reservations {
						for _, instance := range reservation.Instances {
							if StringValue(instance.RootDeviceName) == StringValue(attachment.Device) {
								isRootDevice = true
							}
						}
					}
				}

				if !isRootDevice {
					if loadVolumes {
						g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
							StringValue(volume.VolumeId),
							StringValue(volume.VolumeId),
							ebsVolumeResourceType,
							"aws",
							ebsAllowEmptyValues,
						))
					}

					if loadVolumeAttachments {
						for _, attachment := range volume.Attachments {
							if attachment.State == types.VolumeAttachmentStateAttached {
								attachmentID := g.volumeAttachmentID(
									StringValue(attachment.Device),
									StringValue(attachment.VolumeId),
									StringValue(attachment.InstanceId))
								g.Resources = append(g.Resources, terraformutils.NewResource(
									attachmentID,
									StringValue(attachment.InstanceId)+":"+StringValue(attachment.Device),
									ebsVolumeAttachmentResourceType,
									"aws",
									map[string]string{
										"device_name": StringValue(attachment.Device),
										"volume_id":   StringValue(attachment.VolumeId),
										"instance_id": StringValue(attachment.InstanceId),
									},
									[]string{},
									map[string]interface{}{},
								))
							}
						}
					}
				}
			}
		}
	}
	if g.shouldLoadEBSResource("ebs_snapshot") {
		if err := g.loadSnapshots(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadEBSResource("ebs_fast_snapshot_restore") {
		if err := g.loadFastSnapshotRestores(svc); err != nil {
			return err
		}
	}
	if err := g.loadAccountSettings(svc); err != nil {
		return err
	}
	return nil
}

func (g *EbsGenerator) loadSnapshots(svc *ec2.Client) error {
	var filters []types.Filter
	for _, filter := range g.Filter {
		if strings.HasPrefix(filter.FieldPath, "tags.") && filter.IsApplicable("ebs_snapshot") {
			filters = append(filters, types.Filter{
				Name:   aws.String("tag:" + strings.TrimPrefix(filter.FieldPath, "tags.")),
				Values: filter.AcceptableValues,
			})
		}
	}
	p := ec2.NewDescribeSnapshotsPaginator(svc, &ec2.DescribeSnapshotsInput{
		Filters:  filters,
		OwnerIds: []string{ebsSelfOwnerID},
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, snapshot := range page.Snapshots {
			if resource, ok := newEBSSnapshotResource(snapshot); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *EbsGenerator) loadFastSnapshotRestores(svc *ec2.Client) error {
	p := ec2.NewDescribeFastSnapshotRestoresPaginator(svc, &ec2.DescribeFastSnapshotRestoresInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, restore := range page.FastSnapshotRestores {
			if resource, ok := newEBSFastSnapshotRestoreResource(restore); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *EbsGenerator) loadAccountSettings(svc *ec2.Client) error {
	if g.shouldLoadEBSResource("ebs_encryption_by_default") {
		encryption, err := svc.GetEbsEncryptionByDefault(context.TODO(), &ec2.GetEbsEncryptionByDefaultInput{})
		if err != nil {
			return err
		}
		if resource, ok := newEBSEncryptionByDefaultResource(aws.ToBool(encryption.EbsEncryptionByDefault)); ok {
			g.Resources = append(g.Resources, resource)
		}
	}

	if g.shouldLoadEBSResource("ebs_default_kms_key") {
		defaultKey, err := svc.GetEbsDefaultKmsKeyId(context.TODO(), &ec2.GetEbsDefaultKmsKeyIdInput{})
		if err != nil {
			return err
		}
		if resource, ok := newEBSDefaultKMSKeyResource(StringValue(defaultKey.KmsKeyId)); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *EbsGenerator) shouldLoadEBSResource(serviceNames ...string) bool {
	return shouldLoadAWSResourceForTypedFilters(g.Filter, serviceNames...)
}

func newEBSSnapshotResource(snapshot types.Snapshot) (terraformutils.Resource, bool) {
	if !ebsSnapshotImportable(snapshot) {
		return terraformutils.Resource{}, false
	}
	id := StringValue(snapshot.SnapshotId)
	return terraformutils.NewSimpleResource(
		id,
		ebsResourceName("snapshot", StringValue(snapshot.VolumeId), id),
		ebsSnapshotResourceType,
		"aws",
		ebsAllowEmptyValues,
	), true
}

func newEBSFastSnapshotRestoreResource(restore types.DescribeFastSnapshotRestoreSuccessItem) (terraformutils.Resource, bool) {
	if !ebsFastSnapshotRestoreImportable(restore) {
		return terraformutils.Resource{}, false
	}
	availabilityZone := StringValue(restore.AvailabilityZone)
	snapshotID := StringValue(restore.SnapshotId)
	importID := ebsFastSnapshotRestoreImportID(availabilityZone, snapshotID)
	return terraformutils.NewSimpleResource(
		importID,
		ebsResourceName("fast_snapshot_restore", availabilityZone, snapshotID),
		ebsFastSnapshotRestoreResourceType,
		"aws",
		ebsAllowEmptyValues,
	), true
}

func newEBSEncryptionByDefaultResource(enabled bool) (terraformutils.Resource, bool) {
	if !enabled {
		return terraformutils.Resource{}, false
	}
	resource := terraformutils.NewResource(
		ebsEncryptionByDefaultImportID,
		ebsEncryptionByDefaultImportID,
		ebsEncryptionByDefaultResourceType,
		"aws",
		map[string]string{
			"enabled": strconv.FormatBool(enabled),
		},
		ebsAllowEmptyValues,
		map[string]interface{}{},
	)
	setEBSPreserveIDAfterRefresh(&resource)
	return resource, true
}

func newEBSDefaultKMSKeyResource(keyARN string) (terraformutils.Resource, bool) {
	if !ebsDefaultKMSKeyImportable(keyARN) {
		return terraformutils.Resource{}, false
	}
	resource := terraformutils.NewResource(
		keyARN,
		ebsResourceName("default_kms_key", keyARN),
		ebsDefaultKMSKeyResourceType,
		"aws",
		map[string]string{
			"key_arn": keyARN,
		},
		ebsAllowEmptyValues,
		map[string]interface{}{},
	)
	setEBSPreserveIDAfterRefresh(&resource)
	return resource, true
}

func ebsSnapshotImportable(snapshot types.Snapshot) bool {
	if StringValue(snapshot.SnapshotId) == "" {
		return false
	}
	if snapshot.State != types.SnapshotStateCompleted {
		return false
	}
	if StringValue(snapshot.VolumeId) == "" {
		return false
	}
	return snapshot.TransferType == ""
}

func ebsFastSnapshotRestoreImportable(restore types.DescribeFastSnapshotRestoreSuccessItem) bool {
	if StringValue(restore.AvailabilityZone) == "" || StringValue(restore.SnapshotId) == "" {
		return false
	}
	return restore.State == types.FastSnapshotRestoreStateCodeEnabled
}

func ebsDefaultKMSKeyImportable(keyARN string) bool {
	if keyARN == "" {
		return false
	}
	return keyARN != "alias/aws/ebs" && !strings.HasSuffix(keyARN, ":alias/aws/ebs")
}

func ebsFastSnapshotRestoreImportID(availabilityZone, snapshotID string) string {
	if availabilityZone == "" || snapshotID == "" {
		return ""
	}
	return availabilityZone + ebsFastSnapshotRestoreImportIDSeparator + snapshotID
}

func setEBSPreserveIDAfterRefresh(resource *terraformutils.Resource) {
	if resource == nil || resource.InstanceState == nil {
		return
	}
	if resource.InstanceState.Meta == nil {
		resource.InstanceState.Meta = map[string]interface{}{}
	}
	resource.InstanceState.Meta[tfcompat.MetaKeyPreserveIDAfterRefresh] = true
}

func ebsResourceName(parts ...string) string {
	nameParts := make([]string, 0, len(parts)*2)
	for _, part := range parts {
		if part == "" {
			continue
		}
		nameParts = append(nameParts, strconv.Itoa(len(part)), part)
	}
	return strings.Join(nameParts, "_")
}
