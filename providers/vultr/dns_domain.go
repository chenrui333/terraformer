// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr"
)

type DNSDomainGenerator struct {
	VultrService
}

func (g *DNSDomainGenerator) loadDNSDomains(client *govultr.Client) ([]govultr.DNSDomain, error) {
	domainList, err := client.DNSDomain.List(context.Background())
	if err != nil {
		return nil, err
	}
	for _, domain := range domainList {
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			domain.Domain,
			domain.Domain,
			"vultr_dns_domain",
			"vultr",
			[]string{}))
	}
	return domainList, nil
}

func (g *DNSDomainGenerator) loadDNSRecords(client *govultr.Client, domain string) error {
	recordList, err := client.DNSRecord.List(context.Background(), domain)
	if err != nil {
		return err
	}
	for _, record := range recordList {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			strconv.Itoa(record.RecordID),
			strconv.Itoa(record.RecordID),
			"vultr_dns_record",
			"vultr",
			map[string]string{"domain": domain},
			[]string{},
			map[string]interface{}{}))
	}
	return nil
}

func (g *DNSDomainGenerator) InitResources() error {
	client := g.generateClient()
	domainList, err := g.loadDNSDomains(client)
	if err != nil {
		return err
	}
	for _, domain := range domainList {
		err := g.loadDNSRecords(client, domain.Domain)
		if err != nil {
			return err
		}
	}
	return nil
}
