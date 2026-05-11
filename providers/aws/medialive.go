// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/medialive"
	medialivetypes "github.com/aws/aws-sdk-go-v2/service/medialive/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	mediaLiveChannelResourceType            = "aws_medialive_channel"
	mediaLiveInputResourceType              = "aws_medialive_input"
	mediaLiveInputSecurityGroupResourceType = "aws_medialive_input_security_group"
	mediaLiveMultiplexResourceType          = "aws_medialive_multiplex"
	mediaLiveMultiplexProgramResourceType   = "aws_medialive_multiplex_program"
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
		return err
	}

	if err := g.GetInputs(svc); err != nil {
		return err
	}

	if err := g.GetInputSecurityGroups(svc); err != nil {
		return err
	}

	// Keep multiplex discovery optional so follow-up coverage does not block existing MediaLive imports.
	if err := g.GetMultiplexes(svc); err != nil {
		if mediaLiveResourceNotFound(err) {
			return nil
		}
		log.Printf("skipping optional MediaLive multiplex discovery: %v", err)
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
				mediaLiveChannelResourceType,
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
				mediaLiveInputResourceType,
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
				mediaLiveInputSecurityGroupResourceType,
				"aws",
				medialiveAllowEmptyValues))
		}
	}

	return nil
}

func (g *MediaLiveGenerator) GetMultiplexes(svc *medialive.Client) error {
	p := medialive.NewListMultiplexesPaginator(svc, &medialive.ListMultiplexesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, multiplex := range page.Multiplexes {
			multiplexID := StringValue(multiplex.Id)
			if resource, ok := newMediaLiveMultiplexResource(multiplex); ok {
				g.Resources = append(g.Resources, resource)
				if err := g.GetMultiplexPrograms(svc, multiplexID); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (g *MediaLiveGenerator) GetMultiplexPrograms(svc *medialive.Client, multiplexID string) error {
	if multiplexID == "" {
		return nil
	}
	p := medialive.NewListMultiplexProgramsPaginator(svc, &medialive.ListMultiplexProgramsInput{
		MultiplexId: &multiplexID,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if mediaLiveResourceNotFound(err) {
			return nil
		}
		if err != nil {
			log.Printf("skipping optional MediaLive multiplex program discovery for %s: %v", multiplexID, err)
			return nil
		}
		for _, program := range page.MultiplexPrograms {
			if resource, ok := newMediaLiveMultiplexProgramResource(multiplexID, program); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}

	return nil
}

func newMediaLiveMultiplexResource(multiplex medialivetypes.MultiplexSummary) (terraformutils.Resource, bool) {
	multiplexID := StringValue(multiplex.Id)
	if multiplexID == "" || !mediaLiveMultiplexStateImportable(multiplex.State) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		mediaLiveMultiplexImportID(multiplexID),
		mediaLiveResourceName("multiplex", StringValue(multiplex.Name), multiplexID),
		mediaLiveMultiplexResourceType,
		"aws",
		medialiveAllowEmptyValues,
	), true
}

func newMediaLiveMultiplexProgramResource(multiplexID string, program medialivetypes.MultiplexProgramSummary) (terraformutils.Resource, bool) {
	programName := StringValue(program.ProgramName)
	if multiplexID == "" || programName == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		mediaLiveMultiplexProgramImportID(programName, multiplexID),
		mediaLiveResourceName("multiplex-program", multiplexID, programName),
		mediaLiveMultiplexProgramResourceType,
		"aws",
		map[string]string{
			"multiplex_id": multiplexID,
			"program_name": programName,
		},
		medialiveAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func mediaLiveMultiplexImportID(multiplexID string) string {
	return multiplexID
}

func mediaLiveMultiplexProgramImportID(programName, multiplexID string) string {
	return fmt.Sprintf("%s/%s", programName, multiplexID)
}

func mediaLiveMultiplexStateImportable(state medialivetypes.MultiplexState) bool {
	return state != medialivetypes.MultiplexStateDeleting && state != medialivetypes.MultiplexStateDeleted
}

func mediaLiveResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d/%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return "medialive-resource"
	}
	return strings.Join(cleanParts, "/")
}

func mediaLiveResourceNotFound(err error) bool {
	var notFound *medialivetypes.NotFoundException
	return errors.As(err, &notFound)
}
