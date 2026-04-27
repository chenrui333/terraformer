// SPDX-License-Identifier: Apache-2.0

package alicloud

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/chenrui333/terraformer/terraformutils"
)

// VpcGenerator Struct for generating AliCloud Elastic Compute Service
type VpcGenerator struct {
	AliCloudService
}

func resourceFromVpcResponse(vpc vpc.Vpc) terraformutils.Resource {
	return terraformutils.NewResource(
		vpc.VpcId,                  // id
		vpc.VpcId+"__"+vpc.VpcName, // name
		"alicloud_vpc",
		"alicloud",
		map[string]string{},
		[]string{},
		map[string]interface{}{},
	)
}

// InitResources Gets the list of all vpc Vpc ids and generates resources
func (g *VpcGenerator) InitResources() error {
	client, err := g.LoadClientFromProfile()
	if err != nil {
		return err
	}
	remaining := 1
	pageNumber := 1
	pageSize := 10

	allVpcs := make([]vpc.Vpc, 0)

	for remaining > 0 {
		raw, err := client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
			request := vpc.CreateDescribeVpcsRequest()
			request.RegionId = client.RegionID
			request.PageSize = requests.NewInteger(pageSize)
			request.PageNumber = requests.NewInteger(pageNumber)
			return vpcClient.DescribeVpcs(request)
		})
		if err != nil {
			return err
		}

		response := raw.(*vpc.DescribeVpcsResponse)
		allVpcs = append(allVpcs, response.Vpcs.Vpc...)
		remaining = response.TotalCount - pageNumber*pageSize
		pageNumber++
	}

	for _, Vpc := range allVpcs {
		resource := resourceFromVpcResponse(Vpc)
		g.Resources = append(g.Resources, resource)
	}

	return nil
}
