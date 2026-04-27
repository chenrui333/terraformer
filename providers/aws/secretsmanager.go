// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/chenrui333/terraformer/terraformutils"
)

var secretsmanagerAllowEmptyValues = []string{"tags."}

type SecretsManagerGenerator struct {
	AWSService
}

func (g *SecretsManagerGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := secretsmanager.NewFromConfig(config)
	p := secretsmanager.NewListSecretsPaginator(svc, &secretsmanager.ListSecretsInput{})
	var resources []terraformutils.Resource
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, secret := range page.SecretList {
			secretArn := StringValue(secret.ARN)
			secretName := StringValue(secret.Name)
			resources = append(resources, terraformutils.NewSimpleResource(
				secretArn,
				secretName,
				"aws_secretsmanager_secret",
				"aws",
				secretsmanagerAllowEmptyValues))
		}
	}
	g.Resources = resources
	return nil
}
