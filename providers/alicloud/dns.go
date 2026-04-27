// SPDX-License-Identifier: Apache-2.0

package alicloud

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"github.com/chenrui333/terraformer/providers/alicloud/connectivity"
	"github.com/chenrui333/terraformer/terraformutils"
)

// DNSGenerator Struct for generating AliCloud Elastic Compute Service
type DNSGenerator struct {
	AliCloudService
}

func resourceFromDomain(domain alidns.DomainInDescribeDomains) terraformutils.Resource {
	return terraformutils.NewResource(
		domain.DomainName, // id
		domain.DomainId+"__"+domain.DomainName,
		"alicloud_alidns_domain",
		"alicloud",
		map[string]string{},
		[]string{},
		map[string]interface{}{},
	)
}

func resourceFromDomainRecord(record alidns.Record) terraformutils.Resource {
	return terraformutils.NewResource(
		record.RecordId, // id
		record.RecordId+"__"+record.DomainName,
		"alicloud_alidns_record",
		"alicloud",
		map[string]string{},
		[]string{},
		map[string]interface{}{},
	)
}

func initDomains(client *connectivity.AliyunClient) ([]alidns.DomainInDescribeDomains, error) {
	remaining := 1
	pageNumber := 1
	pageSize := 10

	allDomains := make([]alidns.DomainInDescribeDomains, 0)

	for remaining > 0 {
		raw, err := client.WithDNSClient(func(alidnsClient *alidns.Client) (interface{}, error) {
			request := alidns.CreateDescribeDomainsRequest()
			request.RegionId = client.RegionID
			request.PageSize = requests.NewInteger(pageSize)
			request.PageNumber = requests.NewInteger(pageNumber)
			return alidnsClient.DescribeDomains(request)
		})
		if err != nil {
			return nil, err
		}

		response := raw.(*alidns.DescribeDomainsResponse)
		allDomains = append(allDomains, response.Domains.Domain...)
		remaining = int(response.TotalCount) - pageNumber*pageSize
		pageNumber++
	}

	return allDomains, nil
}

func initDomainRecords(client *connectivity.AliyunClient, allDomains []alidns.DomainInDescribeDomains) ([]alidns.Record, error) {
	allDomainRecords := make([]alidns.Record, 0)

	for _, domain := range allDomains {
		remaining := 1
		pageNumber := 1
		pageSize := 10

		for remaining > 0 {
			raw, err := client.WithDNSClient(func(alidnsClient *alidns.Client) (interface{}, error) {
				request := alidns.CreateDescribeDomainRecordsRequest()
				request.RegionId = client.RegionID
				request.DomainName = domain.DomainName
				request.PageSize = requests.NewInteger(pageSize)
				request.PageNumber = requests.NewInteger(pageNumber)
				return alidnsClient.DescribeDomainRecords(request)
			})
			if err != nil {
				return nil, err
			}

			response := raw.(*alidns.DescribeDomainRecordsResponse)
			allDomainRecords = append(allDomainRecords, response.DomainRecords.Record...)
			remaining = int(response.TotalCount) - pageNumber*pageSize
			pageNumber++
		}
	}

	return allDomainRecords, nil
}

// InitResources Gets the list of all alidns domain ids and generates resources
func (g *DNSGenerator) InitResources() error {
	client, err := g.LoadClientFromProfile()
	if err != nil {
		return err
	}

	allDomains, err := initDomains(client)
	if err != nil {
		return err
	}

	allDomainRecords, err := initDomainRecords(client, allDomains)
	if err != nil {
		return err
	}

	for _, domain := range allDomains {
		resource := resourceFromDomain(domain)
		g.Resources = append(g.Resources, resource)
	}

	for _, record := range allDomainRecords {
		resource := resourceFromDomainRecord(record)
		g.Resources = append(g.Resources, resource)
	}

	return nil
}
