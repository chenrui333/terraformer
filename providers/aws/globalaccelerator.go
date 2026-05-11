// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/globalaccelerator"
	globalacceleratortypes "github.com/aws/aws-sdk-go-v2/service/globalaccelerator/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	globalAcceleratorAcceleratorResourceType                = "aws_globalaccelerator_accelerator"
	globalAcceleratorListenerResourceType                   = "aws_globalaccelerator_listener"
	globalAcceleratorEndpointGroupResourceType              = "aws_globalaccelerator_endpoint_group"
	globalAcceleratorCustomRoutingAcceleratorResourceType   = "aws_globalaccelerator_custom_routing_accelerator"
	globalAcceleratorCustomRoutingListenerResourceType      = "aws_globalaccelerator_custom_routing_listener"
	globalAcceleratorCustomRoutingEndpointGroupResourceType = "aws_globalaccelerator_custom_routing_endpoint_group"
	globalAcceleratorCrossAccountAttachmentResourceType     = "aws_globalaccelerator_cross_account_attachment"
	globalAcceleratorControlPlaneRegion                     = "us-west-2"
	globalAcceleratorListenerARNPart                        = "/listener/"
	globalAcceleratorEndpointGroupARNPart                   = "/endpoint-group/"
)

var globalAcceleratorAllowEmptyValues = []string{"tags."}

type globalAcceleratorOptionalResourceLoader struct {
	name string
	load func() error
}

type GlobalAcceleratorGenerator struct {
	AWSService
}

func (g *GlobalAcceleratorGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := globalaccelerator.NewFromConfig(globalAcceleratorClientConfig(config))

	if err := g.loadAccelerators(svc); err != nil {
		return err
	}

	g.loadOptionalResources([]globalAcceleratorOptionalResourceLoader{
		{name: "custom routing accelerators", load: func() error { return g.loadCustomRoutingAccelerators(svc) }},
		{name: "cross-account attachments", load: func() error { return g.loadCrossAccountAttachments(svc) }},
	})

	return nil
}

func (g *GlobalAcceleratorGenerator) loadOptionalResources(loaders []globalAcceleratorOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			if globalAcceleratorResourceNotFound(err) {
				continue
			}
			log.Printf("Skipping Global Accelerator %s: %v", loader.name, err)
		}
	}
}

func globalAcceleratorClientConfig(config aws.Config) aws.Config {
	globalAcceleratorConfig := config.Copy()
	globalAcceleratorConfig.Region = globalAcceleratorControlPlaneRegion
	return globalAcceleratorConfig
}

