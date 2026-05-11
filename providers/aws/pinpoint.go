// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/pinpoint"
	pinpointtypes "github.com/aws/aws-sdk-go-v2/service/pinpoint/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	pinpointAppResourceType          = "aws_pinpoint_app"
	pinpointEmailChannelResourceType = "aws_pinpoint_email_channel"
	pinpointEventStreamResourceType  = "aws_pinpoint_event_stream"
	pinpointSMSChannelResourceType   = "aws_pinpoint_sms_channel"
)

var pinpointAllowEmptyValues = []string{"tags."}

type PinpointGenerator struct {
	AWSService
}

type pinpointApplicationReference struct {
	id   string
	name string
}

type pinpointOptionalResourceLoader struct {
	name string
	load func() error
}

func (g *PinpointGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := pinpoint.NewFromConfig(config)

	apps, err := g.loadApps(svc)
	if err != nil {
		return err
	}
	for _, app := range apps {
		g.getOptionalPinpointResources(
			pinpointOptionalResourceLoader{name: "email channels", load: func() error {
				return g.loadEmailChannel(svc, app.id, app.name)
			}},
			pinpointOptionalResourceLoader{name: "SMS channels", load: func() error {
				return g.loadSMSChannel(svc, app.id, app.name)
			}},
			pinpointOptionalResourceLoader{name: "event streams", load: func() error {
				return g.loadEventStream(svc, app.id, app.name)
			}},
		)
	}
	return nil
}

func (g *PinpointGenerator) loadApps(svc *pinpoint.Client) ([]pinpointApplicationReference, error) {
	apps := []pinpointApplicationReference{}
	input := &pinpoint.GetAppsInput{}
	for {
		output, err := svc.GetApps(context.TODO(), input)
		if err != nil {
			return nil, err
		}
		if output == nil || output.ApplicationsResponse == nil {
			return apps, nil
		}
		for _, app := range output.ApplicationsResponse.Item {
			resource, ok := newPinpointAppResource(app)
			if !ok {
				continue
			}
			g.Resources = append(g.Resources, resource)
			apps = append(apps, pinpointApplicationReference{
				id:   StringValue(app.Id),
				name: StringValue(app.Name),
			})
		}
		if output.ApplicationsResponse.NextToken == nil || StringValue(output.ApplicationsResponse.NextToken) == "" {
			return apps, nil
		}
		input.Token = output.ApplicationsResponse.NextToken
	}
}

func (g *PinpointGenerator) loadEmailChannel(svc *pinpoint.Client, applicationID, applicationName string) error {
	output, err := svc.GetEmailChannel(context.TODO(), &pinpoint.GetEmailChannelInput{
		ApplicationId: &applicationID,
	})
	if err != nil {
		if pinpointNotFound(err) {
			return nil
		}
		return err
	}
	if output == nil || output.EmailChannelResponse == nil {
		return nil
	}
	if resource, ok := newPinpointEmailChannelResource(applicationID, applicationName, *output.EmailChannelResponse); ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *PinpointGenerator) loadSMSChannel(svc *pinpoint.Client, applicationID, applicationName string) error {
	output, err := svc.GetSmsChannel(context.TODO(), &pinpoint.GetSmsChannelInput{
		ApplicationId: &applicationID,
	})
	if err != nil {
		if pinpointNotFound(err) {
			return nil
		}
		return err
	}
	if output == nil || output.SMSChannelResponse == nil {
		return nil
	}
	if resource, ok := newPinpointSMSChannelResource(applicationID, applicationName, *output.SMSChannelResponse); ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *PinpointGenerator) loadEventStream(svc *pinpoint.Client, applicationID, applicationName string) error {
	output, err := svc.GetEventStream(context.TODO(), &pinpoint.GetEventStreamInput{
		ApplicationId: &applicationID,
	})
	if err != nil {
		if pinpointNotFound(err) {
			return nil
		}
		return err
	}
	if output == nil || output.EventStream == nil {
		return nil
	}
	if resource, ok := newPinpointEventStreamResource(applicationID, applicationName, *output.EventStream); ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *PinpointGenerator) getOptionalPinpointResources(loaders ...pinpointOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			log.Printf("skipping Pinpoint %s discovery: %v", loader.name, err)
		}
	}
}

func newPinpointAppResource(app pinpointtypes.ApplicationResponse) (terraformutils.Resource, bool) {
	applicationID := StringValue(app.Id)
	if applicationID == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"application_id": applicationID,
	}
	if name := StringValue(app.Name); name != "" {
		attributes["name"] = name
	}
	return terraformutils.NewResource(
		pinpointApplicationImportID(applicationID),
		pinpointResourceName("app", StringValue(app.Name), applicationID),
		pinpointAppResourceType,
		"aws",
		attributes,
		pinpointAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newPinpointEmailChannelResource(applicationID, applicationName string, channel pinpointtypes.EmailChannelResponse) (terraformutils.Resource, bool) {
	if applicationID == "" || StringValue(channel.FromAddress) == "" || StringValue(channel.Identity) == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"application_id": applicationID,
		"from_address":   StringValue(channel.FromAddress),
		"identity":       StringValue(channel.Identity),
	}
	if channel.Enabled != nil {
		attributes["enabled"] = strconv.FormatBool(*channel.Enabled)
	}
	return terraformutils.NewResource(
		pinpointApplicationImportID(applicationID),
		pinpointResourceName("email_channel", applicationName, applicationID),
		pinpointEmailChannelResourceType,
		"aws",
		attributes,
		pinpointAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newPinpointSMSChannelResource(applicationID, applicationName string, channel pinpointtypes.SMSChannelResponse) (terraformutils.Resource, bool) {
	if applicationID == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"application_id": applicationID,
	}
	if channel.Enabled != nil {
		attributes["enabled"] = strconv.FormatBool(*channel.Enabled)
	}
	if senderID := StringValue(channel.SenderId); senderID != "" {
		attributes["sender_id"] = senderID
	}
	if shortCode := StringValue(channel.ShortCode); shortCode != "" {
		attributes["short_code"] = shortCode
	}
	return terraformutils.NewResource(
		pinpointApplicationImportID(applicationID),
		pinpointResourceName("sms_channel", applicationName, applicationID),
		pinpointSMSChannelResourceType,
		"aws",
		attributes,
		pinpointAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newPinpointEventStreamResource(applicationID, applicationName string, eventStream pinpointtypes.EventStream) (terraformutils.Resource, bool) {
	if applicationID == "" || StringValue(eventStream.DestinationStreamArn) == "" || StringValue(eventStream.RoleArn) == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		pinpointApplicationImportID(applicationID),
		pinpointResourceName("event_stream", applicationName, applicationID),
		pinpointEventStreamResourceType,
		"aws",
		map[string]string{
			"application_id":         applicationID,
			"destination_stream_arn": StringValue(eventStream.DestinationStreamArn),
			"role_arn":               StringValue(eventStream.RoleArn),
		},
		pinpointAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func pinpointApplicationImportID(applicationID string) string {
	return applicationID
}

func pinpointResourceName(parts ...string) string {
	var name strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		if name.Len() > 0 {
			name.WriteString("_")
		}
		name.WriteString(strconv.Itoa(len(part)))
		name.WriteString("_")
		name.WriteString(part)
	}
	return name.String()
}

func pinpointNotFound(err error) bool {
	var notFound *pinpointtypes.NotFoundException
	return errors.As(err, &notFound)
}
