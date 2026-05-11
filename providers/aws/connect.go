// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/connect"
	connecttypes "github.com/aws/aws-sdk-go-v2/service/connect/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	connectBotAssociationResourceType            = "aws_connect_bot_association"
	connectHoursOfOperationResourceType          = "aws_connect_hours_of_operation"
	connectInstanceResourceType                  = "aws_connect_instance"
	connectInstanceStorageConfigResourceType     = "aws_connect_instance_storage_config"
	connectLambdaFunctionAssociationResourceType = "aws_connect_lambda_function_association"
	connectPhoneNumberResourceType               = "aws_connect_phone_number"
	connectQueueResourceType                     = "aws_connect_queue"
	connectQuickConnectResourceType              = "aws_connect_quick_connect"
	connectRoutingProfileResourceType            = "aws_connect_routing_profile"
	connectSecurityProfileResourceType           = "aws_connect_security_profile"
	connectUserResourceType                      = "aws_connect_user"
	connectUserHierarchyGroupResourceType        = "aws_connect_user_hierarchy_group"
	connectUserHierarchyStructureResourceType    = "aws_connect_user_hierarchy_structure"

	connectResourceIDSeparator                  = ":"
	connectLambdaFunctionAssociationIDSeparator = ","
)

var connectAllowEmptyValues = []string{"tags."}

type ConnectGenerator struct {
	AWSService
}

type connectInstanceReference struct {
	id string
}

type connectOptionalResourceLoader struct {
	name string
	load func() error
}

func (g *ConnectGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := connect.NewFromConfig(config)

	instances, err := g.loadInstances(svc)
	if err != nil {
		return err
	}
	for _, instance := range instances {
		g.getOptionalConnectResources(
			connectOptionalResourceLoader{name: "instance storage configs", load: func() error {
				return g.loadInstanceStorageConfigs(svc, instance.id)
			}},
			connectOptionalResourceLoader{name: "lambda function associations", load: func() error {
				return g.loadLambdaFunctionAssociations(svc, instance.id)
			}},
			connectOptionalResourceLoader{name: "bot associations", load: func() error {
				return g.loadBotAssociations(svc, instance.id)
			}},
			connectOptionalResourceLoader{name: "hours of operation", load: func() error {
				return g.loadHoursOfOperations(svc, instance.id)
			}},
			connectOptionalResourceLoader{name: "queues", load: func() error {
				return g.loadQueues(svc, instance.id)
			}},
			connectOptionalResourceLoader{name: "quick connects", load: func() error {
				return g.loadQuickConnects(svc, instance.id)
			}},
			connectOptionalResourceLoader{name: "routing profiles", load: func() error {
				return g.loadRoutingProfiles(svc, instance.id)
			}},
			connectOptionalResourceLoader{name: "security profiles", load: func() error {
				return g.loadSecurityProfiles(svc, instance.id)
			}},
			connectOptionalResourceLoader{name: "users", load: func() error {
				return g.loadUsers(svc, instance.id)
			}},
			connectOptionalResourceLoader{name: "user hierarchy groups", load: func() error {
				return g.loadUserHierarchyGroups(svc, instance.id)
			}},
			connectOptionalResourceLoader{name: "user hierarchy structure", load: func() error {
				return g.loadUserHierarchyStructure(svc, instance.id)
			}},
			connectOptionalResourceLoader{name: "phone numbers", load: func() error {
				return g.loadPhoneNumbers(svc, instance.id)
			}},
		)
	}
	return nil
}

func (g *ConnectGenerator) loadInstances(svc *connect.Client) ([]connectInstanceReference, error) {
	instances := []connectInstanceReference{}
	p := connect.NewListInstancesPaginator(svc, &connect.ListInstancesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		for _, instance := range page.InstanceSummaryList {
			resource, ok := newConnectInstanceResource(instance)
			if !ok {
				continue
			}
			g.Resources = append(g.Resources, resource)
			instances = append(instances, connectInstanceReference{
				id: StringValue(instance.Id),
			})
		}
	}
	return instances, nil
}

