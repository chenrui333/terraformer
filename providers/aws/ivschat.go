// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ivschat"
	ivschattypes "github.com/aws/aws-sdk-go-v2/service/ivschat/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	ivsChatLoggingConfigurationResourceType = "aws_ivschat_logging_configuration"
	ivsChatRoomResourceType                 = "aws_ivschat_room"
)

var ivsChatAllowEmptyValues = []string{"tags."}

type IvsChatGenerator struct {
	AWSService
}

func (g *IvsChatGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := ivschat.NewFromConfig(config)

	if err := g.loadLoggingConfigurations(svc); err != nil {
		return err
	}
	return g.loadRooms(svc)
}

func (g *IvsChatGenerator) loadLoggingConfigurations(svc *ivschat.Client) error {
	paginator := ivschat.NewListLoggingConfigurationsPaginator(svc, &ivschat.ListLoggingConfigurationsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, summary := range page.LoggingConfigurations {
			configuration, err := getIvsChatLoggingConfiguration(svc, StringValue(summary.Arn))
			if ivsChatResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newIvsChatLoggingConfigurationResource(configuration); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *IvsChatGenerator) loadRooms(svc *ivschat.Client) error {
	paginator := ivschat.NewListRoomsPaginator(svc, &ivschat.ListRoomsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, summary := range page.Rooms {
			room, err := getIvsChatRoom(svc, StringValue(summary.Arn))
			if ivsChatResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newIvsChatRoomResource(room); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func getIvsChatLoggingConfiguration(svc *ivschat.Client, arn string) (*ivschat.GetLoggingConfigurationOutput, error) {
	if arn == "" {
		return nil, nil
	}
	return svc.GetLoggingConfiguration(context.TODO(), &ivschat.GetLoggingConfigurationInput{
		Identifier: aws.String(arn),
	})
}

func getIvsChatRoom(svc *ivschat.Client, arn string) (*ivschat.GetRoomOutput, error) {
	if arn == "" {
		return nil, nil
	}
	return svc.GetRoom(context.TODO(), &ivschat.GetRoomInput{
		Identifier: aws.String(arn),
	})
}

func newIvsChatLoggingConfigurationResource(configuration *ivschat.GetLoggingConfigurationOutput) (terraformutils.Resource, bool) {
	if !ivsChatLoggingConfigurationImportable(configuration) {
		return terraformutils.Resource{}, false
	}
	arn := StringValue(configuration.Arn)
	return terraformutils.NewSimpleResource(
		ivsChatARNImportID(arn),
		ivsChatResourceName("logging-configuration", StringValue(configuration.Name), arn),
		ivsChatLoggingConfigurationResourceType,
		"aws",
		ivsChatAllowEmptyValues,
	), true
}

func newIvsChatRoomResource(room *ivschat.GetRoomOutput) (terraformutils.Resource, bool) {
	if room == nil {
		return terraformutils.Resource{}, false
	}
	arn := StringValue(room.Arn)
	if arn == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		ivsChatARNImportID(arn),
		ivsChatResourceName("room", StringValue(room.Name), arn),
		ivsChatRoomResourceType,
		"aws",
		ivsChatAllowEmptyValues,
	), true
}

func ivsChatLoggingConfigurationImportable(configuration *ivschat.GetLoggingConfigurationOutput) bool {
	if configuration == nil || !ivsChatLoggingConfigurationStateImportable(configuration.State) {
		return false
	}
	return StringValue(configuration.Arn) != ""
}

func ivsChatARNImportID(arn string) string {
	return arn
}

func ivsChatLoggingConfigurationStateImportable(state ivschattypes.LoggingConfigurationState) bool {
	switch state {
	case "", ivschattypes.LoggingConfigurationStateDeleting:
		return false
	default:
		return true
	}
}

func ivsChatResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d/%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return "ivschat-resource"
	}
	return strings.Join(cleanParts, "/")
}

func ivsChatResourceNotFound(err error) bool {
	var notFound *ivschattypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
