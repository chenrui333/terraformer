// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/devicefarm"
	devicefarmtypes "github.com/aws/aws-sdk-go-v2/service/devicefarm/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	deviceFarmProjectResourceType         = "aws_devicefarm_project"
	deviceFarmDevicePoolResourceType      = "aws_devicefarm_device_pool"
	deviceFarmNetworkProfileResourceType  = "aws_devicefarm_network_profile"
	deviceFarmTestGridProjectResourceType = "aws_devicefarm_test_grid_project"
	deviceFarmUploadResourceType          = "aws_devicefarm_upload"
	deviceFarmInstanceProfileResourceType = "aws_devicefarm_instance_profile"
)

var devicefarmAllowEmptyValues = []string{"tags."}

type DeviceFarmGenerator struct {
	AWSService
}

func (g *DeviceFarmGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := devicefarm.NewFromConfig(config)
	projectIDFilter := deviceFarmProjectIDFilter(g.Filter)
	explicitProjectIDFilter := awsTypedIDFilterValues(g.Filter, deviceFarmProjectResourceType)

	if err := g.loadProjects(svc, projectIDFilter); err != nil {
		return err
	}
	if len(explicitProjectIDFilter) == 0 || len(awsTypedIDFilterValues(g.Filter, deviceFarmTestGridProjectResourceType)) > 0 {
		if err := g.loadTestGridProjects(svc); err != nil {
			log.Printf("[WARN] Skipping Device Farm test grid projects: %v", err)
		}
	}
	if len(explicitProjectIDFilter) == 0 || len(awsTypedIDFilterValues(g.Filter, deviceFarmInstanceProfileResourceType)) > 0 {
		if err := g.loadInstanceProfiles(svc); err != nil {
			log.Printf("[WARN] Skipping Device Farm instance profiles: %v", err)
		}
	}

	return nil
}

func (g *DeviceFarmGenerator) loadProjects(svc *devicefarm.Client, projectIDFilter map[string]bool) error {
	p := devicefarm.NewListProjectsPaginator(svc, &devicefarm.ListProjectsInput{})
	for p.HasMorePages() {
		page, e := p.NextPage(context.TODO())
		if e != nil {
			return e
		}
		for _, project := range page.Projects {
			projectArn := StringValue(project.Arn)
			if !awsIDFilterAllows(projectIDFilter, projectArn) {
				continue
			}
			if resource, ok := newDeviceFarmProjectResource(project); ok {
				g.Resources = append(g.Resources, resource)
			}
			if projectArn == "" {
				continue
			}
			if err := g.loadDevicePools(svc, projectArn); err != nil {
				log.Printf("[WARN] Skipping Device Farm device pools for project %s: %v", projectArn, err)
			}
			if err := g.loadNetworkProfiles(svc, projectArn); err != nil {
				log.Printf("[WARN] Skipping Device Farm network profiles for project %s: %v", projectArn, err)
			}
			if err := g.loadUploads(svc, projectArn); err != nil {
				log.Printf("[WARN] Skipping Device Farm uploads for project %s: %v", projectArn, err)
			}
		}
	}

	return nil
}

