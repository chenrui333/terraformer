// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
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
				instances, _ := svc.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{
					InstanceIds: []string{StringValue(attachment.InstanceId)},
				})
				for _, reservation := range instances.Reservations {
					for _, instance := range reservation.Instances {
						if StringValue(instance.RootDeviceName) == StringValue(attachment.Device) {
							isRootDevice = true
						}
					}
				}
			}

			if !isRootDevice {
				g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
					StringValue(volume.VolumeId),
					StringValue(volume.VolumeId),
					"aws_ebs_volume",
					"aws",
					ebsAllowEmptyValues,
				))

				for _, attachment := range volume.Attachments {
					if attachment.State == types.VolumeAttachmentStateAttached {
						attachmentID := g.volumeAttachmentID(
							StringValue(attachment.Device),
							StringValue(attachment.VolumeId),
							StringValue(attachment.InstanceId))
						g.Resources = append(g.Resources, terraformutils.NewResource(
							attachmentID,
							StringValue(attachment.InstanceId)+":"+StringValue(attachment.Device),
							"aws_volume_attachment",
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
	return nil
}
