// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	devicefarmtypes "github.com/aws/aws-sdk-go-v2/service/devicefarm/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewDeviceFarmProjectResource(t *testing.T) {
	arn := "arn:aws:devicefarm:us-west-2:123456789012:project:11111111-2222-3333-4444-555555555555"
	resource, ok := newDeviceFarmProjectResource(devicefarmtypes.Project{
		Arn:  aws.String(arn),
		Name: aws.String("core"),
	})
	assertDeviceFarmResource(t, resource, ok, arn, deviceFarmResourceName("project", "core", arn), deviceFarmProjectResourceType)

	if _, ok := newDeviceFarmProjectResource(devicefarmtypes.Project{}); ok {
		t.Fatal("project with empty ARN should be skipped")
	}
}

func TestNewDeviceFarmChildResources(t *testing.T) {
	tests := []struct {
		name      string
		resource  terraformutils.Resource
		ok        bool
		wantID    string
		wantType  string
		wantName  string
		skipEmpty bool
	}{
		{
			name: "device pool",
			resource: mustDeviceFarmResource(newDeviceFarmDevicePoolResource(devicefarmtypes.DevicePool{
				Arn:  aws.String("arn:aws:devicefarm:us-west-2:123456789012:devicepool:project-id/device-pool-id"),
				Name: aws.String("phones"),
				Type: devicefarmtypes.DevicePoolTypePrivate,
			})),
			ok:       true,
			wantID:   "arn:aws:devicefarm:us-west-2:123456789012:devicepool:project-id/device-pool-id",
			wantType: deviceFarmDevicePoolResourceType,
			wantName: deviceFarmResourceName("device-pool", "phones", "arn:aws:devicefarm:us-west-2:123456789012:devicepool:project-id/device-pool-id"),
		},
		{
			name: "network profile",
			resource: mustDeviceFarmResource(newDeviceFarmNetworkProfileResource(devicefarmtypes.NetworkProfile{
				Arn:  aws.String("arn:aws:devicefarm:us-west-2:123456789012:networkprofile:project-id/profile-id"),
				Name: aws.String("office-wifi"),
				Type: devicefarmtypes.NetworkProfileTypePrivate,
			})),
			ok:       true,
			wantID:   "arn:aws:devicefarm:us-west-2:123456789012:networkprofile:project-id/profile-id",
			wantType: deviceFarmNetworkProfileResourceType,
			wantName: deviceFarmResourceName("network-profile", "office-wifi", "arn:aws:devicefarm:us-west-2:123456789012:networkprofile:project-id/profile-id"),
		},
		{
			name: "test grid project",
			resource: mustDeviceFarmResource(newDeviceFarmTestGridProjectResource(devicefarmtypes.TestGridProject{
				Arn:  aws.String("arn:aws:devicefarm:us-west-2:123456789012:testgrid-project:grid-id"),
				Name: aws.String("browser-grid"),
			})),
			ok:       true,
			wantID:   "arn:aws:devicefarm:us-west-2:123456789012:testgrid-project:grid-id",
			wantType: deviceFarmTestGridProjectResourceType,
			wantName: deviceFarmResourceName("test-grid-project", "browser-grid", "arn:aws:devicefarm:us-west-2:123456789012:testgrid-project:grid-id"),
		},
		{
			name: "upload",
			resource: mustDeviceFarmResource(newDeviceFarmUploadResource(devicefarmtypes.Upload{
				Arn:      aws.String("arn:aws:devicefarm:us-west-2:123456789012:upload:project-id/upload-id"),
				Name:     aws.String("app.apk"),
				Category: devicefarmtypes.UploadCategoryPrivate,
			})),
			ok:       true,
			wantID:   "arn:aws:devicefarm:us-west-2:123456789012:upload:project-id/upload-id",
			wantType: deviceFarmUploadResourceType,
			wantName: deviceFarmResourceName("upload", "app.apk", "arn:aws:devicefarm:us-west-2:123456789012:upload:project-id/upload-id"),
		},
		{
			name: "instance profile",
			resource: mustDeviceFarmResource(newDeviceFarmInstanceProfileResource(devicefarmtypes.InstanceProfile{
				Arn:  aws.String("arn:aws:devicefarm:us-west-2:123456789012:instanceprofile:profile-id"),
				Name: aws.String("private-devices"),
			})),
			ok:       true,
			wantID:   "arn:aws:devicefarm:us-west-2:123456789012:instanceprofile:profile-id",
			wantType: deviceFarmInstanceProfileResourceType,
			wantName: deviceFarmResourceName("instance-profile", "private-devices", "arn:aws:devicefarm:us-west-2:123456789012:instanceprofile:profile-id"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertDeviceFarmResource(t, tt.resource, tt.ok, tt.wantID, tt.wantName, tt.wantType)
		})
	}
}

