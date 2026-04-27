// SPDX-License-Identifier: Apache-2.0

package tencentcloud

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

type EipGenerator struct {
	TencentCloudService
}

func (g *EipGenerator) InitResources() error {
	args := g.GetArgs()
	region := args["region"].(string)
	credential := args["credential"].(common.Credential)
	profile := NewTencentCloudClientProfile()
	client, err := vpc.NewClient(&credential, region, profile)
	if err != nil {
		return err
	}

	var offset int64
	var pageSize int64 = 50
	allInstances := make([]*vpc.Address, 0)
	request := vpc.NewDescribeAddressesRequest()
	filters := make([]string, 0)
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("tencentcloud_eip") {
			filters = append(filters, filter.AcceptableValues...)
		}
	}
	for i := range filters {
		request.AddressIds = append(request.AddressIds, &filters[i])
	}

	for {
		request.Offset = &offset
		request.Limit = &pageSize
		response, err := client.DescribeAddresses(request)
		if err != nil {
			return err
		}

		allInstances = append(allInstances, response.Response.AddressSet...)
		if len(response.Response.AddressSet) < int(pageSize) {
			break
		}
		offset += pageSize
	}

	for _, instance := range allInstances {
		resource := terraformutils.NewResource(
			*instance.AddressId,
			*instance.AddressId,
			"tencentcloud_eip",
			"tencentcloud",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		)
		g.Resources = append(g.Resources, resource)

		if instance.InstanceId != nil && *instance.InstanceId != "" {
			association := terraformutils.NewResource(
				*instance.AddressId+"::"+*instance.InstanceId,
				*instance.AddressId,
				"tencentcloud_eip_association",
				"tencentcloud",
				map[string]string{},
				[]string{},
				map[string]interface{}{},
			)
			association.AdditionalFields["eip_id"] = "${tencentcloud_eip." + resource.ResourceName + ".id}"
			g.Resources = append(g.Resources, association)
		}
	}

	return nil
}
