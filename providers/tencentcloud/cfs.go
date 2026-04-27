// SPDX-License-Identifier: Apache-2.0

package tencentcloud

import (
	"github.com/chenrui333/terraformer/terraformutils"
	cfs "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cfs/v20190719"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

type CfsGenerator struct {
	TencentCloudService
}

func (g *CfsGenerator) InitResources() error {
	args := g.GetArgs()
	region := args["region"].(string)
	credential := args["credential"].(common.Credential)
	profile := NewTencentCloudClientProfile()
	client, err := cfs.NewClient(&credential, region, profile)
	if err != nil {
		return err
	}

	request := cfs.NewDescribeCfsFileSystemsRequest()
	response, err := client.DescribeCfsFileSystems(request)
	if err != nil {
		return err
	}

	for _, instance := range response.Response.FileSystems {
		resource := terraformutils.NewResource(
			*instance.FileSystemId,
			*instance.FsName+"_"+*instance.FileSystemId,
			"tencentcloud_cfs_file_system",
			"tencentcloud",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		)
		g.Resources = append(g.Resources, resource)
	}

	return nil
}
