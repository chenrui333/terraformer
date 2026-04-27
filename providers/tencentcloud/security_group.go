// SPDX-License-Identifier: Apache-2.0

package tencentcloud

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

type SecurityGroupGenerator struct {
	TencentCloudService
}

func (g *SecurityGroupGenerator) InitResources() error {
	args := g.GetArgs()
	region := args["region"].(string)
	credential := args["credential"].(common.Credential)
	profile := NewTencentCloudClientProfile()
	client, err := vpc.NewClient(&credential, region, profile)
	if err != nil {
		return err
	}

	request := vpc.NewDescribeSecurityGroupsRequest()
	filters := make([]string, 0)
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("tencentcloud_security_group") {
			filters = append(filters, filter.AcceptableValues...)
		}
	}
	for i := range filters {
		request.SecurityGroupIds = append(request.SecurityGroupIds, &filters[i])
	}

	var offset int64
	var pageSize int64 = 50
	allInstances := make([]*vpc.SecurityGroup, 0)

	for {
		offsetString := fmt.Sprintf("%d", offset)
		limitString := fmt.Sprintf("%d", pageSize)
		request.Offset = &offsetString
		request.Limit = &limitString
		response, err := client.DescribeSecurityGroups(request)
		if err != nil {
			return err
		}

		allInstances = append(allInstances, response.Response.SecurityGroupSet...)
		if len(response.Response.SecurityGroupSet) < int(pageSize) {
			break
		}
		offset += pageSize
	}

	for _, instance := range allInstances {
		resource := terraformutils.NewResource(
			*instance.SecurityGroupId,
			*instance.SecurityGroupName+"_"+*instance.SecurityGroupId,
			"tencentcloud_security_group",
			"tencentcloud",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		)
		g.Resources = append(g.Resources, resource)

		ruleResource := terraformutils.NewResource(
			*instance.SecurityGroupId,
			*instance.SecurityGroupName+"_"+*instance.SecurityGroupId,
			"tencentcloud_security_group_lite_rule",
			"tencentcloud",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		)
		ruleResource.AdditionalFields["security_group_id"] = "${tencentcloud_security_group." + resource.ResourceName + ".id}"
		g.Resources = append(g.Resources, ruleResource)
	}

	return nil
}
