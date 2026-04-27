// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/mediastore"
	"github.com/chenrui333/terraformer/terraformutils"
)

var mediastoreAllowEmptyValues = []string{"tags."}

type MediaStoreGenerator struct {
	AWSService
}

func (g *MediaStoreGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := mediastore.NewFromConfig(config)
	p := mediastore.NewListContainersPaginator(svc, &mediastore.ListContainersInput{})
	var resources []terraformutils.Resource
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, container := range page.Containers {
			containerName := StringValue(container.Name)
			resources = append(resources, terraformutils.NewSimpleResource(
				containerName,
				containerName,
				"aws_media_store_container",
				"aws",
				mediastoreAllowEmptyValues))
		}
	}
	g.Resources = resources
	return nil
}
