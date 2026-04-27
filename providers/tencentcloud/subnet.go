// SPDX-License-Identifier: Apache-2.0

package tencentcloud

import (
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

type SubnetGenerator struct {
	TencentCloudService
}

func (g *SubnetGenerator) InitResources() error {
	args := g.GetArgs()
	region := args["region"].(string)
	credential := args["credential"].(common.Credential)
	profile := NewTencentCloudClientProfile()
	client, err := vpc.NewClient(&credential, region, profile)
	if err != nil {
		return err
	}

	request := vpc.NewDescribeSubnetsRequest()
	offset := 0
	pageSize := 50
	allSubnets := make([]*vpc.Subnet, 0)

	for {
		offsetString := strconv.Itoa(offset)
		limitString := strconv.Itoa(pageSize)
		request.Offset = &offsetString
		request.Limit = &limitString
		response, err := client.DescribeSubnets(request)
		if err != nil {
			return err
		}

		allSubnets = append(allSubnets, response.Response.SubnetSet...)
		if len(response.Response.SubnetSet) < pageSize {
			break
		}
		offset += pageSize
	}

	for _, subnet := range allSubnets {
		resource := terraformutils.NewResource(
			*subnet.SubnetId,
			*subnet.SubnetName+"_"+*subnet.SubnetId,
			"tencentcloud_subnet",
			"tencentcloud",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		)
		g.Resources = append(g.Resources, resource)
	}

	return nil
}
