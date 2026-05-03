package opal

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	opalsdk "github.com/opalsecurity/opal-go"
)

type MessageChannelGenerator struct {
	OpalService
}

func (g *MessageChannelGenerator) createResources(messageChannels []opalsdk.MessageChannel) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	countByName := make(map[string]int)

	for _, channel := range messageChannels {
		resourceID, err := opalRequiredString("opal_message_channel", "message_channel_id", channel.MessageChannelId)
		if err != nil {
			return nil, err
		}
		name := opalUniqueResourceName(opalResourceDisplayName(channel.Name, resourceID), countByName)

		resources = append(resources, terraformutils.NewSimpleResource(
			resourceID,
			name,
			"opal_message_channel",
			"opal",
			[]string{},
		))
	}

	return resources, nil
}

func (g *MessageChannelGenerator) InitResources() error {
	client, err := g.newClient()
	if err != nil {
		return fmt.Errorf("unable to list opal message channels: %w", err)
	}

	messageChannels, _, err := client.MessageChannelsAPI.GetMessageChannels(context.TODO()).Execute()
	if err != nil {
		return fmt.Errorf("unable to list opal message channels: %w", err)
	}

	resources, err := g.createResources(messageChannels.Channels)
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, resources...)

	return nil
}
