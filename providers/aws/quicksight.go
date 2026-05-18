// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/quicksight"
	quicksighttypes "github.com/aws/aws-sdk-go-v2/service/quicksight/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	quickSightFolderResourceType           = "aws_quicksight_folder"
	quickSightFolderMembershipResourceType = "aws_quicksight_folder_membership"
	quickSightGroupResourceType            = "aws_quicksight_group"
	quickSightGroupMembershipResourceType  = "aws_quicksight_group_membership"
	quickSightNamespaceResourceType        = "aws_quicksight_namespace"
	quickSightVPCConnectionResourceType    = "aws_quicksight_vpc_connection"
	quickSightCommaSeparator               = ","
	quickSightSlashSeparator               = "/"
	quickSightResourceNameFallback         = "quicksight-resource"
)

var (
	quickSightAllowEmptyValues = []string{"tags."}
	quickSightResourceTypes    = []string{
		quickSightServiceName(quickSightFolderResourceType),
		quickSightServiceName(quickSightFolderMembershipResourceType),
		quickSightServiceName(quickSightGroupResourceType),
		quickSightServiceName(quickSightGroupMembershipResourceType),
		quickSightServiceName(quickSightNamespaceResourceType),
		quickSightServiceName(quickSightVPCConnectionResourceType),
	}
)

type QuickSightGenerator struct {
	AWSService
}

func (g *QuickSightGenerator) InitialCleanup() {
	if len(g.Filter) == 0 {
		return
	}
	filteredResources := []terraformutils.Resource{}
	for _, resource := range g.Resources {
		serviceName := quickSightServiceName(resource.InstanceInfo.Type)
		if g.hasTypedQuickSightFilter() && !g.hasTypedFilterFor(serviceName) && !g.hasUntypedFilter() {
			continue
		}
		allPredicatesTrue := true
		for _, filter := range g.Filter {
			if filter.FieldPath != "id" {
				continue
			}
			if filter.ServiceName != "" && filter.ServiceName != serviceName {
				continue
			}
			allPredicatesTrue = allPredicatesTrue && filter.Filter(resource)
		}
		if allPredicatesTrue && !terraformutils.ContainsResource(filteredResources, resource) {
			filteredResources = append(filteredResources, resource)
		}
	}
	g.Resources = filteredResources
}

func (g *QuickSightGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	accountID, err := g.getAccountNumber(config)
	if err != nil {
		return err
	}
	if StringValue(accountID) == "" {
		return nil
	}

	svc := quicksight.NewFromConfig(config)
	loadNamespaces := g.shouldLoadQuickSightResource(quickSightServiceName(quickSightNamespaceResourceType))
	loadGroups := g.shouldLoadQuickSightResource(quickSightServiceName(quickSightGroupResourceType))
	loadGroupMemberships := g.shouldLoadQuickSightResource(quickSightServiceName(quickSightGroupMembershipResourceType))
	loadFolders := g.shouldLoadQuickSightResource(quickSightServiceName(quickSightFolderResourceType))
	loadFolderMemberships := g.shouldLoadQuickSightResource(quickSightServiceName(quickSightFolderMembershipResourceType))
	loadVPCConnections := g.shouldLoadQuickSightResource(quickSightServiceName(quickSightVPCConnectionResourceType))

	if loadNamespaces || loadGroups || loadGroupMemberships {
		namespaces, err := listQuickSightNamespaces(svc, StringValue(accountID))
		if err != nil {
			if quickSightServiceUnavailable(err) {
				log.Printf("[WARN] skipping QuickSight discovery: %s", err)
			} else {
				return err
			}
		} else {
			if loadNamespaces {
				g.loadNamespaces(StringValue(accountID), namespaces)
			}
			if loadGroups || loadGroupMemberships {
				if err := g.loadGroupsAndMemberships(svc, StringValue(accountID), namespaces, loadGroups, loadGroupMemberships); err != nil {
					return err
				}
			}
		}
	}

	if loadFolders || loadFolderMemberships {
		folders, err := listQuickSightFolders(svc, StringValue(accountID))
		if err != nil {
			if quickSightServiceUnavailable(err) {
				log.Printf("[WARN] skipping QuickSight folders: %s", err)
			} else {
				return err
			}
		} else {
			if loadFolders {
				g.loadFolders(StringValue(accountID), folders)
			}
			if loadFolderMemberships {
				if err := g.loadFolderMemberships(svc, StringValue(accountID), folders); err != nil {
					return err
				}
			}
		}
	}

	if loadVPCConnections {
		if err := g.loadVPCConnections(svc, StringValue(accountID)); err != nil {
			if quickSightServiceUnavailable(err) {
				log.Printf("[WARN] skipping QuickSight VPC connections: %s", err)
				return nil
			}
			return err
		}
	}

	return nil
}

