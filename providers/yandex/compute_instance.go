// SPDX-License-Identifier: Apache-2.0

package yandex

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	ycsdk "github.com/yandex-cloud/go-sdk/v2"
)

type InstanceGenerator struct {
	YandexService
}

func (g *InstanceGenerator) loadInstances(sdk *ycsdk.SDK, folderID string) ([]*compute.Instance, error) {
	instances := []*compute.Instance{}
	pageToken := ""
	for {
		resp, err := compute.NewInstanceServiceClient(yandexGRPCClient(sdk)).List(context.Background(), &compute.ListInstancesRequest{
			FolderId:  folderID,
			PageSize:  defaultPageSize,
			PageToken: pageToken,
		})

		if err != nil {
			return nil, err
		}

		instances = append(instances, resp.GetInstances()...)

		if resp.GetNextPageToken() == "" {
			break
		}
	}
	return instances, nil
}

func (g *InstanceGenerator) InitResources() error {
	sdk, err := g.InitSDK()
	if err != nil {
		return err
	}

	result, err := g.loadInstances(sdk, g.Args["folder_id"].(string))
	if err != nil {
		return err
	}

	g.Resources = g.createResources(result)

	return nil
}

func (g *InstanceGenerator) createResources(instances []*compute.Instance) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, instance := range instances {
		resources = append(resources, terraformutils.NewSimpleResource(
			instance.GetId(),
			instance.GetId(),
			"yandex_compute_instance",
			"yandex",
			[]string{}))
	}
	return resources
}
