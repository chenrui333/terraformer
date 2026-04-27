// SPDX-License-Identifier: Apache-2.0

package tencentcloud

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	redis "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/redis/v20180412"
)

type RedisGenerator struct {
	TencentCloudService
}

func (g *RedisGenerator) InitResources() error {
	args := g.GetArgs()
	region := args["region"].(string)
	credential := args["credential"].(common.Credential)
	profile := NewTencentCloudClientProfile()
	client, err := redis.NewClient(&credential, region, profile)
	if err != nil {
		return err
	}

	request := redis.NewDescribeInstancesRequest()

	var offset uint64
	var pageSize uint64 = 50
	allInstances := make([]*redis.InstanceSet, 0)

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
			"tencentcloud_redis_instance",
			"tencentcloud",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		)
		g.Resources = append(g.Resources, resource)
	}

	return nil
}
