// SPDX-License-Identifier: Apache-2.0

package tencentcloud

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

type CvmGenerator struct {
	TencentCloudService
}

func (g *CvmGenerator) InitResources() error {
	args := g.GetArgs()
	region := args["region"].(string)
	credential := args["credential"].(common.Credential)
	profile := NewTencentCloudClientProfile()
	client, err := cvm.NewClient(&credential, region, profile)
	if err != nil {
		return err
	}

	request := cvm.NewDescribeInstancesRequest()
	filters := make([]string, 0)
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("tencentcloud_instance") {
			filters = append(filters, filter.AcceptableValues...)
		}
	}
	for i := range filters {
		request.InstanceIds = append(request.InstanceIds, &filters[i])
	}

	var offset int64
	var pageSize int64 = 50
	allInstances := make([]*cvm.Instance, 0)

	for {
		request.Offset = &offset
		request.Limit = &pageSize
		response, err := client.DescribeInstances(request)
		if err != nil {
			return err
		}

		allInstances = append(allInstances, response.Response.InstanceSet...)
		if len(response.Response.InstanceSet) < int(pageSize) {
			break
		}
		offset += pageSize
	}

	for _, instance := range allInstances {
		resource := terraformutils.NewResource(
			*instance.InstanceId,
			*instance.InstanceName+"_"+*instance.InstanceId,
			"tencentcloud_instance",
			"tencentcloud",
			map[string]string{
				"disable_monitor_service":  "false",
				"disable_security_service": "false",
				"force_delete":             "false",
			},
			[]string{},
			map[string]interface{}{},
		)
		// Do not collect keys with CVM cause there can be keys not belong to any of them
		g.Resources = append(g.Resources, resource)
	}

	return nil
}

/*
func (g *CvmGenerator) PostConvertHook() error {
	for _, resource := range g.Resources {
		if resource.InstanceInfo.Type == "tencentcloud_instance" {
			resource.InstanceState.Attributes["disable_monitor_service"] = "false"
			resource.InstanceState.Attributes["disable_security_service"] = "false"
			resource.InstanceState.Attributes["force_delete"] = "false"
		}
	}
	return nil
}
*/
