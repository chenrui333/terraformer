// SPDX-License-Identifier: Apache-2.0

//nolint:revive // lint triage: legacy provider/API/security baseline is tracked in #175.
package tencentcloud

import (
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

type VpcGenerator struct {
	TencentCloudService
}

func (g *VpcGenerator) InitResources() error {
	args := g.GetArgs()
	region := args["region"].(string)
	credential := args["credential"].(common.Credential)
	profile := NewTencentCloudClientProfile()
	client, err := vpc.NewClient(&credential, region, profile)
	if err != nil {
		return err
	}

	request := vpc.NewDescribeVpcsRequest()
	request.Filters = make([]*vpc.Filter, 0)
	vpcIds := make([]string, 0)
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("tencentcloud_vpc") {
			vpcIds = append(vpcIds, filter.AcceptableValues...)
		}
	}
	if len(vpcIds) > 0 {
		request.VpcIds = make([]*string, 0, len(vpcIds))
		for i := range vpcIds {
			request.VpcIds = append(request.VpcIds, &vpcIds[i])
		}
	}

	offset := 0
	pageSize := 50
	allVpcs := make([]*vpc.Vpc, 0)

	for {
		offsetString := strconv.Itoa(offset)
		limitString := strconv.Itoa(pageSize)
		request.Offset = &offsetString
		request.Limit = &limitString
		response, err := client.DescribeVpcs(request)
		if err != nil {
			return err
		}

		allVpcs = append(allVpcs, response.Response.VpcSet...)
		if len(response.Response.VpcSet) < pageSize {
			break
		}
		offset += pageSize
	}

	for _, vpcInstance := range allVpcs {
		resource := terraformutils.NewResource(
			*vpcInstance.VpcId,
			*vpcInstance.VpcName+"_"+*vpcInstance.VpcId,
			"tencentcloud_vpc",
			"tencentcloud",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		)
		g.Resources = append(g.Resources, resource)
	}

	return nil
}
