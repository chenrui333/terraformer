// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/servicecatalog"
	"github.com/chenrui333/terraformer/terraformutils"
)

var servicecatalogAllowEmptyValues = []string{"tags."}

type ServiceCatalogGenerator struct {
	AWSService
}

func (g *ServiceCatalogGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := servicecatalog.NewFromConfig(config)
	p := servicecatalog.NewListPortfoliosPaginator(svc, &servicecatalog.ListPortfoliosInput{})
	var resources []terraformutils.Resource
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, portfolio := range page.PortfolioDetails {
			portfolioID := StringValue(portfolio.Id)
			portfolioName := StringValue(portfolio.DisplayName)
			resources = append(resources, terraformutils.NewSimpleResource(
				portfolioID,
				portfolioName,
				"aws_servicecatalog_portfolio",
				"aws",
				servicecatalogAllowEmptyValues))
		}
	}
	g.Resources = resources
	return nil
}