func (g *ConnectGenerator) loadInstanceStorageConfigs(svc *connect.Client, instanceID string) error {
	for _, resourceType := range connecttypes.InstanceStorageResourceType("").Values() {
		if resourceType == "" {
			continue
		}
		p := connect.NewListInstanceStorageConfigsPaginator(svc, &connect.ListInstanceStorageConfigsInput{
			InstanceId:   &instanceID,
			ResourceType: resourceType,
		})
		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if err != nil {
				if connectNotFound(err) {
					break
				}
				return err
			}
			for _, config := range page.StorageConfigs {
				if resource, ok := newConnectInstanceStorageConfigResource(instanceID, resourceType, config); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func (g *ConnectGenerator) loadLambdaFunctionAssociations(svc *connect.Client, instanceID string) error {
	p := connect.NewListLambdaFunctionsPaginator(svc, &connect.ListLambdaFunctionsInput{InstanceId: &instanceID})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, functionARN := range page.LambdaFunctions {
			if resource, ok := newConnectLambdaFunctionAssociationResource(instanceID, functionARN); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *ConnectGenerator) loadBotAssociations(svc *connect.Client, instanceID string) error {
	p := connect.NewListBotsPaginator(svc, &connect.ListBotsInput{
		InstanceId: &instanceID,
		LexVersion: connecttypes.LexVersionV1,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, bot := range page.LexBots {
			if resource, ok := newConnectBotAssociationResource(instanceID, bot); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *ConnectGenerator) loadHoursOfOperations(svc *connect.Client, instanceID string) error {
	p := connect.NewListHoursOfOperationsPaginator(svc, &connect.ListHoursOfOperationsInput{InstanceId: &instanceID})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, hours := range page.HoursOfOperationSummaryList {
			if resource, ok := newConnectHoursOfOperationResource(instanceID, hours); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *ConnectGenerator) loadQueues(svc *connect.Client, instanceID string) error {
	p := connect.NewListQueuesPaginator(svc, &connect.ListQueuesInput{
		InstanceId: &instanceID,
		QueueTypes: []connecttypes.QueueType{connecttypes.QueueTypeStandard},
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, queue := range page.QueueSummaryList {
			if resource, ok := newConnectQueueResource(instanceID, queue); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *ConnectGenerator) loadQuickConnects(svc *connect.Client, instanceID string) error {
	p := connect.NewListQuickConnectsPaginator(svc, &connect.ListQuickConnectsInput{InstanceId: &instanceID})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, quickConnect := range page.QuickConnectSummaryList {
			if resource, ok := newConnectQuickConnectResource(instanceID, quickConnect); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *ConnectGenerator) loadRoutingProfiles(svc *connect.Client, instanceID string) error {
	p := connect.NewListRoutingProfilesPaginator(svc, &connect.ListRoutingProfilesInput{InstanceId: &instanceID})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, profile := range page.RoutingProfileSummaryList {
			if resource, ok := newConnectRoutingProfileResource(instanceID, profile); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *ConnectGenerator) loadSecurityProfiles(svc *connect.Client, instanceID string) error {
	p := connect.NewListSecurityProfilesPaginator(svc, &connect.ListSecurityProfilesInput{InstanceId: &instanceID})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, profile := range page.SecurityProfileSummaryList {
			if resource, ok := newConnectSecurityProfileResource(instanceID, profile); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *ConnectGenerator) loadUsers(svc *connect.Client, instanceID string) error {
	p := connect.NewListUsersPaginator(svc, &connect.ListUsersInput{InstanceId: &instanceID})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, user := range page.UserSummaryList {
			if resource, ok := newConnectUserResource(instanceID, user); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *ConnectGenerator) loadUserHierarchyGroups(svc *connect.Client, instanceID string) error {
	p := connect.NewListUserHierarchyGroupsPaginator(svc, &connect.ListUserHierarchyGroupsInput{InstanceId: &instanceID})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, summary := range page.UserHierarchyGroupSummaryList {
			groupID := StringValue(summary.Id)
			if groupID == "" {
				continue
			}
			output, err := svc.DescribeUserHierarchyGroup(context.TODO(), &connect.DescribeUserHierarchyGroupInput{
				HierarchyGroupId: &groupID,
				InstanceId:       &instanceID,
			})
			if err != nil {
				if connectNotFound(err) {
					continue
				}
				return err
			}
			if output == nil || output.HierarchyGroup == nil {
				continue
			}
			if resource, ok := newConnectUserHierarchyGroupResource(instanceID, *output.HierarchyGroup); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *ConnectGenerator) loadUserHierarchyStructure(svc *connect.Client, instanceID string) error {
	output, err := svc.DescribeUserHierarchyStructure(context.TODO(), &connect.DescribeUserHierarchyStructureInput{InstanceId: &instanceID})
	if err != nil {
		if connectNotFound(err) {
			return nil
		}
		return err
	}
	if output == nil || output.HierarchyStructure == nil {
		return nil
	}
	if resource, ok := newConnectUserHierarchyStructureResource(instanceID, *output.HierarchyStructure); ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *ConnectGenerator) loadPhoneNumbers(svc *connect.Client, instanceID string) error {
	p := connect.NewListPhoneNumbersV2Paginator(svc, &connect.ListPhoneNumbersV2Input{InstanceId: &instanceID})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, summary := range page.ListPhoneNumbersSummaryList {
			phoneNumberID := StringValue(summary.PhoneNumberId)
			if phoneNumberID == "" {
				continue
			}
			output, err := svc.DescribePhoneNumber(context.TODO(), &connect.DescribePhoneNumberInput{PhoneNumberId: &phoneNumberID})
			if err != nil {
				if connectNotFound(err) {
					continue
				}
				return err
			}
			if output == nil || output.ClaimedPhoneNumberSummary == nil {
				continue
			}
			if resource, ok := newConnectPhoneNumberResource(*output.ClaimedPhoneNumberSummary); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *ConnectGenerator) getOptionalConnectResources(loaders ...connectOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			log.Printf("skipping Connect %s discovery: %v", loader.name, err)
		}
	}
}

func newConnectInstanceResource(instance connecttypes.InstanceSummary) (terraformutils.Resource, bool) {
	instanceID := StringValue(instance.Id)
	instanceAlias := StringValue(instance.InstanceAlias)
	if instanceID == "" || instanceAlias == "" || !connectInstanceImportable(instance) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"identity_management_type": string(instance.IdentityManagementType),
		"instance_alias":           instanceAlias,
	}
	if instance.InboundCallsEnabled != nil {
		attributes["inbound_calls_enabled"] = strconv.FormatBool(*instance.InboundCallsEnabled)
	}
	if instance.OutboundCallsEnabled != nil {
		attributes["outbound_calls_enabled"] = strconv.FormatBool(*instance.OutboundCallsEnabled)
	}
	return terraformutils.NewResource(
		connectInstanceImportID(instanceID),
		connectResourceName("instance", instanceAlias, instanceID),
		connectInstanceResourceType,
		"aws",
		attributes,
		connectAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newConnectInstanceStorageConfigResource(instanceID string, resourceType connecttypes.InstanceStorageResourceType, config connecttypes.InstanceStorageConfig) (terraformutils.Resource, bool) {
	associationID := StringValue(config.AssociationId)
	resourceTypeValue := string(resourceType)
	if instanceID == "" || associationID == "" || resourceTypeValue == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		connectInstanceStorageConfigImportID(instanceID, associationID, resourceTypeValue),
		connectResourceName("instance_storage_config", instanceID, resourceTypeValue, associationID),
		connectInstanceStorageConfigResourceType,
		"aws",
		map[string]string{
			"association_id": associationID,
			"instance_id":    instanceID,
			"resource_type":  resourceTypeValue,
		},
		connectAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newConnectLambdaFunctionAssociationResource(instanceID, functionARN string) (terraformutils.Resource, bool) {
	if instanceID == "" || functionARN == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		connectLambdaFunctionAssociationImportID(instanceID, functionARN),
		connectResourceName("lambda_function_association", instanceID, functionARN),
		connectLambdaFunctionAssociationResourceType,
		"aws",
		map[string]string{
			"function_arn": functionARN,
			"instance_id":  instanceID,
		},
		connectAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newConnectBotAssociationResource(instanceID string, bot connecttypes.LexBotConfig) (terraformutils.Resource, bool) {
	if instanceID == "" || bot.LexBot == nil {
		return terraformutils.Resource{}, false
	}
	botName := StringValue(bot.LexBot.Name)
	lexRegion := StringValue(bot.LexBot.LexRegion)
	if botName == "" || lexRegion == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		connectBotAssociationImportID(instanceID, botName, lexRegion),
		connectResourceName("bot_association", instanceID, lexRegion, botName),
		connectBotAssociationResourceType,
		"aws",
		map[string]string{
			"instance_id": instanceID,
		},
		connectAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newConnectHoursOfOperationResource(instanceID string, hours connecttypes.HoursOfOperationSummary) (terraformutils.Resource, bool) {
	resourceID := StringValue(hours.Id)
	if instanceID == "" || resourceID == "" {
		return terraformutils.Resource{}, false
	}
	attributes := connectChildAttributes(instanceID, resourceID, "hours_of_operation_id")
	if name := StringValue(hours.Name); name != "" {
		attributes["name"] = name
	}
	return terraformutils.NewResource(
		connectTwoPartImportID(instanceID, resourceID),
		connectResourceName("hours_of_operation", instanceID, StringValue(hours.Name), resourceID),
		connectHoursOfOperationResourceType,
		"aws",
		attributes,
		connectAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newConnectQueueResource(instanceID string, queue connecttypes.QueueSummary) (terraformutils.Resource, bool) {
	queueID := StringValue(queue.Id)
	if instanceID == "" || queueID == "" || queue.QueueType == connecttypes.QueueTypeAgent {
		return terraformutils.Resource{}, false
	}
	attributes := connectChildAttributes(instanceID, queueID, "queue_id")
	if name := StringValue(queue.Name); name != "" {
		attributes["name"] = name
	}
	return terraformutils.NewResource(
		connectTwoPartImportID(instanceID, queueID),
		connectResourceName("queue", instanceID, StringValue(queue.Name), queueID),
		connectQueueResourceType,
		"aws",
		attributes,
		connectAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newConnectQuickConnectResource(instanceID string, quickConnect connecttypes.QuickConnectSummary) (terraformutils.Resource, bool) {
	quickConnectID := StringValue(quickConnect.Id)
	if instanceID == "" || quickConnectID == "" {
		return terraformutils.Resource{}, false
	}
	attributes := connectChildAttributes(instanceID, quickConnectID, "quick_connect_id")
	if name := StringValue(quickConnect.Name); name != "" {
		attributes["name"] = name
	}
	return terraformutils.NewResource(
		connectTwoPartImportID(instanceID, quickConnectID),
		connectResourceName("quick_connect", instanceID, StringValue(quickConnect.Name), quickConnectID),
		connectQuickConnectResourceType,
		"aws",
		attributes,
		connectAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newConnectRoutingProfileResource(instanceID string, profile connecttypes.RoutingProfileSummary) (terraformutils.Resource, bool) {
	profileID := StringValue(profile.Id)
	if instanceID == "" || profileID == "" {
		return terraformutils.Resource{}, false
	}
	attributes := connectChildAttributes(instanceID, profileID, "routing_profile_id")
	if name := StringValue(profile.Name); name != "" {
		attributes["name"] = name
	}
	return terraformutils.NewResource(
		connectTwoPartImportID(instanceID, profileID),
		connectResourceName("routing_profile", instanceID, StringValue(profile.Name), profileID),
		connectRoutingProfileResourceType,
		"aws",
		attributes,
		connectAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newConnectSecurityProfileResource(instanceID string, profile connecttypes.SecurityProfileSummary) (terraformutils.Resource, bool) {
	profileID := StringValue(profile.Id)
	if instanceID == "" || profileID == "" {
		return terraformutils.Resource{}, false
	}
	attributes := connectChildAttributes(instanceID, profileID, "security_profile_id")
	if name := StringValue(profile.Name); name != "" {
		attributes["name"] = name
	}
	return terraformutils.NewResource(
		connectTwoPartImportID(instanceID, profileID),
		connectResourceName("security_profile", instanceID, StringValue(profile.Name), profileID),
		connectSecurityProfileResourceType,
		"aws",
		attributes,
		connectAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newConnectUserResource(instanceID string, user connecttypes.UserSummary) (terraformutils.Resource, bool) {
	userID := StringValue(user.Id)
	if instanceID == "" || userID == "" {
		return terraformutils.Resource{}, false
	}
	attributes := connectChildAttributes(instanceID, userID, "user_id")
	if username := StringValue(user.Username); username != "" {
		attributes["name"] = username
	}
	return terraformutils.NewResource(
		connectTwoPartImportID(instanceID, userID),
		connectResourceName("user", instanceID, StringValue(user.Username), userID),
		connectUserResourceType,
		"aws",
		attributes,
		connectAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newConnectUserHierarchyGroupResource(instanceID string, group connecttypes.HierarchyGroup) (terraformutils.Resource, bool) {
	groupID := StringValue(group.Id)
	if instanceID == "" || groupID == "" {
		return terraformutils.Resource{}, false
	}
	attributes := connectChildAttributes(instanceID, groupID, "hierarchy_group_id")
	if name := StringValue(group.Name); name != "" {
		attributes["name"] = name
	}
	if parentGroupID := connectHierarchyGroupParentID(group); parentGroupID != "" {
		attributes["parent_group_id"] = parentGroupID
	}
	return terraformutils.NewResource(
		connectTwoPartImportID(instanceID, groupID),
		connectResourceName("user_hierarchy_group", instanceID, StringValue(group.Name), groupID),
		connectUserHierarchyGroupResourceType,
		"aws",
		attributes,
		connectAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newConnectUserHierarchyStructureResource(instanceID string, structure connecttypes.HierarchyStructure) (terraformutils.Resource, bool) {
	if instanceID == "" || !connectHierarchyStructureConfigured(structure) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		connectInstanceImportID(instanceID),
		connectResourceName("user_hierarchy_structure", instanceID),
		connectUserHierarchyStructureResourceType,
		"aws",
		map[string]string{
			"instance_id": instanceID,
		},
		connectAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newConnectPhoneNumberResource(phoneNumber connecttypes.ClaimedPhoneNumberSummary) (terraformutils.Resource, bool) {
	phoneNumberID := StringValue(phoneNumber.PhoneNumberId)
	targetARN := StringValue(phoneNumber.TargetArn)
	if phoneNumberID == "" || targetARN == "" || phoneNumber.PhoneNumberCountryCode == "" || phoneNumber.PhoneNumberType == "" || !connectPhoneNumberImportable(phoneNumber) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"country_code": string(phoneNumber.PhoneNumberCountryCode),
		"target_arn":   targetARN,
		"type":         string(phoneNumber.PhoneNumberType),
	}
	if description := StringValue(phoneNumber.PhoneNumberDescription); description != "" {
		attributes["description"] = description
	}
	return terraformutils.NewResource(
		connectPhoneNumberImportID(phoneNumberID),
		connectResourceName("phone_number", StringValue(phoneNumber.PhoneNumber), phoneNumberID),
		connectPhoneNumberResourceType,
		"aws",
		attributes,
		connectAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func connectChildAttributes(instanceID, resourceID, resourceIDField string) map[string]string {
	return map[string]string{
		"instance_id":   instanceID,
		resourceIDField: resourceID,
	}
}

func connectInstanceImportID(instanceID string) string {
	return instanceID
}

func connectInstanceStorageConfigImportID(instanceID, associationID, resourceType string) string {
	return strings.Join([]string{instanceID, associationID, resourceType}, connectResourceIDSeparator)
}

func connectLambdaFunctionAssociationImportID(instanceID, functionARN string) string {
	return strings.Join([]string{instanceID, functionARN}, connectLambdaFunctionAssociationIDSeparator)
}

func connectBotAssociationImportID(instanceID, botName, lexRegion string) string {
	return strings.Join([]string{instanceID, botName, lexRegion}, connectResourceIDSeparator)
}

func connectTwoPartImportID(instanceID, resourceID string) string {
	return strings.Join([]string{instanceID, resourceID}, connectResourceIDSeparator)
}

func connectPhoneNumberImportID(phoneNumberID string) string {
	return phoneNumberID
}

func connectResourceName(parts ...string) string {
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

func connectInstanceImportable(instance connecttypes.InstanceSummary) bool {
	return instance.InstanceStatus == "" || instance.InstanceStatus == connecttypes.InstanceStatusActive
}

func connectHierarchyStructureConfigured(structure connecttypes.HierarchyStructure) bool {
	return structure.LevelOne != nil ||
		structure.LevelTwo != nil ||
		structure.LevelThree != nil ||
		structure.LevelFour != nil ||
		structure.LevelFive != nil
}

func connectHierarchyGroupParentID(group connecttypes.HierarchyGroup) string {
	groupID := StringValue(group.Id)
	if groupID == "" || group.HierarchyPath == nil {
		return ""
	}
	parentID := ""
	for _, levelID := range connectHierarchyPathIDs(*group.HierarchyPath) {
		if levelID == "" {
			continue
		}
		if levelID == groupID {
			return parentID
		}
		parentID = levelID
	}
	return parentID
}

func connectHierarchyPathIDs(path connecttypes.HierarchyPath) []string {
	return []string{
		connectHierarchyGroupSummaryID(path.LevelOne),
		connectHierarchyGroupSummaryID(path.LevelTwo),
		connectHierarchyGroupSummaryID(path.LevelThree),
		connectHierarchyGroupSummaryID(path.LevelFour),
		connectHierarchyGroupSummaryID(path.LevelFive),
	}
}

func connectHierarchyGroupSummaryID(group *connecttypes.HierarchyGroupSummary) string {
	if group == nil {
		return ""
	}
	return StringValue(group.Id)
}

func connectPhoneNumberImportable(phoneNumber connecttypes.ClaimedPhoneNumberSummary) bool {
	if phoneNumber.PhoneNumberStatus == nil {
		return true
	}
	return phoneNumber.PhoneNumberStatus.Status == "" || phoneNumber.PhoneNumberStatus.Status == connecttypes.PhoneNumberWorkflowStatusClaimed
}

func connectNotFound(err error) bool {
	var notFound *connecttypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
