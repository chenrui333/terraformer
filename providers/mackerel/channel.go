// SPDX-License-Identifier: Apache-2.0

package mackerel

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/mackerelio/mackerel-client-go"
)

// ChannelGenerator ...
type ChannelGenerator struct {
	MackerelService
}

func (g *ChannelGenerator) createResources(channels []*mackerel.Channel) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, channel := range channels {
		if channel.Type != "email" && channel.Type != "slack" && channel.Type != "webhook" {
			continue
		}

		if channel.Type == "email" {
			if channel.Events != nil {
				events := *channel.Events
				for _, event := range events {
					if event != "alert" && event != "alertGroup" {
						continue
					}
				}
			} else {
				continue
			}
		}

		resources = append(resources, g.createResource(channel.ID))
	}
	return resources
}

func (g *ChannelGenerator) createResource(channelID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		channelID,
		fmt.Sprintf("channel_%s", channelID),
		"mackerel_channel",
		"mackerel",
		[]string{},
	)
}

// InitResources Generate TerraformResources from Mackerel API,
// from each channel create 1 TerraformResource.
// Need Channel ID as ID for terraform resource
func (g *ChannelGenerator) InitResources() error {
	client := g.Args["mackerelClient"].(*mackerel.Client)
	channels, err := client.FindChannels()
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, g.createResources(channels)...)
	return nil
}
