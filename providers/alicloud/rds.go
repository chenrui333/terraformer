// SPDX-License-Identifier: Apache-2.0

package alicloud

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
	"github.com/chenrui333/terraformer/terraformutils"
)

// RdsGenerator Struct for generating AliCloud Elastic Compute Service
type RdsGenerator struct {
	AliCloudService
}

func resourceFromrdsResponse(rds rds.DBInstance) terraformutils.Resource {
	return terraformutils.NewResource(
		rds.DBInstanceId,
		rds.DBInstanceId+"__"+rds.DBInstanceDescription,
		"alicloud_db_instance",
		"alicloud",
		map[string]string{},
		[]string{},
		map[string]interface{}{},
	)
}

// InitResources Gets the list of all rds ids and generates resources
func (g *RdsGenerator) InitResources() error {
	client, err := g.LoadClientFromProfile()
	if err != nil {
		return err
	}
	remaining := 1
	pageNumber := 1
	pageSize := 10

	allrdss := make([]rds.DBInstance, 0)

	for remaining > 0 {
		raw, err := client.WithRdsClient(func(rdsClient *rds.Client) (interface{}, error) {
			request := rds.CreateDescribeDBInstancesRequest()
			request.RegionId = client.RegionID
			request.PageSize = requests.NewInteger(pageSize)
			request.PageNumber = requests.NewInteger(pageNumber)
			return rdsClient.DescribeDBInstances(request)
		})
		if err != nil {
			return err
		}

		response := raw.(*rds.DescribeDBInstancesResponse)
		allrdss = append(allrdss, response.Items.DBInstance...)
		remaining = response.TotalRecordCount - pageNumber*pageSize
		pageNumber++
	}

	for _, rds := range allrdss {
		resource := resourceFromrdsResponse(rds)
		g.Resources = append(g.Resources, resource)
	}

	return nil
}

// PostConvertHook Runs before HCL files are generated
func (g *RdsGenerator) PostConvertHook() error {
	for _, r := range g.Resources {
		if r.InstanceInfo.Type == "alicloud_db_instance" {
			// https://www.terraform.io/docs/providers/alicloud/r/db_instance.html#period
			if r.Item["instance_charge_type"] != "PrePaid" {
				delete(r.Item, "period")
			}
		}
	}

	return nil
}
