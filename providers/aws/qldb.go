// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/qldb"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	qldbLedgerResourceType = "aws_qldb_ledger"
)

var qldbAllowEmptyValues = []string{"tags."}

type QLDBGenerator struct {
	AWSService
}

func (g *QLDBGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := qldb.NewFromConfig(config)
	return g.loadLedgers(svc)
}

func (g *QLDBGenerator) loadLedgers(svc *qldb.Client) error {
	ledgerIDFilter := awsTypedIDFilterValues(g.Filter, qldbLedgerResourceType)
	p := qldb.NewListLedgersPaginator(svc, &qldb.ListLedgersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, ledger := range page.Ledgers {
			ledgerName := StringValue(ledger.Name)
			if !awsIDFilterAllows(ledgerIDFilter, ledgerName) {
				continue
			}
			if resource, ok := newQLDBLedgerResource(ledgerName); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newQLDBLedgerResource(ledgerName string) (terraformutils.Resource, bool) {
	if ledgerName == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		qldbLedgerImportID(ledgerName),
		ledgerName,
		qldbLedgerResourceType,
		"aws",
		qldbAllowEmptyValues), true
}

func qldbLedgerImportID(ledgerName string) string {
	return ledgerName
}
