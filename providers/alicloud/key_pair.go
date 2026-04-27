// SPDX-License-Identifier: Apache-2.0

package alicloud

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/chenrui333/terraformer/terraformutils"
)

// KeyPairGenerator Struct for generating AliCloud Key pair
type KeyPairGenerator struct {
	AliCloudService
}

func resourceFromKeyPair(keyPair ecs.KeyPair) terraformutils.Resource {
	return terraformutils.NewResource(
		keyPair.KeyPairName,
		keyPair.KeyPairName+"__"+keyPair.KeyPairName,
		"alicloud_key_pair",
		"alicloud",
		map[string]string{},
		[]string{},
		map[string]interface{}{},
	)
}

// InitResources Gets the list of all key pair ids and generates resources
func (g *KeyPairGenerator) InitResources() error {
	client, err := g.LoadClientFromProfile()
	if err != nil {
		return err
	}
	remaining := 1
	pageNumber := 1
	pageSize := 10

	allKeyPairs := make([]ecs.KeyPair, 0)

	for remaining > 0 {
		raw, err := client.WithEcsClient(func(ecsClient *ecs.Client) (interface{}, error) {
			request := ecs.CreateDescribeKeyPairsRequest()
			request.RegionId = client.RegionID
			request.PageSize = requests.NewInteger(pageSize)
			request.PageNumber = requests.NewInteger(pageNumber)
			return ecsClient.DescribeKeyPairs(request)
		})
		if err != nil {
			return err
		}

		response := raw.(*ecs.DescribeKeyPairsResponse)
		allKeyPairs = append(allKeyPairs, response.KeyPairs.KeyPair...)
		remaining = response.TotalCount - pageNumber*pageSize
		pageNumber++
	}

	for _, keypair := range allKeyPairs {
		resource := resourceFromKeyPair(keypair)
		g.Resources = append(g.Resources, resource)
	}

	return nil
}
