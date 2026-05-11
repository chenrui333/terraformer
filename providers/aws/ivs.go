// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ivs"
	ivstypes "github.com/aws/aws-sdk-go-v2/service/ivs/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	ivsChannelResourceType                = "aws_ivs_channel"
	ivsRecordingConfigurationResourceType = "aws_ivs_recording_configuration"
)

var ivsAllowEmptyValues = []string{"tags."}

type IvsGenerator struct {
	AWSService
}

func (g *IvsGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := ivs.NewFromConfig(config)

	if err := g.loadChannels(svc); err != nil {
		return err
	}
	return g.loadRecordingConfigurations(svc)
}

func (g *IvsGenerator) loadChannels(svc *ivs.Client) error {
	paginator := ivs.NewListChannelsPaginator(svc, &ivs.ListChannelsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, summary := range page.Channels {
			channel, err := getIvsChannel(svc, StringValue(summary.Arn))
			if ivsResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newIvsChannelResource(channel); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *IvsGenerator) loadRecordingConfigurations(svc *ivs.Client) error {
	paginator := ivs.NewListRecordingConfigurationsPaginator(svc, &ivs.ListRecordingConfigurationsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, summary := range page.RecordingConfigurations {
			recordingConfiguration, err := getIvsRecordingConfiguration(svc, StringValue(summary.Arn))
			if ivsResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newIvsRecordingConfigurationResource(recordingConfiguration); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func getIvsChannel(svc *ivs.Client, arn string) (*ivstypes.Channel, error) {
	if arn == "" {
		return nil, nil
	}
	output, err := svc.GetChannel(context.TODO(), &ivs.GetChannelInput{
		Arn: aws.String(arn),
	})
	if err != nil {
		return nil, err
	}
	if output == nil {
		return nil, nil
	}
	return output.Channel, nil
}

func getIvsRecordingConfiguration(svc *ivs.Client, arn string) (*ivstypes.RecordingConfiguration, error) {
	if arn == "" {
		return nil, nil
	}
	output, err := svc.GetRecordingConfiguration(context.TODO(), &ivs.GetRecordingConfigurationInput{
		Arn: aws.String(arn),
	})
	if err != nil {
		return nil, err
	}
	if output == nil {
		return nil, nil
	}
	return output.RecordingConfiguration, nil
}

func newIvsChannelResource(channel *ivstypes.Channel) (terraformutils.Resource, bool) {
	if channel == nil {
		return terraformutils.Resource{}, false
	}
	arn := StringValue(channel.Arn)
	if arn == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		ivsARNImportID(arn),
		ivsResourceName("channel", StringValue(channel.Name), arn),
		ivsChannelResourceType,
		"aws",
		ivsAllowEmptyValues,
	), true
}

func newIvsRecordingConfigurationResource(recordingConfiguration *ivstypes.RecordingConfiguration) (terraformutils.Resource, bool) {
	if !ivsRecordingConfigurationImportable(recordingConfiguration) {
		return terraformutils.Resource{}, false
	}
	arn := StringValue(recordingConfiguration.Arn)
	return terraformutils.NewResource(
		ivsARNImportID(arn),
		ivsResourceName("recording-configuration", StringValue(recordingConfiguration.Name), arn),
		ivsRecordingConfigurationResourceType,
		"aws",
		ivsRecordingConfigurationAttributes(recordingConfiguration),
		ivsAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func ivsRecordingConfigurationImportable(recordingConfiguration *ivstypes.RecordingConfiguration) bool {
	if recordingConfiguration == nil || !ivsRecordingConfigurationStateImportable(recordingConfiguration.State) {
		return false
	}
	return StringValue(recordingConfiguration.Arn) != "" && ivsRecordingConfigurationBucketName(recordingConfiguration) != ""
}

func ivsRecordingConfigurationAttributes(recordingConfiguration *ivstypes.RecordingConfiguration) map[string]string {
	return map[string]string{
		"destination_configuration.#":                  "1",
		"destination_configuration.0.s3.#":             "1",
		"destination_configuration.0.s3.0.bucket_name": ivsRecordingConfigurationBucketName(recordingConfiguration),
	}
}

func ivsRecordingConfigurationBucketName(recordingConfiguration *ivstypes.RecordingConfiguration) string {
	if recordingConfiguration == nil || recordingConfiguration.DestinationConfiguration == nil || recordingConfiguration.DestinationConfiguration.S3 == nil {
		return ""
	}
	return StringValue(recordingConfiguration.DestinationConfiguration.S3.BucketName)
}

func ivsARNImportID(arn string) string {
	return arn
}

func ivsRecordingConfigurationStateImportable(state ivstypes.RecordingConfigurationState) bool {
	return state != ""
}

func ivsResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d/%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return "ivs-resource"
	}
	return strings.Join(cleanParts, "/")
}

func ivsResourceNotFound(err error) bool {
	var notFound *ivstypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
