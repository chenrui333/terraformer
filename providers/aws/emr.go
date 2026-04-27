// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/emr"
	"github.com/chenrui333/terraformer/terraformutils"
)

var emrAllowEmptyValues = []string{"tags."}

type EmrGenerator struct {
	AWSService
}

func (g *EmrGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	client := emr.NewFromConfig(config)

	err := g.addClusters(client)
	if err != nil {
		return err
	}
	err = g.addSecurityConfigurations(client)
	return err
}

func (g *EmrGenerator) addClusters(client *emr.Client) error {
	p := emr.NewListClustersPaginator(client, &emr.ListClustersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, cluster := range page.Clusters {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				*cluster.Id,
				*cluster.Name,
				"aws_emr_cluster",
				"aws",
				emrAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *EmrGenerator) addSecurityConfigurations(client *emr.Client) error {
	p := emr.NewListSecurityConfigurationsPaginator(client, &emr.ListSecurityConfigurationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, securityConfiguration := range page.SecurityConfigurations {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				*securityConfiguration.Name,
				*securityConfiguration.Name,
				"aws_emr_security_configuration",
				"aws",
				emrAllowEmptyValues,
			))
		}
	}
	return nil
}
