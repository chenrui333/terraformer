// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	ec2PlacementGroupResourceType          = "aws_placement_group"
	ec2CapacityReservationResourceType     = "aws_ec2_capacity_reservation"
	ec2HostResourceType                    = "aws_ec2_host"
	ec2InstanceConnectEndpointResourceType = "aws_ec2_instance_connect_endpoint"
	ec2NetworkInsightsPathResourceType     = "aws_ec2_network_insights_path"
	ec2TrafficMirrorFilterResourceType     = "aws_ec2_traffic_mirror_filter"
	ec2TrafficMirrorFilterRuleResourceType = "aws_ec2_traffic_mirror_filter_rule"
	ec2TrafficMirrorSessionResourceType    = "aws_ec2_traffic_mirror_session"
	ec2TrafficMirrorTargetResourceType     = "aws_ec2_traffic_mirror_target"
	ec2TrafficMirrorFilterRuleIDSeparator  = ":"
)

var ec2CoreAllowEmptyValues = []string{"tags."}

var ec2CoreTagFilterResources = map[string]struct{}{
	"placement_group":               {},
	"ec2_instance_connect_endpoint": {},
}

type EC2CoreGenerator struct {
	AWSService
}

func (g *EC2CoreGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := ec2.NewFromConfig(config)

	loaders := []struct {
		serviceName string
		load        func(*ec2.Client) error
	}{
		{serviceName: "placement_group", load: g.loadPlacementGroups},
		{serviceName: "ec2_instance_connect_endpoint", load: g.loadInstanceConnectEndpoints},
		{serviceName: "ec2_capacity_reservation", load: g.loadCapacityReservations},
		{serviceName: "ec2_host", load: g.loadHosts},
		{serviceName: "ec2_network_insights_path", load: g.loadNetworkInsightsPaths},
		{serviceName: "ec2_traffic_mirror_filter", load: g.loadTrafficMirrorFilters},
		{serviceName: "ec2_traffic_mirror_filter_rule", load: g.loadTrafficMirrorFilterRules},
		{serviceName: "ec2_traffic_mirror_target", load: g.loadTrafficMirrorTargets},
		{serviceName: "ec2_traffic_mirror_session", load: g.loadTrafficMirrorSessions},
	}
	for _, loader := range loaders {
		if !g.shouldLoadEC2CoreResource(loader.serviceName) {
			continue
		}
		if err := loader.load(svc); err != nil {
			return err
		}
	}
	return nil
}

