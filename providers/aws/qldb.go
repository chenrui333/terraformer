// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/qldb"
	"github.com/chenrui333/terraformer/terraformutils"
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
	p := qldb.NewListLedgersPaginator(svc, &qldb.ListLedgersInput{})
	var resources []terraformutils.Resource
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, ledger := range page.Ledgers {
			ledgerName := StringValue(ledger.Name)
			resources = append(resources, terraformutils.NewSimpleResource(
				ledgerName,
				ledgerName,
				"aws_qldb_ledger",
				"aws",
				qldbAllowEmptyValues))
		}
	}
	g.Resources = resources
	return nil
}