func (g *QuickSightGenerator) shouldLoadQuickSightResource(serviceName string) bool {
	if !g.hasTypedQuickSightFilter() {
		return true
	}
	return g.hasTypedFilterFor(serviceName) || g.hasUntypedFilter()
}

func (g *QuickSightGenerator) hasTypedQuickSightFilter() bool {
	for _, serviceName := range quickSightResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *QuickSightGenerator) hasTypedFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName {
			return true
		}
	}
	return false
}

func (g *QuickSightGenerator) hasUntypedFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "" {
			return true
		}
	}
	return false
}

func quickSightServiceName(resourceType string) string {
	return strings.TrimPrefix(resourceType, "aws_")
}

func listQuickSightNamespaces(svc quicksight.ListNamespacesAPIClient, accountID string) ([]quicksighttypes.NamespaceInfoV2, error) {
	p := quicksight.NewListNamespacesPaginator(svc, &quicksight.ListNamespacesInput{AwsAccountId: &accountID})
	namespaces := []quicksighttypes.NamespaceInfoV2{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		namespaces = append(namespaces, page.Namespaces...)
	}
	return namespaces, nil
}

func (g *QuickSightGenerator) loadNamespaces(accountID string, namespaces []quicksighttypes.NamespaceInfoV2) {
	for _, namespace := range namespaces {
		if resource, ok := newQuickSightNamespaceResource(accountID, namespace); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
}

func (g *QuickSightGenerator) loadGroupsAndMemberships(svc *quicksight.Client, accountID string, namespaces []quicksighttypes.NamespaceInfoV2, loadGroups, loadMemberships bool) error {
	for _, namespace := range namespaces {
		namespaceName := StringValue(namespace.Name)
		if namespaceName == "" || !quickSightNamespaceImportable(namespace.CreationStatus) {
			continue
		}
		groups, err := listQuickSightGroups(svc, accountID, namespaceName)
		if err != nil {
			if quickSightResourceNotFound(err) {
				continue
			}
			return err
		}
		if loadGroups {
			for _, group := range groups {
				if resource, ok := newQuickSightGroupResource(accountID, namespaceName, group); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
		if loadMemberships {
			for _, group := range groups {
				groupName := StringValue(group.GroupName)
				if groupName == "" {
					continue
				}
				memberships, err := listQuickSightGroupMemberships(svc, accountID, namespaceName, groupName)
				if err != nil {
					if quickSightResourceNotFound(err) {
						continue
					}
					return err
				}
				for _, membership := range memberships {
					if resource, ok := newQuickSightGroupMembershipResource(accountID, namespaceName, groupName, membership); ok {
						g.Resources = append(g.Resources, resource)
					}
				}
			}
		}
	}
	return nil
}

func listQuickSightGroups(svc quicksight.ListGroupsAPIClient, accountID, namespace string) ([]quicksighttypes.Group, error) {
	p := quicksight.NewListGroupsPaginator(svc, &quicksight.ListGroupsInput{
		AwsAccountId: &accountID,
		Namespace:    &namespace,
	})
	groups := []quicksighttypes.Group{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		groups = append(groups, page.GroupList...)
	}
	return groups, nil
}

func listQuickSightGroupMemberships(svc quicksight.ListGroupMembershipsAPIClient, accountID, namespace, groupName string) ([]quicksighttypes.GroupMember, error) {
	p := quicksight.NewListGroupMembershipsPaginator(svc, &quicksight.ListGroupMembershipsInput{
		AwsAccountId: &accountID,
		GroupName:    &groupName,
		Namespace:    &namespace,
	})
	memberships := []quicksighttypes.GroupMember{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		memberships = append(memberships, page.GroupMemberList...)
	}
	return memberships, nil
}

func listQuickSightFolders(svc quicksight.ListFoldersAPIClient, accountID string) ([]quicksighttypes.FolderSummary, error) {
	p := quicksight.NewListFoldersPaginator(svc, &quicksight.ListFoldersInput{AwsAccountId: &accountID})
	folders := []quicksighttypes.FolderSummary{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		folders = append(folders, page.FolderSummaryList...)
	}
	return folders, nil
}

func (g *QuickSightGenerator) loadFolders(accountID string, folders []quicksighttypes.FolderSummary) {
	for _, folder := range folders {
		if resource, ok := newQuickSightFolderResource(accountID, folder); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
}

func (g *QuickSightGenerator) loadFolderMemberships(svc *quicksight.Client, accountID string, folders []quicksighttypes.FolderSummary) error {
	for _, folder := range folders {
		folderID := StringValue(folder.FolderId)
		if folderID == "" {
			continue
		}
		members, err := listQuickSightFolderMembers(svc, accountID, folderID)
		if err != nil {
			if quickSightResourceNotFound(err) {
				continue
			}
			return err
		}
		for _, member := range members {
			if resource, ok := newQuickSightFolderMembershipResource(accountID, folderID, member); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func listQuickSightFolderMembers(svc quicksight.ListFolderMembersAPIClient, accountID, folderID string) ([]quicksighttypes.MemberIdArnPair, error) {
	p := quicksight.NewListFolderMembersPaginator(svc, &quicksight.ListFolderMembersInput{
		AwsAccountId: &accountID,
		FolderId:     &folderID,
	})
	members := []quicksighttypes.MemberIdArnPair{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		members = append(members, page.FolderMemberList...)
	}
	return members, nil
}

func (g *QuickSightGenerator) loadVPCConnections(svc *quicksight.Client, accountID string) error {
	connections, err := listQuickSightVPCConnections(svc, accountID)
	if err != nil {
		return err
	}
	for _, connection := range connections {
		if resource, ok := newQuickSightVPCConnectionResource(accountID, connection); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func listQuickSightVPCConnections(svc quicksight.ListVPCConnectionsAPIClient, accountID string) ([]quicksighttypes.VPCConnectionSummary, error) {
	p := quicksight.NewListVPCConnectionsPaginator(svc, &quicksight.ListVPCConnectionsInput{AwsAccountId: &accountID})
	connections := []quicksighttypes.VPCConnectionSummary{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		connections = append(connections, page.VPCConnectionSummaries...)
	}
	return connections, nil
}

func newQuickSightNamespaceResource(accountID string, namespace quicksighttypes.NamespaceInfoV2) (terraformutils.Resource, bool) {
	namespaceName := StringValue(namespace.Name)
	if accountID == "" || namespaceName == "" || !quickSightNamespaceImportable(namespace.CreationStatus) {
		return terraformutils.Resource{}, false
	}
	return quickSightResource(quickSightNamespaceImportID(accountID, namespaceName), quickSightResourceName("namespace", namespaceName), quickSightNamespaceResourceType, map[string]string{
		"aws_account_id": accountID,
		"identity_store": string(namespace.IdentityStore),
		"namespace":      namespaceName,
	})
}

func newQuickSightGroupResource(accountID, namespace string, group quicksighttypes.Group) (terraformutils.Resource, bool) {
	groupName := StringValue(group.GroupName)
	if accountID == "" || namespace == "" || groupName == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"aws_account_id": accountID,
		"group_name":     groupName,
		"namespace":      namespace,
	}
	if description := StringValue(group.Description); description != "" {
		attributes["description"] = description
	}
	return quickSightResource(quickSightGroupImportID(accountID, namespace, groupName), quickSightResourceName("group", namespace, groupName), quickSightGroupResourceType, attributes)
}

func newQuickSightGroupMembershipResource(accountID, namespace, groupName string, membership quicksighttypes.GroupMember) (terraformutils.Resource, bool) {
	memberName := StringValue(membership.MemberName)
	if accountID == "" || namespace == "" || groupName == "" || memberName == "" {
		return terraformutils.Resource{}, false
	}
	return quickSightResource(quickSightGroupMembershipImportID(accountID, namespace, groupName, memberName), quickSightResourceName("group-membership", namespace, groupName, memberName), quickSightGroupMembershipResourceType, map[string]string{
		"aws_account_id": accountID,
		"group_name":     groupName,
		"member_name":    memberName,
		"namespace":      namespace,
	})
}

func newQuickSightFolderResource(accountID string, folder quicksighttypes.FolderSummary) (terraformutils.Resource, bool) {
	folderID := StringValue(folder.FolderId)
	if accountID == "" || folderID == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"aws_account_id": accountID,
		"folder_id":      folderID,
	}
	if folder.FolderType != "" {
		attributes["folder_type"] = string(folder.FolderType)
	}
	if name := StringValue(folder.Name); name != "" {
		attributes["name"] = name
	}
	return quickSightResource(quickSightFolderImportID(accountID, folderID), quickSightResourceName("folder", folderID), quickSightFolderResourceType, attributes)
}

func newQuickSightFolderMembershipResource(accountID, folderID string, member quicksighttypes.MemberIdArnPair) (terraformutils.Resource, bool) {
	memberID := StringValue(member.MemberId)
	memberType := quickSightFolderMemberTypeFromARN(StringValue(member.MemberArn))
	if accountID == "" || folderID == "" || memberID == "" || memberType == "" {
		return terraformutils.Resource{}, false
	}
	return quickSightResource(quickSightFolderMembershipImportID(accountID, folderID, memberType, memberID), quickSightResourceName("folder-membership", folderID, memberType, memberID), quickSightFolderMembershipResourceType, map[string]string{
		"aws_account_id": accountID,
		"folder_id":      folderID,
		"member_id":      memberID,
		"member_type":    memberType,
	})
}

func newQuickSightVPCConnectionResource(accountID string, connection quicksighttypes.VPCConnectionSummary) (terraformutils.Resource, bool) {
	connectionID := StringValue(connection.VPCConnectionId)
	name := StringValue(connection.Name)
	roleARN := StringValue(connection.RoleArn)
	if accountID == "" || connectionID == "" || name == "" || roleARN == "" || !quickSightVPCConnectionImportable(connection.Status, connection.AvailabilityStatus) {
		return terraformutils.Resource{}, false
	}
	return quickSightResource(quickSightVPCConnectionImportID(accountID, connectionID), quickSightResourceName("vpc-connection", connectionID), quickSightVPCConnectionResourceType, map[string]string{
		"aws_account_id":    accountID,
		"name":              name,
		"role_arn":          roleARN,
		"vpc_connection_id": connectionID,
	})
}

func quickSightResource(importID, name, resourceType string, attributes map[string]string) (terraformutils.Resource, bool) {
	if importID == "" || name == "" || resourceType == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		importID,
		name,
		resourceType,
		"aws",
		attributes,
		quickSightAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func quickSightNamespaceImportID(accountID, namespace string) string {
	return strings.Join([]string{accountID, namespace}, quickSightCommaSeparator)
}

func quickSightGroupImportID(accountID, namespace, groupName string) string {
	return strings.Join([]string{accountID, namespace, groupName}, quickSightSlashSeparator)
}

func quickSightGroupMembershipImportID(accountID, namespace, groupName, memberName string) string {
	return strings.Join([]string{accountID, namespace, groupName, memberName}, quickSightSlashSeparator)
}

func quickSightFolderImportID(accountID, folderID string) string {
	return strings.Join([]string{accountID, folderID}, quickSightCommaSeparator)
}

func quickSightFolderMembershipImportID(accountID, folderID, memberType, memberID string) string {
	return strings.Join([]string{accountID, folderID, memberType, memberID}, quickSightCommaSeparator)
}

func quickSightVPCConnectionImportID(accountID, connectionID string) string {
	return strings.Join([]string{accountID, connectionID}, quickSightCommaSeparator)
}

func quickSightFolderMemberTypeFromARN(memberARN string) string {
	resource := arnLastSegment(memberARN, ":")
	switch {
	case strings.HasPrefix(resource, "dashboard/"):
		return string(quicksighttypes.MemberTypeDashboard)
	case strings.HasPrefix(resource, "analysis/"):
		return string(quicksighttypes.MemberTypeAnalysis)
	case strings.HasPrefix(resource, "dataset/"):
		return string(quicksighttypes.MemberTypeDataset)
	case strings.HasPrefix(resource, "datasource/"):
		return string(quicksighttypes.MemberTypeDatasource)
	case strings.HasPrefix(resource, "topic/"):
		return string(quicksighttypes.MemberTypeTopic)
	default:
		return ""
	}
}

func quickSightResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d-%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return quickSightResourceNameFallback
	}
	return strings.Join(cleanParts, "/")
}

func quickSightNamespaceImportable(status quicksighttypes.NamespaceStatus) bool {
	return status == quicksighttypes.NamespaceStatusCreated
}

func quickSightVPCConnectionImportable(status quicksighttypes.VPCConnectionResourceStatus, availability quicksighttypes.VPCConnectionAvailabilityStatus) bool {
	if status != quicksighttypes.VPCConnectionResourceStatusCreationSuccessful && status != quicksighttypes.VPCConnectionResourceStatusUpdateSuccessful {
		return false
	}
	return availability == "" ||
		availability == quicksighttypes.VPCConnectionAvailabilityStatusAvailable ||
		availability == quicksighttypes.VPCConnectionAvailabilityStatusPartiallyAvailable
}

func quickSightResourceNotFound(err error) bool {
	var notFound *quicksighttypes.ResourceNotFoundException
	if errors.As(err, &notFound) {
		return true
	}
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	errorCode := strings.ToLower(apiErr.ErrorCode())
	errorMessage := strings.ToLower(apiErr.ErrorMessage())
	return strings.Contains(errorCode, "notfound") ||
		strings.Contains(errorCode, "not_found") ||
		strings.Contains(errorMessage, "not found") ||
		strings.Contains(errorMessage, "does not exist")
}

func quickSightServiceUnavailable(err error) bool {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	errorCode := strings.ToLower(apiErr.ErrorCode())
	errorMessage := strings.ToLower(apiErr.ErrorMessage())
	return quickSightResourceNotFound(err) ||
		strings.Contains(errorCode, "unsupporteduseredition") ||
		strings.Contains(errorMessage, "not subscribed") ||
		strings.Contains(errorMessage, "not currently subscribed") ||
		strings.Contains(errorMessage, "not enabled") ||
		strings.Contains(errorMessage, "account is not setup") ||
		strings.Contains(errorMessage, "precondition")
}
