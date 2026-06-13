// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"
	"fmt"
	"net/http"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr/v3"
)

type DNSDomainGenerator struct {
	VultrService
}

func (g *DNSDomainGenerator) loadDNSDomains(client *govultr.Client) ([]govultr.Domain, error) {
	domainList, err := listAllVultrResources(context.Background(), client.Domain.List)
	if err != nil {
		return nil, fmt.Errorf("list vultr DNS domains: %w", err)
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
	recordList, err := listAllVultrResources(context.Background(), func(ctx context.Context, opt *govultr.ListOptions) ([]govultr.DomainRecord, *govultr.Meta, *http.Response, error) {
		return client.DomainRecord.List(ctx, domain, opt)
	})
	if err != nil {
		return fmt.Errorf("list vultr DNS records for %q: %w", domain, err)
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
	client, err := g.generateClient()
	if err != nil {
		return err
	}
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
