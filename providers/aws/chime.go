// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/chimesdkvoice"
	chimetypes "github.com/aws/aws-sdk-go-v2/service/chimesdkvoice/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	chimeVoiceConnectorResourceType            = "aws_chime_voice_connector"
	chimeVoiceConnectorGroupResourceType       = "aws_chime_voice_connector_group"
	chimeVoiceConnectorLoggingResourceType     = "aws_chime_voice_connector_logging"
	chimeVoiceConnectorOriginationResourceType = "aws_chime_voice_connector_origination"
	chimeVoiceConnectorStreamingResourceType   = "aws_chime_voice_connector_streaming"
	chimeVoiceConnectorTerminationResourceType = "aws_chime_voice_connector_termination"
	chimeVoiceConnectorAWSRegionAttribute      = "aws" + "_region"
)

var chimeAllowEmptyValues = []string{"tags."}

type ChimeGenerator struct {
	AWSService
}

type chimeOptionalResourceLoader struct {
	name string
	load func() error
}

func (g *ChimeGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := chimesdkvoice.NewFromConfig(config)
	if err := g.loadVoiceConnectors(svc); err != nil {
		return err
	}
	return g.loadVoiceConnectorGroups(svc)
}

func (g *ChimeGenerator) loadVoiceConnectors(svc *chimesdkvoice.Client) error {
	p := chimesdkvoice.NewListVoiceConnectorsPaginator(svc, &chimesdkvoice.ListVoiceConnectorsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, connector := range page.VoiceConnectors {
			connectorID := StringValue(connector.VoiceConnectorId)
			if resource, ok := newChimeVoiceConnectorResource(connector); ok {
				g.Resources = append(g.Resources, resource)
			}
			if connectorID == "" {
				continue
			}
			g.getOptionalChimeResources(
				chimeOptionalResourceLoader{name: "voice connector logging", load: func() error {
					return g.loadVoiceConnectorLogging(svc, connectorID)
				}},
				chimeOptionalResourceLoader{name: "voice connector origination", load: func() error {
					return g.loadVoiceConnectorOrigination(svc, connectorID)
				}},
				chimeOptionalResourceLoader{name: "voice connector streaming", load: func() error {
					return g.loadVoiceConnectorStreaming(svc, connectorID)
				}},
				chimeOptionalResourceLoader{name: "voice connector termination", load: func() error {
					return g.loadVoiceConnectorTermination(svc, connectorID)
				}},
			)
		}
	}
	return nil
}

func (g *ChimeGenerator) loadVoiceConnectorGroups(svc *chimesdkvoice.Client) error {
	p := chimesdkvoice.NewListVoiceConnectorGroupsPaginator(svc, &chimesdkvoice.ListVoiceConnectorGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, group := range page.VoiceConnectorGroups {
			if resource, ok := newChimeVoiceConnectorGroupResource(group); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *ChimeGenerator) loadVoiceConnectorLogging(svc *chimesdkvoice.Client, connectorID string) error {
	output, err := svc.GetVoiceConnectorLoggingConfiguration(context.TODO(), &chimesdkvoice.GetVoiceConnectorLoggingConfigurationInput{
		VoiceConnectorId: &connectorID,
	})
	if err != nil {
		if chimeNotFound(err) {
			return nil
		}
		return err
	}
	if resource, ok := newChimeVoiceConnectorLoggingResource(connectorID, output); ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *ChimeGenerator) loadVoiceConnectorOrigination(svc *chimesdkvoice.Client, connectorID string) error {
	output, err := svc.GetVoiceConnectorOrigination(context.TODO(), &chimesdkvoice.GetVoiceConnectorOriginationInput{
		VoiceConnectorId: &connectorID,
	})
	if err != nil {
		if chimeNotFound(err) {
			return nil
		}
		return err
	}
	if resource, ok := newChimeVoiceConnectorOriginationResource(connectorID, output); ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *ChimeGenerator) loadVoiceConnectorStreaming(svc *chimesdkvoice.Client, connectorID string) error {
	output, err := svc.GetVoiceConnectorStreamingConfiguration(context.TODO(), &chimesdkvoice.GetVoiceConnectorStreamingConfigurationInput{
		VoiceConnectorId: &connectorID,
	})
	if err != nil {
		if chimeNotFound(err) {
			return nil
		}
		return err
	}
	if resource, ok := newChimeVoiceConnectorStreamingResource(connectorID, output); ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *ChimeGenerator) loadVoiceConnectorTermination(svc *chimesdkvoice.Client, connectorID string) error {
	output, err := svc.GetVoiceConnectorTermination(context.TODO(), &chimesdkvoice.GetVoiceConnectorTerminationInput{
		VoiceConnectorId: &connectorID,
	})
	if err != nil {
		if chimeNotFound(err) {
			return nil
		}
		return err
	}
	if resource, ok := newChimeVoiceConnectorTerminationResource(connectorID, output); ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *ChimeGenerator) getOptionalChimeResources(loaders ...chimeOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			log.Printf("skipping Chime %s discovery: %v", loader.name, err)
		}
	}
}

