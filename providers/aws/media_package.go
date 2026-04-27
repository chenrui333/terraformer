// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/mediapackage"
	"github.com/chenrui333/terraformer/terraformutils"
)

var mediapackageAllowEmptyValues = []string{"tags."}

type MediaPackageGenerator struct {
	AWSService
}

func (g *MediaPackageGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := mediapackage.NewFromConfig(config)
	p := mediapackage.NewListChannelsPaginator(svc, &mediapackage.ListChannelsInput{})
	var resources []terraformutils.Resource
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, channel := range page.Channels {
			channelID := StringValue(channel.Id)
			resources = append(resources, terraformutils.NewSimpleResource(
				channelID,
				channelID,
				"aws_media_package_channel",
				"aws",
				mediapackageAllowEmptyValues))
		}
	}
	g.Resources = resources
	return nil
}
