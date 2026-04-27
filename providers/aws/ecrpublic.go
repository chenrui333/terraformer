// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecrpublic"

	"github.com/chenrui333/terraformer/terraformutils"
)

var ecrPublicAllowEmptyValues = []string{"tags."}

type EcrPublicGenerator struct {
	AWSService
}

func (g *EcrPublicGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}

	ecrPublicConfig := config.Copy()
	ecrPublicConfig.Region = MainRegionPublicPartition
	svc := ecrpublic.NewFromConfig(ecrPublicConfig)

	p := ecrpublic.NewDescribeRepositoriesPaginator(svc, &ecrpublic.DescribeRepositoriesInput{})
	for p.HasMorePages() {
		page, e := p.NextPage(context.TODO())
		if e != nil {
			return e
		}
		for _, repository := range page.Repositories {
			resource := terraformutils.NewSimpleResource(
				*repository.RepositoryName,
				*repository.RepositoryName,
				"aws_ecrpublic_repository",
				"aws",
				ecrPublicAllowEmptyValues)
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}
