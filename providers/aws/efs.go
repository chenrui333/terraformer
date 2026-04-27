// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/efs"
)

var efsAllowEmptyValues = []string{"tags."}

type EfsGenerator struct {
	AWSService
}

func (g *EfsGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := efs.NewFromConfig(config)
	if err := g.loadFileSystem(svc); err != nil {
		return err
	}
	if err := g.loadAccessPoint(svc); err != nil {
		return err
	}
	return nil
}

func (g *EfsGenerator) loadFileSystem(svc *efs.Client) error {
	p := efs.NewDescribeFileSystemsPaginator(svc, &efs.DescribeFileSystemsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, fileSystem := range page.FileSystems {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				StringValue(fileSystem.FileSystemId),
				StringValue(fileSystem.FileSystemId),
				"aws_efs_file_system",
				"aws",
				efsAllowEmptyValues))

			targetsResponse, err := svc.DescribeMountTargets(context.TODO(), &efs.DescribeMountTargetsInput{
				FileSystemId: fileSystem.FileSystemId,
			})
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			for _, mountTarget := range targetsResponse.MountTargets {
				g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
					StringValue(mountTarget.MountTargetId),
					StringValue(mountTarget.MountTargetId),
					"aws_efs_mount_target",
					"aws",
					efsAllowEmptyValues))
			}

			policyResponse, err := svc.DescribeFileSystemPolicy(context.TODO(), &efs.DescribeFileSystemPolicyInput{
				FileSystemId: fileSystem.FileSystemId,
			})
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				StringValue(fileSystem.FileSystemId),
				StringValue(fileSystem.FileSystemId),
				"aws_efs_file_system_policy",
				"aws",
				map[string]string{
					"file_system_id": StringValue(fileSystem.FileSystemId),
					"policy":         StringValue(policyResponse.Policy),
				},
				efsAllowEmptyValues,
				map[string]interface{}{}))
		}
	}
	return nil
}

func (g *EfsGenerator) loadAccessPoint(svc *efs.Client) error {
	p := efs.NewDescribeAccessPointsPaginator(svc, &efs.DescribeAccessPointsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, fileSystem := range page.AccessPoints {
			id := StringValue(fileSystem.AccessPointId)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				id,
				id,
				"aws_efs_access_point",
				"aws",
				efsAllowEmptyValues))
		}
	}
	return nil
}

// PostConvertHook for add policy json as heredoc
func (g *EfsGenerator) PostConvertHook() error {
	for i, resource := range g.Resources {
		if resource.InstanceInfo.Type == "aws_efs_file_system_policy" {
			if val, ok := g.Resources[i].Item["policy"]; ok {
				policy := g.escapeAwsInterpolation(val.(string))
				g.Resources[i].Item["policy"] = fmt.Sprintf(`<<POLICY
%s
POLICY`, policy)
			}
		}
	}
	return nil
}