func (g *DeviceFarmGenerator) loadDevicePools(svc *devicefarm.Client, projectArn string) error {
	p := devicefarm.NewListDevicePoolsPaginator(svc, &devicefarm.ListDevicePoolsInput{
		Arn:  aws.String(projectArn),
		Type: devicefarmtypes.DevicePoolTypePrivate,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, devicePool := range page.DevicePools {
			if resource, ok := newDeviceFarmDevicePoolResource(devicePool); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}

	return nil
}

func (g *DeviceFarmGenerator) loadNetworkProfiles(svc *devicefarm.Client, projectArn string) error {
	input := &devicefarm.ListNetworkProfilesInput{
		Arn:  aws.String(projectArn),
		Type: devicefarmtypes.NetworkProfileTypePrivate,
	}
	for {
		output, err := svc.ListNetworkProfiles(context.TODO(), input)
		if err != nil {
			return err
		}
		for _, networkProfile := range output.NetworkProfiles {
			if resource, ok := newDeviceFarmNetworkProfileResource(networkProfile); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
		if output.NextToken == nil {
			break
		}
		input.NextToken = output.NextToken
	}

	return nil
}

func (g *DeviceFarmGenerator) loadUploads(svc *devicefarm.Client, projectArn string) error {
	p := devicefarm.NewListUploadsPaginator(svc, &devicefarm.ListUploadsInput{
		Arn: aws.String(projectArn),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, upload := range page.Uploads {
			if resource, ok := newDeviceFarmUploadResource(upload); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}

	return nil
}

func (g *DeviceFarmGenerator) loadTestGridProjects(svc *devicefarm.Client) error {
	p := devicefarm.NewListTestGridProjectsPaginator(svc, &devicefarm.ListTestGridProjectsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, project := range page.TestGridProjects {
			if resource, ok := newDeviceFarmTestGridProjectResource(project); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}

	return nil
}

func (g *DeviceFarmGenerator) loadInstanceProfiles(svc *devicefarm.Client) error {
	input := &devicefarm.ListInstanceProfilesInput{MaxResults: aws.Int32(100)}
	for {
		output, err := svc.ListInstanceProfiles(context.TODO(), input)
		if err != nil {
			return err
		}
		for _, profile := range output.InstanceProfiles {
			if resource, ok := newDeviceFarmInstanceProfileResource(profile); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
		if output.NextToken == nil {
			break
		}
		input.NextToken = output.NextToken
	}

	return nil
}

func deviceFarmProjectIDFilter(filters []terraformutils.ResourceFilter) map[string]bool {
	projectARNs, ok := deviceFarmProjectARNsFromChildFilterValues(filters)
	if !ok {
		return nil
	}
	return awsMergeIDFilterValues(awsTypedIDFilterValues(filters, deviceFarmProjectResourceType), projectARNs)
}

func deviceFarmProjectARNsFromChildFilterValues(filters []terraformutils.ResourceFilter) (map[string]bool, bool) {
	projectARNs := map[string]bool{}
	for _, resourceType := range []string{
		deviceFarmDevicePoolResourceType,
		deviceFarmNetworkProfileResourceType,
		deviceFarmUploadResourceType,
	} {
		for childARN := range awsTypedIDFilterValues(filters, resourceType) {
			projectARN := deviceFarmProjectARNFromProjectScopedARN(childARN)
			if projectARN == "" {
				return nil, false
			}
			projectARNs[projectARN] = true
		}
	}
	if len(projectARNs) == 0 {
		return nil, true
	}
	return projectARNs, true
}

func deviceFarmProjectARNFromProjectScopedARN(childARN string) string {
	parts := strings.SplitN(childARN, ":", 6)
	if len(parts) != 6 {
		return ""
	}
	resourceType, resourcePath, ok := strings.Cut(parts[5], ":")
	if !ok {
		return ""
	}
	switch resourceType {
	case "devicepool", "networkprofile", "upload":
	default:
		return ""
	}
	projectID, _, ok := strings.Cut(resourcePath, "/")
	if !ok || projectID == "" {
		return ""
	}
	return strings.Join(parts[:5], ":") + ":project:" + projectID
}

func newDeviceFarmProjectResource(project devicefarmtypes.Project) (terraformutils.Resource, bool) {
	projectArn := StringValue(project.Arn)
	if projectArn == "" {
		return terraformutils.Resource{}, false
	}
	projectName := StringValue(project.Name)
	if projectName == "" {
		projectName = projectArn
	}
	return terraformutils.NewSimpleResource(
		deviceFarmARNImportID(projectArn),
		deviceFarmResourceName("project", projectName, projectArn),
		deviceFarmProjectResourceType,
		"aws",
		devicefarmAllowEmptyValues), true
}

func newDeviceFarmDevicePoolResource(devicePool devicefarmtypes.DevicePool) (terraformutils.Resource, bool) {
	if !deviceFarmCustomerResource(devicePool.Type) {
		return terraformutils.Resource{}, false
	}
	return newDeviceFarmARNResource(StringValue(devicePool.Arn), StringValue(devicePool.Name), "device-pool", deviceFarmDevicePoolResourceType)
}

func newDeviceFarmNetworkProfileResource(networkProfile devicefarmtypes.NetworkProfile) (terraformutils.Resource, bool) {
	if !deviceFarmCustomerResource(networkProfile.Type) {
		return terraformutils.Resource{}, false
	}
	return newDeviceFarmARNResource(StringValue(networkProfile.Arn), StringValue(networkProfile.Name), "network-profile", deviceFarmNetworkProfileResourceType)
}

func newDeviceFarmTestGridProjectResource(project devicefarmtypes.TestGridProject) (terraformutils.Resource, bool) {
	return newDeviceFarmARNResource(StringValue(project.Arn), StringValue(project.Name), "test-grid-project", deviceFarmTestGridProjectResourceType)
}

func newDeviceFarmUploadResource(upload devicefarmtypes.Upload) (terraformutils.Resource, bool) {
	if !deviceFarmCustomerResource(upload.Category) {
		return terraformutils.Resource{}, false
	}
	return newDeviceFarmARNResource(StringValue(upload.Arn), StringValue(upload.Name), "upload", deviceFarmUploadResourceType)
}

func newDeviceFarmInstanceProfileResource(profile devicefarmtypes.InstanceProfile) (terraformutils.Resource, bool) {
	return newDeviceFarmARNResource(StringValue(profile.Arn), StringValue(profile.Name), "instance-profile", deviceFarmInstanceProfileResourceType)
}

func newDeviceFarmARNResource(arn, name, family, resourceType string) (terraformutils.Resource, bool) {
	if arn == "" {
		return terraformutils.Resource{}, false
	}
	if name == "" {
		name = arn
	}
	return terraformutils.NewSimpleResource(
		deviceFarmARNImportID(arn),
		deviceFarmResourceName(family, name, arn),
		resourceType,
		"aws",
		devicefarmAllowEmptyValues), true
}

func deviceFarmARNImportID(arn string) string {
	return arn
}

func deviceFarmCustomerResource[T ~string](resourceType T) bool {
	return resourceType == "" || string(resourceType) == "PRIVATE"
}

func deviceFarmResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d/%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return "devicefarm-resource"
	}
	return strings.Join(cleanParts, "/")
}
