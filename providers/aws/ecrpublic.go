// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecrpublic"
	"github.com/aws/aws-sdk-go-v2/service/ecrpublic/types"

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

			repositoryPolicy, err := svc.GetRepositoryPolicy(context.TODO(), &ecrpublic.GetRepositoryPolicyInput{
				RepositoryName: repository.RepositoryName,
				RegistryId:     repository.RegistryId,
			})
			if ecrPublicRepositoryPolicyNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if StringValue(repositoryPolicy.PolicyText) == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				*repository.RepositoryName,
				*repository.RepositoryName,
				"aws_ecrpublic_repository_policy",
				"aws",
				ecrPublicAllowEmptyValues))
		}
	}
	return nil
}

func (g *EcrPublicGenerator) PostConvertHook() error {
	for i, resource := range g.Resources {
		if resource.InstanceInfo.Type != "aws_ecrpublic_repository_policy" {
			continue
		}
		policy, ok := g.Resources[i].Item["policy"].(string)
		if !ok || policy == "" {
			continue
		}
		g.Resources[i].Item["policy"] = fmt.Sprintf("<<POLICY\n%s\nPOLICY", g.escapeAwsInterpolation(policy))
	}
	return nil
}

func ecrPublicRepositoryPolicyNotFound(err error) bool {
	var policyNotFound *types.RepositoryPolicyNotFoundException
	var repositoryNotFound *types.RepositoryNotFoundException
	return errors.As(err, &policyNotFound) || errors.As(err, &repositoryNotFound)
}
