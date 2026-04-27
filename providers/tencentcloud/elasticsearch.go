// SPDX-License-Identifier: Apache-2.0

package tencentcloud

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	es "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/es/v20180416"
)

type EsGenerator struct {
	TencentCloudService
}

func (g *EsGenerator) InitResources() error {
	args := g.GetArgs()
	region := args["region"].(string)
	credential := args["credential"].(common.Credential)
	profile := NewTencentCloudClientProfile()
	client, err := es.NewClient(&credential, region, profile)
	if err != nil {
		return err
	}

	request := es.NewDescribeInstancesRequest()

	var offset uint64
	var pageSize uint64 = 50
	allInstances := make([]*es.InstanceInfo, 0)

	for {
		request.Offset = &offset
		request.Limit = &pageSize
		response, err := client.DescribeInstances(request)
		if err != nil {
			return err
		}

		allInstances = append(allInstances, response.Response.InstanceList...)
		if len(response.Response.InstanceList) < int(pageSize) {
			break
		}
		offset += pageSize
	}

	for _, instance := range allInstances {
		resource := terraformutils.NewResource(
			*instance.InstanceId,
			*instance.InstanceName+"_"+*instance.InstanceId,
			"tencentcloud_elasticsearch_instance",
			"tencentcloud",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		)
		g.Resources = append(g.Resources, resource)
	}

	return nil
}

func (g *EsGenerator) PostConvertHook() error {
	for i := range g.Resources {
		g.Resources[i].Item["password"] = "test1234;"
	}
	return nil
}
