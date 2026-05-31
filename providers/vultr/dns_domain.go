// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr/v3"
)

type DNSDomainGenerator struct {
	VultrService
}

func (g *DNSDomainGenerator) loadDNSDomains(client *govultr.Client) ([]govultr.Domain, error) {
	domainList, _, _, err := client.Domain.List(context.Background(), nil)
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
	recordList, _, _, err := client.DomainRecord.List(context.Background(), domain, nil)
	if err != nil {
		return err
	}
	for _, record := range recordList {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			record.ID,
			record.ID,
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
