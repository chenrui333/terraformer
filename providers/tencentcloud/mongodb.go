// SPDX-License-Identifier: Apache-2.0

package tencentcloud

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	mongodb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/mongodb/v20180408"
)

type MongodbGenerator struct {
	TencentCloudService
}

func (g *MongodbGenerator) InitResources() error {
	args := g.GetArgs()
	region := args["region"].(string)
	credential := args["credential"].(common.Credential)
	profile := NewTencentCloudClientProfile()
	client, err := mongodb.NewClient(&credential, region, profile)
	if err != nil {
		return err
	}

	request := mongodb.NewDescribeDBInstancesRequest()

	var offset uint64
	var pageSize uint64 = 50
	allInstances := make([]*mongodb.MongoDBInstanceDetail, 0)

	for {
		request.Offset = &offset
		request.Limit = &pageSize
		response, err := client.DescribeDBInstances(request)
		if err != nil {
			return err
		}

		allInstances = append(allInstances, response.Response.InstanceDetails...)
		if len(response.Response.InstanceDetails) < int(pageSize) {
			break
		}
		offset += pageSize
	}

	for _, instance := range allInstances {
		resource := terraformutils.NewResource(
			*instance.InstanceId,
			*instance.InstanceName+"_"+*instance.InstanceId,
			"tencentcloud_mongodb_instance",
			"tencentcloud",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		)
		g.Resources = append(g.Resources, resource)
	}

	return nil
}

func (g *MongodbGenerator) PostConvertHook() error {
	for i, resource := range g.Resources {
		if resource.InstanceInfo.Type == "tencentcloud_mongodb_instance" {
			g.Resources[i].Item["password"] = "test1234;"
		}
	}
	return nil
}