func newChimeVoiceConnectorResource(connector chimetypes.VoiceConnector) (terraformutils.Resource, bool) {
	connectorID := StringValue(connector.VoiceConnectorId)
	name := StringValue(connector.Name)
	if connectorID == "" || name == "" || connector.RequireEncryption == nil {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"name":               name,
		"require_encryption": strconv.FormatBool(*connector.RequireEncryption),
	}
	if connector.AwsRegion != "" {
		attributes[chimeVoiceConnectorAWSRegionAttribute] = string(connector.AwsRegion)
	}
	return terraformutils.NewResource(
		chimeVoiceConnectorImportID(connectorID),
		chimeResourceName("voice_connector", name, connectorID),
		chimeVoiceConnectorResourceType,
		"aws",
		attributes,
		chimeAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newChimeVoiceConnectorGroupResource(group chimetypes.VoiceConnectorGroup) (terraformutils.Resource, bool) {
	groupID := StringValue(group.VoiceConnectorGroupId)
	name := StringValue(group.Name)
	if groupID == "" || name == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		chimeVoiceConnectorGroupImportID(groupID),
		chimeResourceName("voice_connector_group", name, groupID),
		chimeVoiceConnectorGroupResourceType,
		"aws",
		map[string]string{"name": name},
		chimeAllowEmptyValues,
		chimeVoiceConnectorGroupAdditionalFields(group),
	), true
}

func newChimeVoiceConnectorLoggingResource(connectorID string, output *chimesdkvoice.GetVoiceConnectorLoggingConfigurationOutput) (terraformutils.Resource, bool) {
	if connectorID == "" || output == nil || output.LoggingConfiguration == nil {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{"voice_connector_id": connectorID}
	if output.LoggingConfiguration.EnableMediaMetricLogs != nil {
		attributes["enable_media_metric_logs"] = strconv.FormatBool(*output.LoggingConfiguration.EnableMediaMetricLogs)
	}
	if output.LoggingConfiguration.EnableSIPLogs != nil {
		attributes["enable_sip_logs"] = strconv.FormatBool(*output.LoggingConfiguration.EnableSIPLogs)
	}
	return terraformutils.NewResource(
		chimeVoiceConnectorLoggingImportID(connectorID),
		chimeResourceName("voice_connector_logging", connectorID),
		chimeVoiceConnectorLoggingResourceType,
		"aws",
		attributes,
		chimeAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newChimeVoiceConnectorOriginationResource(connectorID string, output *chimesdkvoice.GetVoiceConnectorOriginationOutput) (terraformutils.Resource, bool) {
	if connectorID == "" || output == nil || output.Origination == nil || len(output.Origination.Routes) == 0 {
		return terraformutils.Resource{}, false
	}
	routes, ok := chimeOriginationRouteAdditionalFields(output.Origination.Routes)
	if !ok || len(routes) == 0 {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{"voice_connector_id": connectorID}
	if output.Origination.Disabled != nil {
		attributes["disabled"] = strconv.FormatBool(*output.Origination.Disabled)
	}
	return terraformutils.NewResource(
		chimeVoiceConnectorOriginationImportID(connectorID),
		chimeResourceName("voice_connector_origination", connectorID),
		chimeVoiceConnectorOriginationResourceType,
		"aws",
		attributes,
		chimeAllowEmptyValues,
		map[string]interface{}{"route": routes},
	), true
}

func newChimeVoiceConnectorStreamingResource(connectorID string, output *chimesdkvoice.GetVoiceConnectorStreamingConfigurationOutput) (terraformutils.Resource, bool) {
	if connectorID == "" || output == nil || output.StreamingConfiguration == nil || output.StreamingConfiguration.DataRetentionInHours == nil {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"voice_connector_id": connectorID,
		"data_retention":     strconv.Itoa(int(*output.StreamingConfiguration.DataRetentionInHours)),
	}
	if output.StreamingConfiguration.Disabled != nil {
		attributes["disabled"] = strconv.FormatBool(*output.StreamingConfiguration.Disabled)
	}
	return terraformutils.NewResource(
		chimeVoiceConnectorStreamingImportID(connectorID),
		chimeResourceName("voice_connector_streaming", connectorID),
		chimeVoiceConnectorStreamingResourceType,
		"aws",
		attributes,
		chimeAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newChimeVoiceConnectorTerminationResource(connectorID string, output *chimesdkvoice.GetVoiceConnectorTerminationOutput) (terraformutils.Resource, bool) {
	if connectorID == "" || output == nil || output.Termination == nil || len(output.Termination.CallingRegions) == 0 || len(output.Termination.CidrAllowedList) == 0 {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{"voice_connector_id": connectorID}
	if output.Termination.CpsLimit != nil {
		attributes["cps_limit"] = strconv.Itoa(int(*output.Termination.CpsLimit))
	}
	if defaultPhoneNumber := StringValue(output.Termination.DefaultPhoneNumber); defaultPhoneNumber != "" {
		attributes["default_phone_number"] = defaultPhoneNumber
	}
	if output.Termination.Disabled != nil {
		attributes["disabled"] = strconv.FormatBool(*output.Termination.Disabled)
	}
	cidrAllowList, ok := chimeTerminationCIDRAllowList(output.Termination.CidrAllowedList)
	if !ok || len(cidrAllowList) == 0 {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		chimeVoiceConnectorTerminationImportID(connectorID),
		chimeResourceName("voice_connector_termination", connectorID),
		chimeVoiceConnectorTerminationResourceType,
		"aws",
		attributes,
		chimeAllowEmptyValues,
		map[string]interface{}{
			"calling_regions": stringSliceToInterfaceSlice(output.Termination.CallingRegions),
			"cidr_allow_list": cidrAllowList,
		},
	), true
}

func chimeVoiceConnectorGroupAdditionalFields(group chimetypes.VoiceConnectorGroup) map[string]interface{} {
	connectors := make([]interface{}, 0, len(group.VoiceConnectorItems))
	for _, item := range group.VoiceConnectorItems {
		connectorID := StringValue(item.VoiceConnectorId)
		if connectorID == "" || item.Priority == nil {
			continue
		}
		connectors = append(connectors, map[string]interface{}{
			"priority":           int(*item.Priority),
			"voice_connector_id": connectorID,
		})
	}
	if len(connectors) == 0 {
		return map[string]interface{}{}
	}
	return map[string]interface{}{"connector": connectors}
}

func chimeOriginationRouteAdditionalFields(routes []chimetypes.OriginationRoute) ([]interface{}, bool) {
	result := make([]interface{}, 0, len(routes))
	for _, route := range routes {
		host := StringValue(route.Host)
		if host == "" || route.Priority == nil || route.Protocol == "" || route.Weight == nil {
			continue
		}
		if net.ParseIP(host) == nil {
			return nil, false
		}
		item := map[string]interface{}{
			"host":     host,
			"priority": int(*route.Priority),
			"protocol": string(route.Protocol),
			"weight":   int(*route.Weight),
		}
		if route.Port != nil {
			item["port"] = int(*route.Port)
		}
		result = append(result, item)
	}
	return result, true
}

func chimeTerminationCIDRAllowList(cidrList []string) ([]interface{}, bool) {
	result := make([]interface{}, 0, len(cidrList))
	for _, cidr := range cidrList {
		if !chimeProviderValidCIDRNetwork(cidr) {
			return nil, false
		}
		result = append(result, cidr)
	}
	return result, true
}

func chimeProviderValidCIDRNetwork(cidr string) bool {
	address, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	ones, bits := network.Mask.Size()
	if bits != 32 || ones < 27 || ones > 32 {
		return false
	}
	return strings.Split(cidr, "/")[0] == address.String() && address.Equal(network.IP)
}

func chimeVoiceConnectorImportID(connectorID string) string {
	return connectorID
}

func chimeVoiceConnectorGroupImportID(groupID string) string {
	return groupID
}

func chimeVoiceConnectorLoggingImportID(connectorID string) string {
	return connectorID
}

func chimeVoiceConnectorOriginationImportID(connectorID string) string {
	return connectorID
}

func chimeVoiceConnectorStreamingImportID(connectorID string) string {
	return connectorID
}

func chimeVoiceConnectorTerminationImportID(connectorID string) string {
	return connectorID
}

func chimeResourceName(parts ...string) string {
	return resourceNameWithLengthPrefixes(parts...)
}

func chimeNotFound(err error) bool {
	var notFound *chimetypes.NotFoundException
	return errors.As(err, &notFound)
}
