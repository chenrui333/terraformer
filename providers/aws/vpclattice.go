// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/vpclattice"
	vpclatticetypes "github.com/aws/aws-sdk-go-v2/service/vpclattice/types"
	"github.com/aws/smithy-go"
)

var vpclatticeAllowEmptyValues = []string{"tags."}

const (
	vpclatticeAccessLogSubscriptionResourceType            = "aws_vpclattice_access_log_subscription"
	vpclatticeAuthPolicyResourceType                       = "aws_vpclattice_auth_policy"
	vpclatticeListenerResourceType                         = "aws_vpclattice_listener"
	vpclatticeListenerRuleResourceType                     = "aws_vpclattice_listener_rule"
	vpclatticeResourcePolicyResourceType                   = "aws_vpclattice_resource_policy"
	vpclatticeServiceResourceType                          = "aws_vpclattice_service"
	vpclatticeServiceNetworkResourceType                   = "aws_vpclattice_service_network"
	vpclatticeServiceNetworkServiceAssociationResourceType = "aws_vpclattice_service_network_service_association"
	vpclatticeServiceNetworkVpcAssociationResourceType     = "aws_vpclattice_service_network_vpc_association"
	vpclatticeTargetGroupResourceType                      = "aws_vpclattice_target_group"
)

type VPCLatticeGenerator struct {
	AWSService
}