func TestDeviceFarmSkipsUnsafeOrEmptyIdentifiers(t *testing.T) {
	if _, ok := newDeviceFarmDevicePoolResource(devicefarmtypes.DevicePool{
		Arn:  aws.String("arn:aws:devicefarm:us-west-2:123456789012:devicepool:project-id/curated"),
		Name: aws.String("curated"),
		Type: devicefarmtypes.DevicePoolTypeCurated,
	}); ok {
		t.Fatal("curated device pool should be skipped")
	}

	if _, ok := newDeviceFarmNetworkProfileResource(devicefarmtypes.NetworkProfile{
		Arn:  aws.String("arn:aws:devicefarm:us-west-2:123456789012:networkprofile:project-id/curated"),
		Name: aws.String("curated"),
		Type: devicefarmtypes.NetworkProfileTypeCurated,
	}); ok {
		t.Fatal("curated network profile should be skipped")
	}

	if _, ok := newDeviceFarmUploadResource(devicefarmtypes.Upload{
		Arn:      aws.String("arn:aws:devicefarm:us-west-2:123456789012:upload:project-id/curated"),
		Name:     aws.String("curated"),
		Category: devicefarmtypes.UploadCategoryCurated,
	}); ok {
		t.Fatal("curated upload should be skipped")
	}

	if _, ok := newDeviceFarmInstanceProfileResource(devicefarmtypes.InstanceProfile{Name: aws.String("missing-arn")}); ok {
		t.Fatal("instance profile with empty ARN should be skipped")
	}
}

func TestDeviceFarmARNImportID(t *testing.T) {
	arn := "arn:aws:devicefarm:us-west-2:123456789012:project:project-id"
	if got := deviceFarmARNImportID(arn); got != arn {
		t.Fatalf("Device Farm import ID = %q, want %q", got, arn)
	}
}

func TestDeviceFarmProjectIDFilterIncludesProjectScopedChildParents(t *testing.T) {
	service := terraformutils.Service{}
	service.ParseFilters([]string{
		"devicefarm_project='arn:aws:devicefarm:us-west-2:123456789012:project:project-parent'",
		"devicefarm_device_pool='arn:aws:devicefarm:us-west-2:123456789012:devicepool:project-child/device-pool-id'",
		"devicefarm_network_profile='arn:aws:devicefarm:us-west-2:123456789012:networkprofile:project-network/profile-id'",
		"devicefarm_upload='arn:aws:devicefarm:us-west-2:123456789012:upload:project-upload/upload-id'",
	})

	filter := deviceFarmProjectIDFilter(service.Filter)
	for _, projectARN := range []string{
		"arn:aws:devicefarm:us-west-2:123456789012:project:project-parent",
		"arn:aws:devicefarm:us-west-2:123456789012:project:project-child",
		"arn:aws:devicefarm:us-west-2:123456789012:project:project-network",
		"arn:aws:devicefarm:us-west-2:123456789012:project:project-upload",
	} {
		if !awsIDFilterAllows(filter, projectARN) {
			t.Fatalf("Device Farm project filter should allow %q: %#v", projectARN, filter)
		}
	}
	if awsIDFilterAllows(filter, "arn:aws:devicefarm:us-west-2:123456789012:project:project-other") {
		t.Fatalf("Device Farm project filter allowed unrelated project: %#v", filter)
	}
}

func TestDeviceFarmProjectIDFilterAllowsAllForUnparseableChildID(t *testing.T) {
	service := terraformutils.Service{}
	service.ParseFilters([]string{
		"devicefarm_project='arn:aws:devicefarm:us-west-2:123456789012:project:project-parent'",
		"devicefarm_device_pool=malformed",
	})

	filter := deviceFarmProjectIDFilter(service.Filter)
	if !awsIDFilterAllows(filter, "arn:aws:devicefarm:us-west-2:123456789012:project:project-other") {
		t.Fatalf("unparseable Device Farm child ID should disable project prefilter: %#v", filter)
	}
}

func TestDeviceFarmResourceNamesPreserveSegmentBoundaries(t *testing.T) {
	left := terraformutils.TfSanitize(deviceFarmResourceName("device-pool", "a/b_c"))
	right := terraformutils.TfSanitize(deviceFarmResourceName("device", "pool/a_b_c"))
	if left == right {
		t.Fatalf("Device Farm resource names collide: %q", left)
	}
}

func mustDeviceFarmResource(resource terraformutils.Resource, ok bool) terraformutils.Resource {
	if !ok {
		panic("expected Device Farm resource")
	}
	return resource
}

func assertDeviceFarmResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantName, wantType string) {
	t.Helper()
	if !ok {
		t.Fatal("resource was skipped")
	}
	if got := resource.InstanceState.ID; got != wantID {
		t.Fatalf("resource ID = %q, want %q", got, wantID)
	}
	if got := resource.ResourceName; got != terraformutils.TfSanitize(wantName) {
		t.Fatalf("resource name = %q, want %q", got, terraformutils.TfSanitize(wantName))
	}
	if got := resource.InstanceInfo.Type; got != wantType {
		t.Fatalf("resource type = %q, want %q", got, wantType)
	}
}
