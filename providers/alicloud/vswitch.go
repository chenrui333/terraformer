// SPDX-License-Identifier: Apache-2.0

package alicloud

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/chenrui333/terraformer/terraformutils"
)

// VSwitchGenerator Struct for generating AliCloud Elastic Compute Service
type VSwitchGenerator struct {
	AliCloudService
}

func resourceFromVSwitchResponse(vswitch vpc.VSwitch) terraformutils.Resource {
	return terraformutils.NewResource(
		vswitch.VSwitchId,
		vswitch.VSwitchId+"__"+vswitch.VSwitchName,
		"alicloud_vswitch",
		"alicloud",
		map[string]string{},
		[]string{},
		map[string]interface{}{},
	)
}

// InitResources Gets the list of all vpc VSwitch ids and generates resources
func (g *VSwitchGenerator) InitResources() error {
	client, err := g.LoadClientFromProfile()
	if err != nil {
		return err
	}
	remaining := 1
	pageNumber := 1
	pageSize := 10

	allVSwitchs := make([]vpc.VSwitch, 0)

	for remaining > 0 {
		raw, err := client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
			request := vpc.CreateDescribeVSwitchesRequest()
			request.RegionId = client.RegionID
			request.PageSize = requests.NewInteger(pageSize)
			request.PageNumber = requests.NewInteger(pageNumber)
			return vpcClient.DescribeVSwitches(request)
		})
		if err != nil {
			return err
		}

		response := raw.(*vpc.DescribeVSwitchesResponse)
		allVSwitchs = append(allVSwitchs, response.VSwitches.VSwitch...)
		remaining = response.TotalCount - pageNumber*pageSize
		pageNumber++
	}

	for _, VSwitch := range allVSwitchs {
		resource := resourceFromVSwitchResponse(VSwitch)
		g.Resources = append(g.Resources, resource)
	}

	return nil
}