func (g *EC2CoreGenerator) loadPlacementGroups(svc *ec2.Client) error {
	output, err := svc.DescribePlacementGroups(context.TODO(), &ec2.DescribePlacementGroupsInput{
		Filters: g.ec2CoreTagFilters("placement_group"),
	})
	if err != nil {
		return err
	}
	for _, placementGroup := range output.PlacementGroups {
		if resource, ok := newEC2PlacementGroupResource(placementGroup); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *EC2CoreGenerator) loadInstanceConnectEndpoints(svc *ec2.Client) error {
	p := ec2.NewDescribeInstanceConnectEndpointsPaginator(svc, &ec2.DescribeInstanceConnectEndpointsInput{
		Filters: g.ec2CoreTagFilters("ec2_instance_connect_endpoint"),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, endpoint := range page.InstanceConnectEndpoints {
			if resource, ok := newEC2InstanceConnectEndpointResource(endpoint); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *EC2CoreGenerator) loadCapacityReservations(svc *ec2.Client) error {
	p := ec2.NewDescribeCapacityReservationsPaginator(svc, &ec2.DescribeCapacityReservationsInput{
		Filters: g.ec2CoreTagFilters("ec2_capacity_reservation"),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, reservation := range page.CapacityReservations {
			if resource, ok := newEC2CapacityReservationResource(reservation); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *EC2CoreGenerator) loadHosts(svc *ec2.Client) error {
	p := ec2.NewDescribeHostsPaginator(svc, &ec2.DescribeHostsInput{
		Filter: g.ec2CoreTagFilters("ec2_host"),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, host := range page.Hosts {
			if resource, ok := newEC2HostResource(host); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *EC2CoreGenerator) loadNetworkInsightsPaths(svc *ec2.Client) error {
	p := ec2.NewDescribeNetworkInsightsPathsPaginator(svc, &ec2.DescribeNetworkInsightsPathsInput{
		Filters: g.ec2CoreTagFilters("ec2_network_insights_path"),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, path := range page.NetworkInsightsPaths {
			if resource, ok := newEC2NetworkInsightsPathResource(path); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *EC2CoreGenerator) loadTrafficMirrorFilters(svc *ec2.Client) error {
	p := ec2.NewDescribeTrafficMirrorFiltersPaginator(svc, &ec2.DescribeTrafficMirrorFiltersInput{
		Filters: g.ec2CoreTagFilters("ec2_traffic_mirror_filter"),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, filter := range page.TrafficMirrorFilters {
			if resource, ok := newEC2TrafficMirrorFilterResource(filter); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *EC2CoreGenerator) loadTrafficMirrorFilterRules(svc *ec2.Client) error {
	input := &ec2.DescribeTrafficMirrorFilterRulesInput{
		MaxResults: aws.Int32(1000),
	}
	for {
		page, err := svc.DescribeTrafficMirrorFilterRules(context.TODO(), input)
		if err != nil {
			return err
		}
		for _, rule := range page.TrafficMirrorFilterRules {
			if resource, ok := newEC2TrafficMirrorFilterRuleResource(rule); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
		if page.NextToken == nil {
			break
		}
		input.NextToken = page.NextToken
	}
	return nil
}

func (g *EC2CoreGenerator) loadTrafficMirrorTargets(svc *ec2.Client) error {
	p := ec2.NewDescribeTrafficMirrorTargetsPaginator(svc, &ec2.DescribeTrafficMirrorTargetsInput{
		Filters: g.ec2CoreTagFilters("ec2_traffic_mirror_target"),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, target := range page.TrafficMirrorTargets {
			if resource, ok := newEC2TrafficMirrorTargetResource(target); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *EC2CoreGenerator) loadTrafficMirrorSessions(svc *ec2.Client) error {
	p := ec2.NewDescribeTrafficMirrorSessionsPaginator(svc, &ec2.DescribeTrafficMirrorSessionsInput{
		Filters: g.ec2CoreTagFilters("ec2_traffic_mirror_session"),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, session := range page.TrafficMirrorSessions {
			if resource, ok := newEC2TrafficMirrorSessionResource(session); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *EC2CoreGenerator) ec2CoreTagFilters(resourceName string) []types.Filter {
	if !ec2CoreSupportsTagFilters(resourceName) {
		return nil
	}
	var filters []types.Filter
	for _, filter := range g.Filter {
		if strings.HasPrefix(filter.FieldPath, "tags.") && filter.IsApplicable(resourceName) {
			filters = append(filters, types.Filter{
				Name:   aws.String("tag:" + strings.TrimPrefix(filter.FieldPath, "tags.")),
				Values: filter.AcceptableValues,
			})
		}
	}
	return filters
}

func ec2CoreSupportsTagFilters(resourceName string) bool {
	_, ok := ec2CoreTagFilterResources[resourceName]
	return ok
}

func (g *EC2CoreGenerator) shouldLoadEC2CoreResource(serviceNames ...string) bool {
	return shouldLoadAWSResourceForTypedFilters(g.Filter, serviceNames...)
}

func newEC2PlacementGroupResource(placementGroup types.PlacementGroup) (terraformutils.Resource, bool) {
	if !ec2PlacementGroupImportable(placementGroup) {
		return terraformutils.Resource{}, false
	}
	name := StringValue(placementGroup.GroupName)
	return terraformutils.NewSimpleResource(
		name,
		ec2CoreResourceName("placement_group", name),
		ec2PlacementGroupResourceType,
		"aws",
		ec2CoreAllowEmptyValues,
	), true
}

func newEC2InstanceConnectEndpointResource(endpoint types.Ec2InstanceConnectEndpoint) (terraformutils.Resource, bool) {
	if !ec2InstanceConnectEndpointImportable(endpoint) {
		return terraformutils.Resource{}, false
	}
	id := StringValue(endpoint.InstanceConnectEndpointId)
	return terraformutils.NewSimpleResource(
		id,
		ec2CoreResourceName("instance_connect_endpoint", StringValue(endpoint.SubnetId), id),
		ec2InstanceConnectEndpointResourceType,
		"aws",
		ec2CoreAllowEmptyValues,
	), true
}

func newEC2CapacityReservationResource(reservation types.CapacityReservation) (terraformutils.Resource, bool) {
	if !ec2CapacityReservationImportable(reservation) {
		return terraformutils.Resource{}, false
	}
	id := StringValue(reservation.CapacityReservationId)
	return terraformutils.NewSimpleResource(
		id,
		ec2CoreResourceName("capacity_reservation", StringValue(reservation.AvailabilityZone), id),
		ec2CapacityReservationResourceType,
		"aws",
		ec2CoreAllowEmptyValues,
	), true
}

func newEC2HostResource(host types.Host) (terraformutils.Resource, bool) {
	if !ec2HostImportable(host) {
		return terraformutils.Resource{}, false
	}
	id := StringValue(host.HostId)
	resource := terraformutils.NewSimpleResource(
		id,
		ec2CoreResourceName("host", StringValue(host.AvailabilityZone), id),
		ec2HostResourceType,
		"aws",
		ec2CoreAllowEmptyValues,
	)
	if host.HostProperties != nil && StringValue(host.HostProperties.InstanceType) != "" {
		resource.IgnoreKeys = append(resource.IgnoreKeys, "^instance_family$")
	}
	return resource, true
}

func newEC2NetworkInsightsPathResource(path types.NetworkInsightsPath) (terraformutils.Resource, bool) {
	id := StringValue(path.NetworkInsightsPathId)
	if id == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		id,
		ec2CoreResourceName("network_insights_path", StringValue(path.Source), StringValue(path.Destination), id),
		ec2NetworkInsightsPathResourceType,
		"aws",
		ec2CoreAllowEmptyValues,
	), true
}

func newEC2TrafficMirrorFilterResource(filter types.TrafficMirrorFilter) (terraformutils.Resource, bool) {
	id := StringValue(filter.TrafficMirrorFilterId)
	if id == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		id,
		ec2CoreResourceName("traffic_mirror_filter", id),
		ec2TrafficMirrorFilterResourceType,
		"aws",
		ec2CoreAllowEmptyValues,
	), true
}

func newEC2TrafficMirrorFilterRuleResource(rule types.TrafficMirrorFilterRule) (terraformutils.Resource, bool) {
	ruleID := StringValue(rule.TrafficMirrorFilterRuleId)
	filterID := StringValue(rule.TrafficMirrorFilterId)
	if ruleID == "" || filterID == "" {
		return terraformutils.Resource{}, false
	}
	resource := terraformutils.NewResource(
		ruleID,
		ec2CoreResourceName("traffic_mirror_filter_rule", filterID, ruleID),
		ec2TrafficMirrorFilterRuleResourceType,
		"aws",
		map[string]string{
			"traffic_mirror_filter_id": filterID,
		},
		ec2CoreAllowEmptyValues,
		map[string]interface{}{},
	)
	setEC2ImportID(&resource, ec2TrafficMirrorFilterRuleImportID(filterID, ruleID))
	return resource, true
}

func newEC2TrafficMirrorTargetResource(target types.TrafficMirrorTarget) (terraformutils.Resource, bool) {
	id := StringValue(target.TrafficMirrorTargetId)
	if id == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		id,
		ec2CoreResourceName("traffic_mirror_target", string(target.Type), id),
		ec2TrafficMirrorTargetResourceType,
		"aws",
		ec2CoreAllowEmptyValues,
	), true
}

func newEC2TrafficMirrorSessionResource(session types.TrafficMirrorSession) (terraformutils.Resource, bool) {
	id := StringValue(session.TrafficMirrorSessionId)
	if id == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		id,
		ec2CoreResourceName("traffic_mirror_session", StringValue(session.NetworkInterfaceId), id),
		ec2TrafficMirrorSessionResourceType,
		"aws",
		ec2CoreAllowEmptyValues,
	), true
}

func ec2PlacementGroupImportable(placementGroup types.PlacementGroup) bool {
	if StringValue(placementGroup.GroupName) == "" {
		return false
	}
	if placementGroup.State == types.PlacementGroupStateDeleting || placementGroup.State == types.PlacementGroupStateDeleted {
		return false
	}
	return placementGroup.Operator == nil || placementGroup.Operator.Managed == nil || !*placementGroup.Operator.Managed
}

func ec2InstanceConnectEndpointImportable(endpoint types.Ec2InstanceConnectEndpoint) bool {
	if StringValue(endpoint.InstanceConnectEndpointId) == "" {
		return false
	}
	switch endpoint.State {
	case types.Ec2InstanceConnectEndpointStateCreateComplete, types.Ec2InstanceConnectEndpointStateUpdateComplete:
		return true
	default:
		return false
	}
}

func ec2CapacityReservationImportable(reservation types.CapacityReservation) bool {
	if StringValue(reservation.CapacityReservationId) == "" {
		return false
	}
	if reservation.ReservationType == types.CapacityReservationTypeCapacityBlock {
		return false
	}
	switch reservation.State {
	case types.CapacityReservationStateActive, types.CapacityReservationStateScheduled:
		return true
	default:
		return false
	}
}

func ec2HostImportable(host types.Host) bool {
	if StringValue(host.HostId) == "" {
		return false
	}
	if host.State != types.AllocationStateAvailable {
		return false
	}
	if host.HostProperties == nil {
		return false
	}
	return StringValue(host.HostProperties.InstanceFamily) != "" || StringValue(host.HostProperties.InstanceType) != ""
}

func ec2TrafficMirrorFilterRuleImportID(filterID, ruleID string) string {
	if filterID == "" || ruleID == "" {
		return ""
	}
	return filterID + ec2TrafficMirrorFilterRuleIDSeparator + ruleID
}

func setEC2ImportID(resource *terraformutils.Resource, importID string) {
	if resource == nil || resource.InstanceState == nil || importID == "" {
		return
	}
	if resource.InstanceState.Meta == nil {
		resource.InstanceState.Meta = map[string]interface{}{}
	}
	resource.InstanceState.Meta["import_id"] = importID
}

func ec2CoreResourceName(parts ...string) string {
	nameParts := make([]string, 0, len(parts)*2)
	for _, part := range parts {
		if part == "" {
			continue
		}
		nameParts = append(nameParts, strconv.Itoa(len(part)), part)
	}
	return strings.Join(nameParts, "_")
}
