// SPDX-License-Identifier: Apache-2.0

package linode

import (
	"context"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/linode/linodego"
)

type DomainGenerator struct {
	LinodeService
}

func (g *DomainGenerator) loadDomains(client linodego.Client) ([]linodego.Domain, error) {
	domainList, err := client.ListDomains(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	for _, domain := range domainList {
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			strconv.Itoa(domain.ID),
			strconv.Itoa(domain.ID),
			"linode_domain",
			"linode",
			[]string{}))
	}
	return domainList, nil
}

func (g *DomainGenerator) loadDomainRecords(client linodego.Client, domainID int) error {
	domainRecordList, err := client.ListDomainRecords(context.Background(), domainID, nil)
	if err != nil {
		return err
	}
	for _, domainRecord := range domainRecordList {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			strconv.Itoa(domainRecord.ID),
			strconv.Itoa(domainRecord.ID),
			"linode_domain_record",
			"linode",
			map[string]string{"domain_id": strconv.Itoa(domainID)},
			[]string{},
			map[string]interface{}{}))
	}
	return nil
}

func (g *DomainGenerator) InitResources() error {
	client := g.generateClient()
	domainList, err := g.loadDomains(client)
	if err != nil {
		return err
	}
	for _, domain := range domainList {
		err := g.loadDomainRecords(client, domain.ID)
		if err != nil {
			return err
		}
	}
	return nil
}
