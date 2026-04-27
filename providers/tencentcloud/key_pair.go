// SPDX-License-Identifier: Apache-2.0

package tencentcloud

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

type KeyPairGenerator struct {
	TencentCloudService
}

func (g *KeyPairGenerator) InitResources() error {
	args := g.GetArgs()
	region := args["region"].(string)
	credential := args["credential"].(common.Credential)
	profile := NewTencentCloudClientProfile()
	client, err := cvm.NewClient(&credential, region, profile)
	if err != nil {
		return err
	}

	request := cvm.NewDescribeKeyPairsRequest()

	var offset int64
	var pageSize int64 = 50
	allInstances := make([]*cvm.KeyPair, 0)

	for {
		request.Offset = &offset
		request.Limit = &pageSize
		response, err := client.DescribeKeyPairs(request)
		if err != nil {
			return err
		}

		allInstances = append(allInstances, response.Response.KeyPairSet...)
		if len(response.Response.KeyPairSet) < int(pageSize) {
			break
		}
		offset += pageSize
	}

	for _, instance := range allInstances {
		resource := terraformutils.NewResource(
			*instance.KeyId,
			*instance.KeyName+"_"+*instance.KeyId,
			"tencentcloud_key_pair",
			"tencentcloud",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		)
		g.Resources = append(g.Resources, resource)
	}

	return nil
}
