// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/medialive"
	"github.com/chenrui333/terraformer/terraformutils"
)

var medialiveAllowEmptyValues = []string{"tags."}

type MediaLiveGenerator struct {
	AWSService
}

func (g *MediaLiveGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := medialive.NewFromConfig(config)
	g.Resources = []terraformutils.Resource{}

	if err := g.GetChannels(svc); err != nil {
		log.Println(err)
	}

	if err := g.GetInputs(svc); err != nil {
		log.Println(err)
	}

	if err := g.GetInputSecurityGroups(svc); err != nil {
		log.Println(err)
	}

	return nil
}

func (g *MediaLiveGenerator) GetChannels(svc *medialive.Client) error {
	p := medialive.NewListChannelsPaginator(svc, &medialive.ListChannelsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, channel := range page.Channels {
			channelID := StringValue(channel.Id)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				channelID,
				channelID,
				"aws_medialive_channel",
				"aws",
				medialiveAllowEmptyValues))
		}
	}

	return nil
}

func (g *MediaLiveGenerator) GetInputs(svc *medialive.Client) error {
	p := medialive.NewListInputsPaginator(svc, &medialive.ListInputsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, input := range page.Inputs {
			inputID := StringValue(input.Id)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				inputID,
				inputID,
				"aws_medialive_input",
				"aws",
				medialiveAllowEmptyValues))
		}
	}

	return nil
}

func (g *MediaLiveGenerator) GetInputSecurityGroups(svc *medialive.Client) error {
	p := medialive.NewListInputSecurityGroupsPaginator(svc, &medialive.ListInputSecurityGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, inputSecurityGroup := range page.InputSecurityGroups {
			inputSecurityGroupID := StringValue(inputSecurityGroup.Id)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				inputSecurityGroupID,
				inputSecurityGroupID,
				"aws_medialive_input_security_group",
				"aws",
				medialiveAllowEmptyValues))
		}
	}

	return nil
}
