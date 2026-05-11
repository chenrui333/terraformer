// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/mediapackagev2"
	mediapackagev2types "github.com/aws/aws-sdk-go-v2/service/mediapackagev2/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const mediaPackageV2ChannelGroupResourceType = "aws_media_packagev2_channel_group"

var mediaPackageV2AllowEmptyValues = []string{"tags."}

type MediaPackageV2Generator struct {
	AWSService
}

func (g *MediaPackageV2Generator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := mediapackagev2.NewFromConfig(config)
	return g.GetChannelGroups(svc)
}

func (g *MediaPackageV2Generator) GetChannelGroups(svc *mediapackagev2.Client) error {
	p := mediapackagev2.NewListChannelGroupsPaginator(svc, &mediapackagev2.ListChannelGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if mediaPackageV2ResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, channelGroup := range page.Items {
			if resource, ok := newMediaPackageV2ChannelGroupResource(channelGroup); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}

	return nil
}

func newMediaPackageV2ChannelGroupResource(channelGroup mediapackagev2types.ChannelGroupListConfiguration) (terraformutils.Resource, bool) {
	channelGroupName := StringValue(channelGroup.ChannelGroupName)
	if channelGroupName == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		mediaPackageV2ChannelGroupImportID(channelGroupName),
		mediaPackageV2ResourceName("channel-group", channelGroupName),
		mediaPackageV2ChannelGroupResourceType,
		"aws",
		mediaPackageV2AllowEmptyValues,
	), true
}

func mediaPackageV2ChannelGroupImportID(channelGroupName string) string {
	return channelGroupName
}

func mediaPackageV2ResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d/%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return "media-packagev2-resource"
	}
	return strings.Join(cleanParts, "/")
}

func mediaPackageV2ResourceNotFound(err error) bool {
	var notFound *mediapackagev2types.ResourceNotFoundException
	return errors.As(err, &notFound)
}
