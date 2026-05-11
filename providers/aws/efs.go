// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/efs"
	efstypes "github.com/aws/aws-sdk-go-v2/service/efs/types"
	"github.com/aws/smithy-go"
)

var efsAllowEmptyValues = []string{"tags."}

const (
	efsAccessPointResourceType              = "aws_efs_access_point"
	efsBackupPolicyResourceType             = "aws_efs_backup_policy"
	efsFileSystemResourceType               = "aws_efs_file_system"
	efsFileSystemPolicyResourceType         = "aws_efs_file_system_policy"
	efsMountTargetResourceType              = "aws_efs_mount_target"
	efsReplicationConfigurationResourceType = "aws_efs_replication_configuration"
)

type EfsGenerator struct {
	AWSService
}

func (g *EfsGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := efs.NewFromConfig(config)
	loadFileSystems := g.shouldLoadEFSResource(efsFileSystemResourceType)
	loadMountTargets := g.shouldLoadEFSResource(efsMountTargetResourceType)
	loadFileSystemPolicies := g.shouldLoadEFSResource(efsFileSystemPolicyResourceType)
	if loadFileSystems || loadMountTargets || loadFileSystemPolicies {
		if err := g.loadFileSystem(svc, loadFileSystems, loadMountTargets, loadFileSystemPolicies); err != nil {
			return err
		}
	}
	if g.shouldLoadEFSResource(efsBackupPolicyResourceType) {
		if err := g.loadBackupPolicies(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadEFSResource(efsReplicationConfigurationResourceType) {
		if err := g.loadReplicationConfigurations(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadEFSResource(efsAccessPointResourceType) {
		if err := g.loadAccessPoint(svc); err != nil {
			return err
		}
	}
	return nil
}

func (g *EfsGenerator) shouldLoadEFSResource(serviceNames ...string) bool {
	return shouldLoadAWSResourceForTypedFilters(g.Filter, serviceNames...)
}

func (g *EfsGenerator) loadFileSystem(svc *efs.Client, loadFileSystems, loadMountTargets, loadFileSystemPolicies bool) error {
	p := efs.NewDescribeFileSystemsPaginator(svc, &efs.DescribeFileSystemsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, fileSystem := range page.FileSystems {
			fileSystemID := StringValue(fileSystem.FileSystemId)
			if fileSystemID == "" {
				continue
			}
			if loadFileSystems {
				g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
					fileSystemID,
					fileSystemID,
					efsFileSystemResourceType,
					"aws",
					efsAllowEmptyValues))
			}

			if loadMountTargets {
				targetsResponse, err := svc.DescribeMountTargets(context.TODO(), &efs.DescribeMountTargetsInput{
					FileSystemId: fileSystem.FileSystemId,
				})
				if err != nil {
					return fmt.Errorf("describe efs mount targets for %s: %w", fileSystemID, err)
				}
				for _, mountTarget := range targetsResponse.MountTargets {
					if resource, ok := newEFSMountTargetResource(StringValue(mountTarget.MountTargetId)); ok {
						g.Resources = append(g.Resources, resource)
					}
				}
			}

			if loadFileSystemPolicies {
				policyResponse, err := svc.DescribeFileSystemPolicy(context.TODO(), &efs.DescribeFileSystemPolicyInput{
					FileSystemId: fileSystem.FileSystemId,
				})
				if efsFileSystemPolicyMissing(err) {
					continue
				}
				if err != nil {
					return fmt.Errorf("describe efs file system policy for %s: %w", fileSystemID, err)
				}
				if policyResponse == nil {
					continue
				}
				if resource, ok := newEFSFileSystemPolicyResource(fileSystemID, StringValue(policyResponse.Policy)); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func (g *EfsGenerator) loadBackupPolicies(svc *efs.Client) error {
	p := efs.NewDescribeFileSystemsPaginator(svc, &efs.DescribeFileSystemsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, fileSystem := range page.FileSystems {
			fileSystemID := StringValue(fileSystem.FileSystemId)
			if fileSystemID == "" {
				continue
			}
			backupResponse, err := svc.DescribeBackupPolicy(context.TODO(), &efs.DescribeBackupPolicyInput{
				FileSystemId: fileSystem.FileSystemId,
			})
			if efsFileSystemPolicyMissing(err) {
				continue
			}
			if err != nil {
				return fmt.Errorf("describe efs backup policy for %s: %w", fileSystemID, err)
			}
			if backupResponse == nil || backupResponse.BackupPolicy == nil {
				continue
			}
			if resource, ok := newEFSBackupPolicyResource(fileSystemID, backupResponse.BackupPolicy.Status); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *EfsGenerator) loadReplicationConfigurations(svc *efs.Client) error {
	p := efs.NewDescribeReplicationConfigurationsPaginator(svc, &efs.DescribeReplicationConfigurationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			if efsReplicationConfigurationMissing(err) {
				return nil
			}
			return err
		}
		for _, replication := range page.Replications {
			if resource, ok := newEFSReplicationConfigurationResource(replication); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func efsFileSystemPolicyMissing(err error) bool {
	var notFound *efstypes.PolicyNotFound
	if errors.As(err, &notFound) {
		return true
	}

	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && apiErr.ErrorCode() == "PolicyNotFound"
}

func efsReplicationConfigurationMissing(err error) bool {
	var notFound *efstypes.ReplicationNotFound
	if errors.As(err, &notFound) {
		return true
	}

	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && apiErr.ErrorCode() == "ReplicationNotFound"
}

func newEFSMountTargetResource(mountTargetID string) (terraformutils.Resource, bool) {
	if mountTargetID == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		mountTargetID,
		mountTargetID,
		efsMountTargetResourceType,
		"aws",
		efsAllowEmptyValues), true
}

func newEFSFileSystemPolicyResource(fileSystemID, policy string) (terraformutils.Resource, bool) {
	if fileSystemID == "" || policy == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		fileSystemID,
		fileSystemID,
		efsFileSystemPolicyResourceType,
		"aws",
		map[string]string{
			"file_system_id": fileSystemID,
			"policy":         policy,
		},
		efsAllowEmptyValues,
		map[string]interface{}{}), true
}

func newEFSBackupPolicyResource(fileSystemID string, status efstypes.Status) (terraformutils.Resource, bool) {
	if fileSystemID == "" || !efsBackupPolicyStatusImportable(status) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		fileSystemID,
		fileSystemID,
		efsBackupPolicyResourceType,
		"aws",
		map[string]string{
			"file_system_id":         fileSystemID,
			"backup_policy.#":        "1",
			"backup_policy.0.status": string(status),
		},
		efsAllowEmptyValues,
		map[string]interface{}{}), true
}

func efsBackupPolicyStatusImportable(status efstypes.Status) bool {
	return status == efstypes.StatusEnabled || status == efstypes.StatusDisabled
}

func newEFSReplicationConfigurationResource(replication efstypes.ReplicationConfigurationDescription) (terraformutils.Resource, bool) {
	sourceFileSystemID := StringValue(replication.SourceFileSystemId)
	if sourceFileSystemID == "" || len(replication.Destinations) == 0 {
		return terraformutils.Resource{}, false
	}
	destination := replication.Destinations[0]
	if !efsReplicationStatusImportable(destination.Status) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		sourceFileSystemID,
		sourceFileSystemID,
		efsReplicationConfigurationResourceType,
		"aws",
		map[string]string{
			"source_file_system_id": sourceFileSystemID,
		},
		efsAllowEmptyValues,
		map[string]interface{}{}), true
}

func efsReplicationStatusImportable(status efstypes.ReplicationStatus) bool {
	return status == efstypes.ReplicationStatusEnabled || status == efstypes.ReplicationStatusPaused
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
			if id == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				id,
				id,
				efsAccessPointResourceType,
				"aws",
				efsAllowEmptyValues))
		}
	}
	return nil
}

// PostConvertHook for add policy json as heredoc
func (g *EfsGenerator) PostConvertHook() error {
	for i, resource := range g.Resources {
		if resource.InstanceInfo.Type == efsFileSystemPolicyResourceType {
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