func (g *VPCLatticeGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := vpclattice.NewFromConfig(config)

	loadServiceNetworks := g.shouldLoadVPCLatticeResource(vpclatticeServiceNetworkResourceType)
	loadServices := g.shouldLoadVPCLatticeResource(vpclatticeServiceResourceType)
	loadTargetGroups := g.shouldLoadVPCLatticeResource(vpclatticeTargetGroupResourceType)
	loadListeners := g.shouldLoadVPCLatticeResource(vpclatticeListenerResourceType)
	loadListenerRules := g.shouldLoadVPCLatticeResource(vpclatticeListenerRuleResourceType)
	loadServiceNetworkServiceAssociations := g.shouldLoadVPCLatticeResource(vpclatticeServiceNetworkServiceAssociationResourceType)
	loadServiceNetworkVpcAssociations := g.shouldLoadVPCLatticeResource(vpclatticeServiceNetworkVpcAssociationResourceType)
	loadAuthPolicies := g.shouldLoadVPCLatticeResource(vpclatticeAuthPolicyResourceType)
	loadResourcePolicies := g.shouldLoadVPCLatticeResource(vpclatticeResourcePolicyResourceType)
	loadAccessLogSubscriptions := g.shouldLoadVPCLatticeResource(vpclatticeAccessLogSubscriptionResourceType)

	needOwnedResourceFilter := loadServiceNetworks || loadServices || loadListeners || loadListenerRules || loadServiceNetworkServiceAssociations || loadServiceNetworkVpcAssociations || loadAuthPolicies || loadResourcePolicies || loadAccessLogSubscriptions
	callerAccountID := ""
	if needOwnedResourceFilter {
		accountID, err := g.getAccountNumber(config)
		if err != nil {
			return err
		}
		callerAccountID = StringValue(accountID)
	}

	needServiceNetworks := loadServiceNetworks || loadServiceNetworkServiceAssociations || loadServiceNetworkVpcAssociations || loadAuthPolicies || loadResourcePolicies || loadAccessLogSubscriptions
	if needServiceNetworks {
		serviceNetworks, err := g.listVPCLatticeServiceNetworks(svc)
		if err != nil {
			return err
		}
		for _, serviceNetwork := range serviceNetworks {
			resourceID := StringValue(serviceNetwork.Id)
			resourceARN := StringValue(serviceNetwork.Arn)
			resourceOwnedByAccount := vpclatticeResourceOwnedByAccount(resourceARN, callerAccountID)
			if loadServiceNetworks && resourceOwnedByAccount {
				if resource, ok := newVPCLatticeServiceNetworkResource(serviceNetwork); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
			if loadAuthPolicies && resourceOwnedByAccount {
				if err := g.loadVPCLatticeAuthPolicy(svc, resourceARN); err != nil {
					return err
				}
			}
			if loadResourcePolicies && resourceOwnedByAccount {
				if err := g.loadVPCLatticeResourcePolicy(svc, resourceARN); err != nil {
					return err
				}
			}
			if loadAccessLogSubscriptions && resourceOwnedByAccount {
				if err := g.loadVPCLatticeAccessLogSubscriptions(svc, resourceID); err != nil {
					return err
				}
			}
			if loadServiceNetworkServiceAssociations {
				if err := g.loadVPCLatticeServiceNetworkServiceAssociations(svc, resourceID, callerAccountID); err != nil {
					return err
				}
			}
			if loadServiceNetworkVpcAssociations {
				if err := g.loadVPCLatticeServiceNetworkVpcAssociations(svc, resourceID, callerAccountID); err != nil {
					return err
				}
			}
		}
	}

	needServices := loadServices || loadListeners || loadListenerRules || loadAuthPolicies || loadResourcePolicies || loadAccessLogSubscriptions
	if needServices {
		services, err := g.listVPCLatticeServices(svc)
		if err != nil {
			return err
		}
		for _, service := range services {
			if !vpclatticeServiceStatusImportable(service.Status) {
				continue
			}
			serviceID := StringValue(service.Id)
			serviceARN := StringValue(service.Arn)
			if !vpclatticeResourceOwnedByAccount(serviceARN, callerAccountID) {
				continue
			}
			if loadServices {
				if resource, ok := newVPCLatticeServiceResource(service); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
			if loadAuthPolicies {
				if err := g.loadVPCLatticeAuthPolicy(svc, serviceARN); err != nil {
					return err
				}
			}
			if loadResourcePolicies {
				if err := g.loadVPCLatticeResourcePolicy(svc, serviceARN); err != nil {
					return err
				}
			}
			if loadAccessLogSubscriptions {
				if err := g.loadVPCLatticeAccessLogSubscriptions(svc, serviceID); err != nil {
					return err
				}
			}
			if loadListeners || loadListenerRules {
				if err := g.loadVPCLatticeListeners(svc, serviceID, loadListeners, loadListenerRules); err != nil {
					return err
				}
			}
		}
	}

	if loadTargetGroups {
		if err := g.loadVPCLatticeTargetGroups(svc); err != nil {
			return err
		}
	}
	return nil
}

func (g *VPCLatticeGenerator) shouldLoadVPCLatticeResource(serviceNames ...string) bool {
	return shouldLoadAWSResourceForTypedFilters(g.Filter, serviceNames...)
}

func (g *VPCLatticeGenerator) listVPCLatticeServiceNetworks(svc *vpclattice.Client) ([]vpclatticetypes.ServiceNetworkSummary, error) {
	var serviceNetworks []vpclatticetypes.ServiceNetworkSummary
	input := &vpclattice.ListServiceNetworksInput{}
	for {
		output, err := svc.ListServiceNetworks(context.TODO(), input)
		if err != nil {
			return nil, err
		}
		serviceNetworks = append(serviceNetworks, output.Items...)
		if !awsHasMorePages(output.NextToken) {
			break
		}
		input.NextToken = output.NextToken
	}
	return serviceNetworks, nil
}

func (g *VPCLatticeGenerator) listVPCLatticeServices(svc *vpclattice.Client) ([]vpclatticetypes.ServiceSummary, error) {
	var services []vpclatticetypes.ServiceSummary
	input := &vpclattice.ListServicesInput{}
	for {
		output, err := svc.ListServices(context.TODO(), input)
		if err != nil {
			return nil, err
		}
		services = append(services, output.Items...)
		if !awsHasMorePages(output.NextToken) {
			break
		}
		input.NextToken = output.NextToken
	}
	return services, nil
}

func (g *VPCLatticeGenerator) loadVPCLatticeTargetGroups(svc *vpclattice.Client) error {
	input := &vpclattice.ListTargetGroupsInput{}
	for {
		output, err := svc.ListTargetGroups(context.TODO(), input)
		if err != nil {
			return err
		}
		for _, targetGroup := range output.Items {
			if resource, ok := newVPCLatticeTargetGroupResource(targetGroup); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
		if !awsHasMorePages(output.NextToken) {
			break
		}
		input.NextToken = output.NextToken
	}
	return nil
}

func (g *VPCLatticeGenerator) loadVPCLatticeListeners(svc *vpclattice.Client, serviceID string, loadListeners, loadListenerRules bool) error {
	if serviceID == "" {
		return nil
	}
	input := &vpclattice.ListListenersInput{
		ServiceIdentifier: &serviceID,
	}
	for {
		output, err := svc.ListListeners(context.TODO(), input)
		if err != nil {
			return err
		}
		for _, listener := range output.Items {
			listenerID := StringValue(listener.Id)
			if loadListeners {
				if resource, ok := newVPCLatticeListenerResource(serviceID, listener); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
			if loadListenerRules {
				if err := g.loadVPCLatticeListenerRules(svc, serviceID, listenerID); err != nil {
					return err
				}
			}
		}
		if !awsHasMorePages(output.NextToken) {
			break
		}
		input.NextToken = output.NextToken
	}
	return nil
}

func (g *VPCLatticeGenerator) loadVPCLatticeListenerRules(svc *vpclattice.Client, serviceID, listenerID string) error {
	if serviceID == "" || listenerID == "" {
		return nil
	}
	input := &vpclattice.ListRulesInput{
		ServiceIdentifier:  &serviceID,
		ListenerIdentifier: &listenerID,
	}
	for {
		output, err := svc.ListRules(context.TODO(), input)
		if err != nil {
			return err
		}
		for _, rule := range output.Items {
			if resource, ok := newVPCLatticeListenerRuleResource(serviceID, listenerID, rule); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
		if !awsHasMorePages(output.NextToken) {
			break
		}
		input.NextToken = output.NextToken
	}
	return nil
}

func (g *VPCLatticeGenerator) loadVPCLatticeServiceNetworkServiceAssociations(svc *vpclattice.Client, serviceNetworkID, accountID string) error {
	if serviceNetworkID == "" {
		return nil
	}
	input := &vpclattice.ListServiceNetworkServiceAssociationsInput{
		ServiceNetworkIdentifier: &serviceNetworkID,
	}
	for {
		output, err := svc.ListServiceNetworkServiceAssociations(context.TODO(), input)
		if err != nil {
			return err
		}
		for _, association := range output.Items {
			if resource, ok := newVPCLatticeServiceNetworkServiceAssociationResource(association, accountID); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
		if !awsHasMorePages(output.NextToken) {
			break
		}
		input.NextToken = output.NextToken
	}
	return nil
}

func (g *VPCLatticeGenerator) loadVPCLatticeServiceNetworkVpcAssociations(svc *vpclattice.Client, serviceNetworkID, accountID string) error {
	if serviceNetworkID == "" {
		return nil
	}
	input := &vpclattice.ListServiceNetworkVpcAssociationsInput{
		ServiceNetworkIdentifier: &serviceNetworkID,
	}
	for {
		output, err := svc.ListServiceNetworkVpcAssociations(context.TODO(), input)
		if err != nil {
			return err
		}
		for _, association := range output.Items {
			if resource, ok := newVPCLatticeServiceNetworkVpcAssociationResource(association, accountID); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
		if !awsHasMorePages(output.NextToken) {
			break
		}
		input.NextToken = output.NextToken
	}
	return nil
}

func (g *VPCLatticeGenerator) loadVPCLatticeAuthPolicy(svc *vpclattice.Client, resourceIdentifier string) error {
	if resourceIdentifier == "" {
		return nil
	}
	output, err := svc.GetAuthPolicy(context.TODO(), &vpclattice.GetAuthPolicyInput{
		ResourceIdentifier: &resourceIdentifier,
	})
	if err != nil {
		if vpclatticeOptionalResourceUnavailable(err) {
			return nil
		}
		return err
	}
	if resource, ok := newVPCLatticeAuthPolicyResource(resourceIdentifier, StringValue(output.Policy)); ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *VPCLatticeGenerator) loadVPCLatticeResourcePolicy(svc *vpclattice.Client, resourceARN string) error {
	if resourceARN == "" {
		return nil
	}
	output, err := svc.GetResourcePolicy(context.TODO(), &vpclattice.GetResourcePolicyInput{
		ResourceArn: &resourceARN,
	})
	if err != nil {
		if vpclatticeOptionalResourceUnavailable(err) {
			return nil
		}
		return err
	}
	if resource, ok := newVPCLatticeResourcePolicyResource(resourceARN, StringValue(output.Policy)); ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *VPCLatticeGenerator) loadVPCLatticeAccessLogSubscriptions(svc *vpclattice.Client, resourceIdentifier string) error {
	if resourceIdentifier == "" {
		return nil
	}
	input := &vpclattice.ListAccessLogSubscriptionsInput{
		ResourceIdentifier: &resourceIdentifier,
	}
	for {
		output, err := svc.ListAccessLogSubscriptions(context.TODO(), input)
		if err != nil {
			if vpclatticeOptionalResourceUnavailable(err) {
				return nil
			}
			return err
		}
		for _, subscription := range output.Items {
			if resource, ok := newVPCLatticeAccessLogSubscriptionResource(subscription); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
		if !awsHasMorePages(output.NextToken) {
			break
		}
		input.NextToken = output.NextToken
	}
	return nil
}

func newVPCLatticeServiceNetworkResource(serviceNetwork vpclatticetypes.ServiceNetworkSummary) (terraformutils.Resource, bool) {
	id := StringValue(serviceNetwork.Id)
	if id == "" {
		return terraformutils.Resource{}, false
	}
	name := StringValue(serviceNetwork.Name)
	return terraformutils.NewSimpleResource(
		id,
		vpclatticeResourceName("service_network", id, name),
		vpclatticeServiceNetworkResourceType,
		"aws",
		vpclatticeAllowEmptyValues), true
}

func newVPCLatticeServiceResource(service vpclatticetypes.ServiceSummary) (terraformutils.Resource, bool) {
	id := StringValue(service.Id)
	if id == "" || !vpclatticeServiceStatusImportable(service.Status) {
		return terraformutils.Resource{}, false
	}
	name := StringValue(service.Name)
	return terraformutils.NewSimpleResource(
		id,
		vpclatticeResourceName("service", id, name),
		vpclatticeServiceResourceType,
		"aws",
		vpclatticeAllowEmptyValues), true
}

func vpclatticeServiceStatusImportable(status vpclatticetypes.ServiceStatus) bool {
	return status == vpclatticetypes.ServiceStatusActive
}

func newVPCLatticeTargetGroupResource(targetGroup vpclatticetypes.TargetGroupSummary) (terraformutils.Resource, bool) {
	id := StringValue(targetGroup.Id)
	if id == "" || !vpclatticeTargetGroupStatusImportable(targetGroup.Status) {
		return terraformutils.Resource{}, false
	}
	name := StringValue(targetGroup.Name)
	return terraformutils.NewSimpleResource(
		id,
		vpclatticeResourceName("target_group", id, name),
		vpclatticeTargetGroupResourceType,
		"aws",
		vpclatticeAllowEmptyValues), true
}

func vpclatticeTargetGroupStatusImportable(status vpclatticetypes.TargetGroupStatus) bool {
	return status == vpclatticetypes.TargetGroupStatusActive
}

func vpclatticeResourceOwnedByAccount(resourceARN, accountID string) bool {
	if resourceARN == "" || accountID == "" {
		return false
	}
	parsedARN, err := arn.Parse(resourceARN)
	if err != nil {
		return false
	}
	return parsedARN.AccountID == accountID
}

func vpclatticeServiceNetworkAssociationIdentifier(serviceNetworkID, serviceNetworkARN, associatedResourceARN, accountID string) (string, map[string]interface{}) {
	additionalFields := map[string]interface{}{}
	if serviceNetworkID == "" {
		return "", additionalFields
	}
	if serviceNetworkARN == "" || accountID == "" {
		return serviceNetworkID, additionalFields
	}
	parsedARN, err := arn.Parse(serviceNetworkARN)
	if err != nil || parsedARN.AccountID == "" {
		return serviceNetworkID, additionalFields
	}

	useARN := parsedARN.AccountID != accountID
	if associatedResourceARN != "" {
		associatedARN, err := arn.Parse(associatedResourceARN)
		if err == nil && associatedARN.AccountID != "" && associatedARN.AccountID != parsedARN.AccountID {
			useARN = true
		}
	}
	if !useARN {
		return serviceNetworkID, additionalFields
	}

	additionalFields["service_network_identifier"] = serviceNetworkARN
	return serviceNetworkARN, additionalFields
}

func newVPCLatticeListenerResource(serviceID string, listener vpclatticetypes.ListenerSummary) (terraformutils.Resource, bool) {
	listenerID := StringValue(listener.Id)
	if serviceID == "" || listenerID == "" {
		return terraformutils.Resource{}, false
	}
	importID := vpclatticeListenerImportID(serviceID, listenerID)
	return terraformutils.NewResource(
		importID,
		vpclatticeResourceName("listener", serviceID, listenerID, StringValue(listener.Name)),
		vpclatticeListenerResourceType,
		"aws",
		map[string]string{
			"service_identifier": serviceID,
		},
		vpclatticeAllowEmptyValues,
		map[string]interface{}{}), true
}

func vpclatticeListenerImportID(serviceID, listenerID string) string {
	return fmt.Sprintf("%s/%s", serviceID, listenerID)
}

func newVPCLatticeListenerRuleResource(serviceID, listenerID string, rule vpclatticetypes.RuleSummary) (terraformutils.Resource, bool) {
	ruleID := StringValue(rule.Id)
	if serviceID == "" || listenerID == "" || ruleID == "" || boolValue(rule.IsDefault) {
		return terraformutils.Resource{}, false
	}
	importID := vpclatticeListenerRuleImportID(serviceID, listenerID, ruleID)
	return terraformutils.NewResource(
		importID,
		vpclatticeResourceName("listener_rule", serviceID, listenerID, ruleID, StringValue(rule.Name)),
		vpclatticeListenerRuleResourceType,
		"aws",
		map[string]string{
			"service_identifier":  serviceID,
			"listener_identifier": listenerID,
		},
		vpclatticeAllowEmptyValues,
		map[string]interface{}{}), true
}

func vpclatticeListenerRuleImportID(serviceID, listenerID, ruleID string) string {
	return fmt.Sprintf("%s/%s/%s", serviceID, listenerID, ruleID)
}

func newVPCLatticeServiceNetworkServiceAssociationResource(association vpclatticetypes.ServiceNetworkServiceAssociationSummary, accountID string) (terraformutils.Resource, bool) {
	id := StringValue(association.Id)
	serviceID := StringValue(association.ServiceId)
	serviceNetworkID := StringValue(association.ServiceNetworkId)
	serviceNetworkIdentifier, additionalFields := vpclatticeServiceNetworkAssociationIdentifier(serviceNetworkID, StringValue(association.ServiceNetworkArn), StringValue(association.ServiceArn), accountID)
	if id == "" || serviceID == "" || serviceNetworkID == "" || accountID == "" || StringValue(association.CreatedBy) != accountID || !vpclatticeServiceNetworkServiceAssociationStatusImportable(association.Status) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		id,
		vpclatticeResourceName("service_network_service_association", id, serviceNetworkID, serviceID),
		vpclatticeServiceNetworkServiceAssociationResourceType,
		"aws",
		map[string]string{
			"service_identifier":         serviceID,
			"service_network_identifier": serviceNetworkIdentifier,
		},
		vpclatticeAllowEmptyValues,
		additionalFields), true
}

func vpclatticeServiceNetworkServiceAssociationStatusImportable(status vpclatticetypes.ServiceNetworkServiceAssociationStatus) bool {
	return status == vpclatticetypes.ServiceNetworkServiceAssociationStatusActive
}

func newVPCLatticeServiceNetworkVpcAssociationResource(association vpclatticetypes.ServiceNetworkVpcAssociationSummary, accountID string) (terraformutils.Resource, bool) {
	id := StringValue(association.Id)
	serviceNetworkID := StringValue(association.ServiceNetworkId)
	serviceNetworkIdentifier, additionalFields := vpclatticeServiceNetworkAssociationIdentifier(serviceNetworkID, StringValue(association.ServiceNetworkArn), "", accountID)
	vpcID := StringValue(association.VpcId)
	if id == "" || serviceNetworkID == "" || vpcID == "" || accountID == "" || StringValue(association.CreatedBy) != accountID || !vpclatticeServiceNetworkVpcAssociationStatusImportable(association.Status) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		id,
		vpclatticeResourceName("service_network_vpc_association", id, serviceNetworkID, vpcID),
		vpclatticeServiceNetworkVpcAssociationResourceType,
		"aws",
		map[string]string{
			"service_network_identifier": serviceNetworkIdentifier,
			"vpc_identifier":             vpcID,
		},
		vpclatticeAllowEmptyValues,
		additionalFields), true
}

func vpclatticeServiceNetworkVpcAssociationStatusImportable(status vpclatticetypes.ServiceNetworkVpcAssociationStatus) bool {
	return status == vpclatticetypes.ServiceNetworkVpcAssociationStatusActive
}

func newVPCLatticeAuthPolicyResource(resourceIdentifier, policy string) (terraformutils.Resource, bool) {
	if resourceIdentifier == "" || policy == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		resourceIdentifier,
		vpclatticeResourceName("auth_policy", resourceIdentifier),
		vpclatticeAuthPolicyResourceType,
		"aws",
		map[string]string{
			"resource_identifier": resourceIdentifier,
			"policy":              policy,
		},
		vpclatticeAllowEmptyValues,
		map[string]interface{}{}), true
}

func newVPCLatticeResourcePolicyResource(resourceARN, policy string) (terraformutils.Resource, bool) {
	if resourceARN == "" || policy == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		resourceARN,
		vpclatticeResourceName("resource_policy", resourceARN),
		vpclatticeResourcePolicyResourceType,
		"aws",
		map[string]string{
			"resource_arn": resourceARN,
			"policy":       policy,
		},
		vpclatticeAllowEmptyValues,
		map[string]interface{}{}), true
}

func newVPCLatticeAccessLogSubscriptionResource(subscription vpclatticetypes.AccessLogSubscriptionSummary) (terraformutils.Resource, bool) {
	id := StringValue(subscription.Id)
	resourceIdentifier := StringValue(subscription.ResourceId)
	if resourceIdentifier == "" {
		resourceIdentifier = StringValue(subscription.ResourceArn)
	}
	destinationARN := StringValue(subscription.DestinationArn)
	if id == "" || resourceIdentifier == "" || destinationARN == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		id,
		vpclatticeResourceName("access_log_subscription", id, resourceIdentifier, destinationARN),
		vpclatticeAccessLogSubscriptionResourceType,
		"aws",
		map[string]string{
			"resource_identifier": resourceIdentifier,
			"destination_arn":     destinationARN,
		},
		vpclatticeAllowEmptyValues,
		map[string]interface{}{}), true
}

func vpclatticeResourceName(parts ...string) string {
	return awsResourceNameWithLengths(parts...)
}

func awsResourceNameWithLengths(parts ...string) string {
	nameParts := []string{}
	for _, part := range parts {
		if part == "" {
			continue
		}
		nameParts = append(nameParts, fmt.Sprintf("%d-%s", len(part), part))
	}
	return strings.Join(nameParts, "-")
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func vpclatticeOptionalResourceUnavailable(err error) bool {
	var resourceNotFound *vpclatticetypes.ResourceNotFoundException
	if errors.As(err, &resourceNotFound) {
		return true
	}
	var accessDenied *vpclatticetypes.AccessDeniedException
	if errors.As(err, &accessDenied) {
		return true
	}
	var validation *vpclatticetypes.ValidationException
	if errors.As(err, &validation) {
		return true
	}

	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.ErrorCode() {
	case "ResourceNotFoundException", "AccessDeniedException", "ValidationException":
		return true
	default:
		return false
	}
}

func (g *VPCLatticeGenerator) PostConvertHook() error {
	for i, resource := range g.Resources {
		switch resource.InstanceInfo.Type {
		case vpclatticeAuthPolicyResourceType, vpclatticeResourcePolicyResourceType:
			if val, ok := g.Resources[i].Item["policy"]; ok {
				policy := g.escapeAwsInterpolation(val.(string))
				g.Resources[i].Item["policy"] = fmt.Sprintf("<<POLICY\n%s\nPOLICY", policy)
			}
		}
	}
	return nil
}