func (g *GlobalAcceleratorGenerator) loadAccelerators(svc *globalaccelerator.Client) error {
	p := globalaccelerator.NewListAcceleratorsPaginator(svc, &globalaccelerator.ListAcceleratorsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, acceleratorSummary := range page.Accelerators {
			acceleratorARN := StringValue(acceleratorSummary.AcceleratorArn)
			if acceleratorARN == "" {
				continue
			}
			accelerator, err := getGlobalAcceleratorAccelerator(svc, acceleratorARN)
			if globalAcceleratorResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newGlobalAcceleratorAcceleratorResource(accelerator); ok {
				g.Resources = append(g.Resources, resource)
			}
			if globalAcceleratorAcceleratorImportable(accelerator) {
				if err := g.loadListeners(svc, acceleratorARN); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (g *GlobalAcceleratorGenerator) loadListeners(svc *globalaccelerator.Client, acceleratorARN string) error {
	if acceleratorARN == "" {
		return nil
	}
	p := globalaccelerator.NewListListenersPaginator(svc, &globalaccelerator.ListListenersInput{
		AcceleratorArn: aws.String(acceleratorARN),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if globalAcceleratorResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, listenerSummary := range page.Listeners {
			listenerARN := StringValue(listenerSummary.ListenerArn)
			if listenerARN == "" {
				continue
			}
			listener, err := getGlobalAcceleratorListener(svc, listenerARN)
			if globalAcceleratorResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newGlobalAcceleratorListenerResource(listener); ok {
				g.Resources = append(g.Resources, resource)
			}
			if globalAcceleratorListenerImportable(listener) {
				if err := g.loadEndpointGroups(svc, listenerARN); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (g *GlobalAcceleratorGenerator) loadEndpointGroups(svc *globalaccelerator.Client, listenerARN string) error {
	if listenerARN == "" {
		return nil
	}
	p := globalaccelerator.NewListEndpointGroupsPaginator(svc, &globalaccelerator.ListEndpointGroupsInput{
		ListenerArn: aws.String(listenerARN),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if globalAcceleratorResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, endpointGroupSummary := range page.EndpointGroups {
			endpointGroupARN := StringValue(endpointGroupSummary.EndpointGroupArn)
			if endpointGroupARN == "" {
				continue
			}
			endpointGroup, err := getGlobalAcceleratorEndpointGroup(svc, endpointGroupARN)
			if globalAcceleratorResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newGlobalAcceleratorEndpointGroupResource(endpointGroup); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *GlobalAcceleratorGenerator) loadCustomRoutingAccelerators(svc *globalaccelerator.Client) error {
	p := globalaccelerator.NewListCustomRoutingAcceleratorsPaginator(svc, &globalaccelerator.ListCustomRoutingAcceleratorsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, acceleratorSummary := range page.Accelerators {
			acceleratorARN := StringValue(acceleratorSummary.AcceleratorArn)
			if acceleratorARN == "" {
				continue
			}
			accelerator, err := getGlobalAcceleratorCustomRoutingAccelerator(svc, acceleratorARN)
			if globalAcceleratorResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newGlobalAcceleratorCustomRoutingAcceleratorResource(accelerator); ok {
				g.Resources = append(g.Resources, resource)
			}
			if globalAcceleratorCustomRoutingAcceleratorImportable(accelerator) {
				if err := g.loadCustomRoutingListeners(svc, acceleratorARN); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (g *GlobalAcceleratorGenerator) loadCustomRoutingListeners(svc *globalaccelerator.Client, acceleratorARN string) error {
	if acceleratorARN == "" {
		return nil
	}
	p := globalaccelerator.NewListCustomRoutingListenersPaginator(svc, &globalaccelerator.ListCustomRoutingListenersInput{
		AcceleratorArn: aws.String(acceleratorARN),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if globalAcceleratorResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, listenerSummary := range page.Listeners {
			listenerARN := StringValue(listenerSummary.ListenerArn)
			if listenerARN == "" {
				continue
			}
			listener, err := getGlobalAcceleratorCustomRoutingListener(svc, listenerARN)
			if globalAcceleratorResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newGlobalAcceleratorCustomRoutingListenerResource(listener); ok {
				g.Resources = append(g.Resources, resource)
			}
			if globalAcceleratorCustomRoutingListenerImportable(listener) {
				if err := g.loadCustomRoutingEndpointGroups(svc, listenerARN); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (g *GlobalAcceleratorGenerator) loadCustomRoutingEndpointGroups(svc *globalaccelerator.Client, listenerARN string) error {
	if listenerARN == "" {
		return nil
	}
	p := globalaccelerator.NewListCustomRoutingEndpointGroupsPaginator(svc, &globalaccelerator.ListCustomRoutingEndpointGroupsInput{
		ListenerArn: aws.String(listenerARN),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if globalAcceleratorResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, endpointGroupSummary := range page.EndpointGroups {
			endpointGroupARN := StringValue(endpointGroupSummary.EndpointGroupArn)
			if endpointGroupARN == "" {
				continue
			}
			endpointGroup, err := getGlobalAcceleratorCustomRoutingEndpointGroup(svc, endpointGroupARN)
			if globalAcceleratorResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newGlobalAcceleratorCustomRoutingEndpointGroupResource(endpointGroup); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *GlobalAcceleratorGenerator) loadCrossAccountAttachments(svc *globalaccelerator.Client) error {
	p := globalaccelerator.NewListCrossAccountAttachmentsPaginator(svc, &globalaccelerator.ListCrossAccountAttachmentsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, attachmentSummary := range page.CrossAccountAttachments {
			attachmentARN := StringValue(attachmentSummary.AttachmentArn)
			if attachmentARN == "" {
				continue
			}
			attachment, err := getGlobalAcceleratorCrossAccountAttachment(svc, attachmentARN)
			if globalAcceleratorResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newGlobalAcceleratorCrossAccountAttachmentResource(attachment); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newGlobalAcceleratorAcceleratorResource(accelerator *globalacceleratortypes.Accelerator) (terraformutils.Resource, bool) {
	if !globalAcceleratorAcceleratorImportable(accelerator) {
		return terraformutils.Resource{}, false
	}
	acceleratorARN := StringValue(accelerator.AcceleratorArn)
	attributes := map[string]string{
		"name": StringValue(accelerator.Name),
	}
	globalAcceleratorPutBool(attributes, "enabled", accelerator.Enabled)
	globalAcceleratorPutString(attributes, "ip_address_type", string(accelerator.IpAddressType))
	return terraformutils.NewResource(
		globalAcceleratorImportID(acceleratorARN),
		globalAcceleratorResourceName("accelerator", StringValue(accelerator.Name), globalAcceleratorARNLastPart(acceleratorARN)),
		globalAcceleratorAcceleratorResourceType,
		"aws",
		attributes,
		globalAcceleratorAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newGlobalAcceleratorListenerResource(listener *globalacceleratortypes.Listener) (terraformutils.Resource, bool) {
	if !globalAcceleratorListenerImportable(listener) {
		return terraformutils.Resource{}, false
	}
	listenerARN := StringValue(listener.ListenerArn)
	acceleratorARN := globalAcceleratorAcceleratorARNFromChildARN(listenerARN)
	attributes := map[string]string{
		"accelerator_arn": acceleratorARN,
		"client_affinity": string(listener.ClientAffinity),
		"protocol":        string(listener.Protocol),
	}
	globalAcceleratorPutPortRanges(attributes, "port_range", listener.PortRanges)
	return terraformutils.NewResource(
		globalAcceleratorImportID(listenerARN),
		globalAcceleratorResourceName("listener", globalAcceleratorARNLastPart(acceleratorARN), globalAcceleratorARNLastPart(listenerARN)),
		globalAcceleratorListenerResourceType,
		"aws",
		attributes,
		globalAcceleratorAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newGlobalAcceleratorEndpointGroupResource(endpointGroup *globalacceleratortypes.EndpointGroup) (terraformutils.Resource, bool) {
	if !globalAcceleratorEndpointGroupImportable(endpointGroup) {
		return terraformutils.Resource{}, false
	}
	endpointGroupARN := StringValue(endpointGroup.EndpointGroupArn)
	listenerARN := globalAcceleratorListenerARNFromEndpointGroupARN(endpointGroupARN)
	attributes := map[string]string{
		"endpoint_group_region": StringValue(endpointGroup.EndpointGroupRegion),
		"listener_arn":          listenerARN,
	}
	globalAcceleratorPutInt32(attributes, "health_check_interval_seconds", endpointGroup.HealthCheckIntervalSeconds)
	globalAcceleratorPutString(attributes, "health_check_path", StringValue(endpointGroup.HealthCheckPath))
	globalAcceleratorPutInt32(attributes, "health_check_port", endpointGroup.HealthCheckPort)
	globalAcceleratorPutString(attributes, "health_check_protocol", string(endpointGroup.HealthCheckProtocol))
	globalAcceleratorPutFloat32(attributes, "traffic_dial_percentage", endpointGroup.TrafficDialPercentage)
	globalAcceleratorPutInt32(attributes, "threshold_count", endpointGroup.ThresholdCount)
	globalAcceleratorPutEndpointConfigurations(attributes, endpointGroup.EndpointDescriptions)
	globalAcceleratorPutPortOverrides(attributes, endpointGroup.PortOverrides)
	return terraformutils.NewResource(
		globalAcceleratorImportID(endpointGroupARN),
		globalAcceleratorResourceName("endpoint_group", globalAcceleratorARNLastPart(listenerARN), StringValue(endpointGroup.EndpointGroupRegion), globalAcceleratorARNLastPart(endpointGroupARN)),
		globalAcceleratorEndpointGroupResourceType,
		"aws",
		attributes,
		globalAcceleratorAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newGlobalAcceleratorCustomRoutingAcceleratorResource(accelerator *globalacceleratortypes.CustomRoutingAccelerator) (terraformutils.Resource, bool) {
	if !globalAcceleratorCustomRoutingAcceleratorImportable(accelerator) {
		return terraformutils.Resource{}, false
	}
	acceleratorARN := StringValue(accelerator.AcceleratorArn)
	attributes := map[string]string{
		"name": StringValue(accelerator.Name),
	}
	globalAcceleratorPutBool(attributes, "enabled", accelerator.Enabled)
	globalAcceleratorPutString(attributes, "ip_address_type", string(accelerator.IpAddressType))
	return terraformutils.NewResource(
		globalAcceleratorImportID(acceleratorARN),
		globalAcceleratorResourceName("custom_routing_accelerator", StringValue(accelerator.Name), globalAcceleratorARNLastPart(acceleratorARN)),
		globalAcceleratorCustomRoutingAcceleratorResourceType,
		"aws",
		attributes,
		globalAcceleratorAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newGlobalAcceleratorCustomRoutingListenerResource(listener *globalacceleratortypes.CustomRoutingListener) (terraformutils.Resource, bool) {
	if !globalAcceleratorCustomRoutingListenerImportable(listener) {
		return terraformutils.Resource{}, false
	}
	listenerARN := StringValue(listener.ListenerArn)
	acceleratorARN := globalAcceleratorAcceleratorARNFromChildARN(listenerARN)
	attributes := map[string]string{
		"accelerator_arn": acceleratorARN,
	}
	globalAcceleratorPutPortRanges(attributes, "port_range", listener.PortRanges)
	return terraformutils.NewResource(
		globalAcceleratorImportID(listenerARN),
		globalAcceleratorResourceName("custom_routing_listener", globalAcceleratorARNLastPart(acceleratorARN), globalAcceleratorARNLastPart(listenerARN)),
		globalAcceleratorCustomRoutingListenerResourceType,
		"aws",
		attributes,
		globalAcceleratorAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newGlobalAcceleratorCustomRoutingEndpointGroupResource(endpointGroup *globalacceleratortypes.CustomRoutingEndpointGroup) (terraformutils.Resource, bool) {
	if !globalAcceleratorCustomRoutingEndpointGroupImportable(endpointGroup) {
		return terraformutils.Resource{}, false
	}
	endpointGroupARN := StringValue(endpointGroup.EndpointGroupArn)
	listenerARN := globalAcceleratorListenerARNFromEndpointGroupARN(endpointGroupARN)
	attributes := map[string]string{
		"endpoint_group_region": StringValue(endpointGroup.EndpointGroupRegion),
		"listener_arn":          listenerARN,
	}
	globalAcceleratorPutCustomRoutingDestinationConfigurations(attributes, endpointGroup.DestinationDescriptions)
	globalAcceleratorPutCustomRoutingEndpointConfigurations(attributes, endpointGroup.EndpointDescriptions)
	return terraformutils.NewResource(
		globalAcceleratorImportID(endpointGroupARN),
		globalAcceleratorResourceName("custom_routing_endpoint_group", globalAcceleratorARNLastPart(listenerARN), StringValue(endpointGroup.EndpointGroupRegion), globalAcceleratorARNLastPart(endpointGroupARN)),
		globalAcceleratorCustomRoutingEndpointGroupResourceType,
		"aws",
		attributes,
		globalAcceleratorAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newGlobalAcceleratorCrossAccountAttachmentResource(attachment *globalacceleratortypes.Attachment) (terraformutils.Resource, bool) {
	if !globalAcceleratorCrossAccountAttachmentImportable(attachment) {
		return terraformutils.Resource{}, false
	}
	attachmentARN := StringValue(attachment.AttachmentArn)
	attributes := map[string]string{
		"name": StringValue(attachment.Name),
	}
	globalAcceleratorPutStringList(attributes, "principals", attachment.Principals)
	globalAcceleratorPutCrossAccountResources(attributes, attachment.Resources)
	return terraformutils.NewResource(
		globalAcceleratorImportID(attachmentARN),
		globalAcceleratorResourceName("cross_account_attachment", StringValue(attachment.Name), globalAcceleratorARNLastPart(attachmentARN)),
		globalAcceleratorCrossAccountAttachmentResourceType,
		"aws",
		attributes,
		globalAcceleratorAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func globalAcceleratorAcceleratorImportable(accelerator *globalacceleratortypes.Accelerator) bool {
	return accelerator != nil &&
		StringValue(accelerator.AcceleratorArn) != "" &&
		StringValue(accelerator.Name) != "" &&
		accelerator.Status == globalacceleratortypes.AcceleratorStatusDeployed
}

func globalAcceleratorListenerImportable(listener *globalacceleratortypes.Listener) bool {
	return listener != nil &&
		StringValue(listener.ListenerArn) != "" &&
		globalAcceleratorAcceleratorARNFromChildARN(StringValue(listener.ListenerArn)) != "" &&
		listener.Protocol != "" &&
		globalAcceleratorHasPortRange(listener.PortRanges)
}

func globalAcceleratorEndpointGroupImportable(endpointGroup *globalacceleratortypes.EndpointGroup) bool {
	return endpointGroup != nil &&
		StringValue(endpointGroup.EndpointGroupArn) != "" &&
		StringValue(endpointGroup.EndpointGroupRegion) != "" &&
		globalAcceleratorListenerARNFromEndpointGroupARN(StringValue(endpointGroup.EndpointGroupArn)) != ""
}

func globalAcceleratorCustomRoutingAcceleratorImportable(accelerator *globalacceleratortypes.CustomRoutingAccelerator) bool {
	return accelerator != nil &&
		StringValue(accelerator.AcceleratorArn) != "" &&
		StringValue(accelerator.Name) != "" &&
		accelerator.Status == globalacceleratortypes.CustomRoutingAcceleratorStatusDeployed
}

func globalAcceleratorCustomRoutingListenerImportable(listener *globalacceleratortypes.CustomRoutingListener) bool {
	return listener != nil &&
		StringValue(listener.ListenerArn) != "" &&
		globalAcceleratorAcceleratorARNFromChildARN(StringValue(listener.ListenerArn)) != "" &&
		globalAcceleratorHasPortRange(listener.PortRanges)
}

func globalAcceleratorCustomRoutingEndpointGroupImportable(endpointGroup *globalacceleratortypes.CustomRoutingEndpointGroup) bool {
	return endpointGroup != nil &&
		StringValue(endpointGroup.EndpointGroupArn) != "" &&
		StringValue(endpointGroup.EndpointGroupRegion) != "" &&
		globalAcceleratorListenerARNFromEndpointGroupARN(StringValue(endpointGroup.EndpointGroupArn)) != "" &&
		globalAcceleratorHasCustomRoutingDestination(endpointGroup.DestinationDescriptions)
}

func globalAcceleratorCrossAccountAttachmentImportable(attachment *globalacceleratortypes.Attachment) bool {
	return attachment != nil &&
		StringValue(attachment.AttachmentArn) != "" &&
		StringValue(attachment.Name) != ""
}

func globalAcceleratorHasPortRange(portRanges []globalacceleratortypes.PortRange) bool {
	for _, portRange := range portRanges {
		if portRange.FromPort != nil && portRange.ToPort != nil {
			return true
		}
	}
	return false
}

func globalAcceleratorHasCustomRoutingDestination(destinations []globalacceleratortypes.CustomRoutingDestinationDescription) bool {
	for _, destination := range destinations {
		if destination.FromPort != nil && destination.ToPort != nil && len(destination.Protocols) > 0 {
			return true
		}
	}
	return false
}

func globalAcceleratorImportID(id string) string {
	return id
}

func globalAcceleratorResourceName(parts ...string) string {
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

func globalAcceleratorARNLastPart(resourceARN string) string {
	if resourceARN == "" {
		return ""
	}
	parts := strings.Split(resourceARN, "/")
	return parts[len(parts)-1]
}

func globalAcceleratorAcceleratorARNFromChildARN(childARN string) string {
	return globalAcceleratorParentARN(childARN, globalAcceleratorListenerARNPart)
}

func globalAcceleratorListenerARNFromEndpointGroupARN(endpointGroupARN string) string {
	return globalAcceleratorParentARN(endpointGroupARN, globalAcceleratorEndpointGroupARNPart)
}

func globalAcceleratorParentARN(childARN, childPart string) string {
	if childARN == "" || childPart == "" {
		return ""
	}
	index := strings.Index(childARN, childPart)
	if index <= 0 {
		return ""
	}
	return childARN[:index]
}

func globalAcceleratorPutString(attributes map[string]string, key, value string) {
	if value != "" {
		attributes[key] = value
	}
}

func globalAcceleratorPutBool(attributes map[string]string, key string, value *bool) {
	if value != nil {
		attributes[key] = strconv.FormatBool(aws.ToBool(value))
	}
}

func globalAcceleratorPutInt32(attributes map[string]string, key string, value *int32) {
	if value != nil {
		attributes[key] = strconv.Itoa(int(aws.ToInt32(value)))
	}
}

func globalAcceleratorPutFloat32(attributes map[string]string, key string, value *float32) {
	if value != nil {
		attributes[key] = strconv.FormatFloat(float64(*value), 'f', -1, 32)
	}
}

func globalAcceleratorPutStringList(attributes map[string]string, key string, values []string) {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			filtered = append(filtered, value)
		}
	}
	attributes[key+".#"] = strconv.Itoa(len(filtered))
	for i, value := range filtered {
		attributes[key+"."+strconv.Itoa(i)] = value
	}
}

func globalAcceleratorPutPortRanges(attributes map[string]string, key string, portRanges []globalacceleratortypes.PortRange) {
	filtered := make([]globalacceleratortypes.PortRange, 0, len(portRanges))
	for _, portRange := range portRanges {
		if portRange.FromPort != nil && portRange.ToPort != nil {
			filtered = append(filtered, portRange)
		}
	}
	attributes[key+".#"] = strconv.Itoa(len(filtered))
	for i, portRange := range filtered {
		prefix := key + "." + strconv.Itoa(i)
		globalAcceleratorPutInt32(attributes, prefix+".from_port", portRange.FromPort)
		globalAcceleratorPutInt32(attributes, prefix+".to_port", portRange.ToPort)
	}
}

func globalAcceleratorPutEndpointConfigurations(attributes map[string]string, endpoints []globalacceleratortypes.EndpointDescription) {
	filtered := make([]globalacceleratortypes.EndpointDescription, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if StringValue(endpoint.EndpointId) != "" {
			filtered = append(filtered, endpoint)
		}
	}
	attributes["endpoint_configuration.#"] = strconv.Itoa(len(filtered))
	for i, endpoint := range filtered {
		prefix := "endpoint_configuration." + strconv.Itoa(i)
		globalAcceleratorPutString(attributes, prefix+".endpoint_id", StringValue(endpoint.EndpointId))
		globalAcceleratorPutBool(attributes, prefix+".client_ip_preservation_enabled", endpoint.ClientIPPreservationEnabled)
		globalAcceleratorPutInt32(attributes, prefix+".weight", endpoint.Weight)
	}
}

func globalAcceleratorPutPortOverrides(attributes map[string]string, portOverrides []globalacceleratortypes.PortOverride) {
	filtered := make([]globalacceleratortypes.PortOverride, 0, len(portOverrides))
	for _, portOverride := range portOverrides {
		if portOverride.EndpointPort != nil && portOverride.ListenerPort != nil {
			filtered = append(filtered, portOverride)
		}
	}
	attributes["port_override.#"] = strconv.Itoa(len(filtered))
	for i, portOverride := range filtered {
		prefix := "port_override." + strconv.Itoa(i)
		globalAcceleratorPutInt32(attributes, prefix+".endpoint_port", portOverride.EndpointPort)
		globalAcceleratorPutInt32(attributes, prefix+".listener_port", portOverride.ListenerPort)
	}
}

func globalAcceleratorPutCustomRoutingDestinationConfigurations(attributes map[string]string, destinations []globalacceleratortypes.CustomRoutingDestinationDescription) {
	filtered := make([]globalacceleratortypes.CustomRoutingDestinationDescription, 0, len(destinations))
	for _, destination := range destinations {
		if destination.FromPort != nil && destination.ToPort != nil && len(destination.Protocols) > 0 {
			filtered = append(filtered, destination)
		}
	}
	attributes["destination_configuration.#"] = strconv.Itoa(len(filtered))
	for i, destination := range filtered {
		prefix := "destination_configuration." + strconv.Itoa(i)
		globalAcceleratorPutInt32(attributes, prefix+".from_port", destination.FromPort)
		globalAcceleratorPutInt32(attributes, prefix+".to_port", destination.ToPort)
		attributes[prefix+".protocols.#"] = strconv.Itoa(len(destination.Protocols))
		for j, protocol := range destination.Protocols {
			attributes[prefix+".protocols."+strconv.Itoa(j)] = string(protocol)
		}
	}
}

func globalAcceleratorPutCustomRoutingEndpointConfigurations(attributes map[string]string, endpoints []globalacceleratortypes.CustomRoutingEndpointDescription) {
	filtered := make([]globalacceleratortypes.CustomRoutingEndpointDescription, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if StringValue(endpoint.EndpointId) != "" {
			filtered = append(filtered, endpoint)
		}
	}
	attributes["endpoint_configuration.#"] = strconv.Itoa(len(filtered))
	for i, endpoint := range filtered {
		globalAcceleratorPutString(attributes, "endpoint_configuration."+strconv.Itoa(i)+".endpoint_id", StringValue(endpoint.EndpointId))
	}
}

func globalAcceleratorPutCrossAccountResources(attributes map[string]string, resources []globalacceleratortypes.Resource) {
	filtered := make([]globalacceleratortypes.Resource, 0, len(resources))
	for _, resource := range resources {
		if StringValue(resource.EndpointId) != "" || StringValue(resource.Cidr) != "" {
			filtered = append(filtered, resource)
		}
	}
	attributes["resource.#"] = strconv.Itoa(len(filtered))
	for i, resource := range filtered {
		prefix := "resource." + strconv.Itoa(i)
		globalAcceleratorPutString(attributes, prefix+".endpoint_id", StringValue(resource.EndpointId))
		globalAcceleratorPutString(attributes, prefix+".region", StringValue(resource.Region))
		globalAcceleratorPutString(attributes, prefix+".cidr_block", StringValue(resource.Cidr))
	}
}

func getGlobalAcceleratorAccelerator(svc *globalaccelerator.Client, arn string) (*globalacceleratortypes.Accelerator, error) {
	output, err := svc.DescribeAccelerator(context.TODO(), &globalaccelerator.DescribeAcceleratorInput{
		AcceleratorArn: aws.String(arn),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.Accelerator, nil
}

func getGlobalAcceleratorListener(svc *globalaccelerator.Client, arn string) (*globalacceleratortypes.Listener, error) {
	output, err := svc.DescribeListener(context.TODO(), &globalaccelerator.DescribeListenerInput{
		ListenerArn: aws.String(arn),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.Listener, nil
}

func getGlobalAcceleratorEndpointGroup(svc *globalaccelerator.Client, arn string) (*globalacceleratortypes.EndpointGroup, error) {
	output, err := svc.DescribeEndpointGroup(context.TODO(), &globalaccelerator.DescribeEndpointGroupInput{
		EndpointGroupArn: aws.String(arn),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.EndpointGroup, nil
}

func getGlobalAcceleratorCustomRoutingAccelerator(svc *globalaccelerator.Client, arn string) (*globalacceleratortypes.CustomRoutingAccelerator, error) {
	output, err := svc.DescribeCustomRoutingAccelerator(context.TODO(), &globalaccelerator.DescribeCustomRoutingAcceleratorInput{
		AcceleratorArn: aws.String(arn),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.Accelerator, nil
}

func getGlobalAcceleratorCustomRoutingListener(svc *globalaccelerator.Client, arn string) (*globalacceleratortypes.CustomRoutingListener, error) {
	output, err := svc.DescribeCustomRoutingListener(context.TODO(), &globalaccelerator.DescribeCustomRoutingListenerInput{
		ListenerArn: aws.String(arn),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.Listener, nil
}

func getGlobalAcceleratorCustomRoutingEndpointGroup(svc *globalaccelerator.Client, arn string) (*globalacceleratortypes.CustomRoutingEndpointGroup, error) {
	output, err := svc.DescribeCustomRoutingEndpointGroup(context.TODO(), &globalaccelerator.DescribeCustomRoutingEndpointGroupInput{
		EndpointGroupArn: aws.String(arn),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.EndpointGroup, nil
}

func getGlobalAcceleratorCrossAccountAttachment(svc *globalaccelerator.Client, arn string) (*globalacceleratortypes.Attachment, error) {
	output, err := svc.DescribeCrossAccountAttachment(context.TODO(), &globalaccelerator.DescribeCrossAccountAttachmentInput{
		AttachmentArn: aws.String(arn),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.CrossAccountAttachment, nil
}

func globalAcceleratorResourceNotFound(err error) bool {
	if err == nil {
		return false
	}
	var acceleratorNotFound *globalacceleratortypes.AcceleratorNotFoundException
	if errors.As(err, &acceleratorNotFound) {
		return true
	}
	var listenerNotFound *globalacceleratortypes.ListenerNotFoundException
	if errors.As(err, &listenerNotFound) {
		return true
	}
	var endpointGroupNotFound *globalacceleratortypes.EndpointGroupNotFoundException
	if errors.As(err, &endpointGroupNotFound) {
		return true
	}
	var attachmentNotFound *globalacceleratortypes.AttachmentNotFoundException
	if errors.As(err, &attachmentNotFound) {
		return true
	}
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.ErrorCode() {
	case "AcceleratorNotFoundException",
		"ListenerNotFoundException",
		"EndpointGroupNotFoundException",
		"AttachmentNotFoundException":
		return true
	default:
		return false
	}
}
