// SPDX-License-Identifier: Apache-2.0

package tencentcloud

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

type NatGatewayGenerator struct {
	TencentCloudService
}

func (g *NatGatewayGenerator) InitResources() error {
	args := g.GetArgs()
	region := args["region"].(string)
	credential := args["credential"].(common.Credential)
	profile := NewTencentCloudClientProfile()
	client, err := vpc.NewClient(&credential, region, profile)
	if err != nil {
		return err
	}

	request := vpc.NewDescribeNatGatewaysRequest()

	var offset uint64
	var pageSize uint64 = 50
	allInstances := make([]*vpc.NatGateway, 0)

	for {
		request.Offset = &offset
		request.Limit = &pageSize
		response, err := client.DescribeNatGateways(request)
		if err != nil {
			return err
		}

		allInstances = append(allInstances, response.Response.NatGatewaySet...)
		if len(response.Response.NatGatewaySet) < int(pageSize) {
			break
		}
		offset += pageSize
	}

	for _, instance := range allInstances {
		resource := terraformutils.NewResource(
			*instance.NatGatewayId,
			*instance.NatGatewayName+"_"+*instance.NatGatewayId,
			"tencentcloud_nat_gateway",
			"tencentcloud",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		)
		g.Resources = append(g.Resources, resource)
	}

	return nil
}
