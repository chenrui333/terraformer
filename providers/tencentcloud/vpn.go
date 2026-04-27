// SPDX-License-Identifier: Apache-2.0

package tencentcloud

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

type VpnGenerator struct {
	TencentCloudService
}

func (g *VpnGenerator) InitResources() error {
	args := g.GetArgs()
	region := args["region"].(string)
	credential := args["credential"].(common.Credential)
	profile := NewTencentCloudClientProfile()
	client, err := vpc.NewClient(&credential, region, profile)
	if err != nil {
		return err
	}

	request := vpc.NewDescribeVpnGatewaysRequest()

	var offset uint64
	var pageSize uint64 = 50
	allInstances := make([]*vpc.VpnGateway, 0)

	for {
		request.Offset = &offset
		request.Limit = &pageSize
		response, err := client.DescribeVpnGateways(request)
		if err != nil {
			return err
		}

		allInstances = append(allInstances, response.Response.VpnGatewaySet...)
		if len(response.Response.VpnGatewaySet) < int(pageSize) {
			break
		}
		offset += pageSize
	}

	for _, instance := range allInstances {
		resource := terraformutils.NewResource(
			*instance.VpnGatewayId,
			*instance.VpnGatewayName+"_"+*instance.VpnGatewayId,
			"tencentcloud_vpn_gateway",
			"tencentcloud",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		)
		g.Resources = append(g.Resources, resource)
	}

	return nil
}
