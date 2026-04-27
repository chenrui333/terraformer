// SPDX-License-Identifier: Apache-2.0

package tencentcloud

import (
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

type DnspodGenerator struct {
	TencentCloudService
}

func (g *DnspodGenerator) InitResources() error {
	args := g.GetArgs()
	region := args["region"].(string)
	credential := args["credential"].(common.Credential)
	profile := NewTencentCloudClientProfile()
	client, err := dnspod.NewClient(&credential, region, profile)
	if err != nil {
		return err
	}

	return g.DescribeDomainList(client)
}
func (g *DnspodGenerator) DescribeDomainList(client *dnspod.Client) error {
	request := dnspod.NewDescribeDomainListRequest()

	var offset int64
	var limit int64 = 50
	allInstances := make([]*dnspod.DomainListItem, 0)
	for {
		request.Offset = &offset
		request.Limit = &limit
		response, err := client.DescribeDomainList(request)
		if err != nil {
			return err
		}
		allInstances = append(allInstances, response.Response.DomainList...)
		if len(response.Response.DomainList) < int(limit) {
			break
		}

		offset += limit
	}

	for _, instance := range allInstances {
		resource := terraformutils.NewResource(
			*instance.Name,
			*instance.Name,
			"tencentcloud_dnspod_domain_instance",
			"tencentcloud",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		)
		g.Resources = append(g.Resources, resource)
		if err := g.DescribeRecordList(client, *instance.Name, resource.ResourceName); err != nil {
			return err
		}
	}

	return nil
}
func (g *DnspodGenerator) DescribeRecordList(client *dnspod.Client, name, resourceName string) error {
	request := dnspod.NewDescribeRecordListRequest()

	request.Domain = &name
	var offset uint64
	var limit uint64 = 50
	allInstances := make([]*dnspod.RecordListItem, 0)
	for {
		request.Offset = &offset
		request.Limit = &limit
		response, err := client.DescribeRecordList(request)
		if err != nil {
			return err
		}
		allInstances = append(allInstances, response.Response.RecordList...)
		if len(response.Response.RecordList) < int(limit) {
			break
		}

		offset += limit
	}

	for _, instance := range allInstances {
		resource := terraformutils.NewResource(
			name+"#"+strconv.FormatUint(*instance.RecordId, 10),
			name+"_"+strconv.FormatUint(*instance.RecordId, 10),
			"tencentcloud_dnspod_record",
			"tencentcloud",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		)
		resource.AdditionalFields["domain"] = "${tencentcloud_dnspod_domain_instance." + resourceName + ".id}"
		g.Resources = append(g.Resources, resource)
	}

	return nil
}
