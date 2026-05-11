// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	ssotypes "github.com/aws/aws-sdk-go-v2/service/ssoadmin/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var ssoAdminAllowEmptyValues = []string{"tags."}

const (
	ssoAdminPermissionSetResourceType                     = "aws_ssoadmin_permission_set"
	ssoAdminAccountAssignmentResourceType                 = "aws_ssoadmin_account_assignment"
	ssoAdminInstanceAccessControlAttributesResourceType   = "aws_ssoadmin_instance_access_control_attributes"
	ssoAdminManagedPolicyAttachmentResourceType           = "aws_ssoadmin_managed_policy_attachment"
	ssoAdminCustomerManagedPolicyAttachmentResourceType   = "aws_ssoadmin_customer_managed_policy_attachment"
	ssoAdminPermissionSetInlinePolicyResourceType         = "aws_ssoadmin_permission_set_inline_policy"
	ssoAdminPermissionsBoundaryAttachmentResourceType     = "aws_ssoadmin_permissions_boundary_attachment"
	ssoAdminDefaultCustomerManagedPolicyPath              = "/"
	ssoAdminResourceIDSeparator                           = ","
	ssoAdminResourceNameSeparator                         = ":"
	ssoAdminNestedPermissionsBoundaryAttributePrefix      = "permissions_boundary.0"
	ssoAdminNestedCustomerManagedPolicyReferenceAttribute = "permissions_boundary.0.customer_managed_policy_reference.0"
)

type SSOAdminGenerator struct {
	AWSService
}

func (g *SSOAdminGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := ssoadmin.NewFromConfig(config)
	instances, err := listSSOAdminInstances(svc)
	if err != nil {
		return err
	}
	for _, instance := range instances {
		instanceARN := StringValue(instance.InstanceArn)
		if instanceARN == "" {
			continue
		}
		resourceStart := len(g.Resources)
		if err := g.loadInstanceAccessControlAttributes(svc, instanceARN); err != nil {
			return err
		}
		if err := g.loadPermissionSets(svc, instanceARN); err != nil {
			if ssoAdminResourceNotFound(err) {
				g.Resources = g.Resources[:resourceStart]
				continue
			}
			return err
		}
	}
	return nil
}

func (g *SSOAdminGenerator) PostConvertHook() error {
	g.updateManagedPolicyAttachmentDependencies()
	for i, resource := range g.Resources {
		if resource.InstanceInfo == nil {
			continue
		}
		switch resource.InstanceInfo.Type {
		case ssoAdminPermissionSetInlinePolicyResourceType:
			inlinePolicy, ok := resource.Item["inline_policy"].(string)
			if !ok || inlinePolicy == "" {
				continue
			}
			g.Resources[i].Item["inline_policy"] = fmt.Sprintf("<<POLICY\n%s\nPOLICY", g.escapeAwsInterpolation(inlinePolicy))
		case ssoAdminInstanceAccessControlAttributesResourceType:
			ssoAdminEscapeInstanceAccessControlAttributeSources(g.Resources[i].Item)
		}
	}
	return nil
}

func listSSOAdminInstances(svc *ssoadmin.Client) ([]ssotypes.InstanceMetadata, error) {
	p := ssoadmin.NewListInstancesPaginator(svc, &ssoadmin.ListInstancesInput{})
	var instances []ssotypes.InstanceMetadata
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		instances = append(instances, page.Instances...)
	}
	return instances, nil
}

func (g *SSOAdminGenerator) loadPermissionSets(svc *ssoadmin.Client, instanceARN string) error {
	p := ssoadmin.NewListPermissionSetsPaginator(svc, &ssoadmin.ListPermissionSetsInput{
		InstanceArn: aws.String(instanceARN),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, permissionSetARN := range page.PermissionSets {
			if permissionSetARN == "" {
				continue
			}
			permissionSet, err := describeSSOAdminPermissionSet(svc, instanceARN, permissionSetARN)
			if err != nil {
				if ssoAdminResourceNotFound(err) {
					continue
				}
				return err
			}
			if permissionSet == nil {
				continue
			}
			resourceStart := len(g.Resources)
			g.Resources = append(g.Resources, newSSOAdminPermissionSetResource(instanceARN, permissionSet))
			if err := g.loadPermissionSetChildren(svc, instanceARN, permissionSetARN); err != nil {
				if ssoAdminResourceNotFound(err) {
					g.Resources = g.Resources[:resourceStart]
					continue
				}
				return err
			}
		}
	}
	return nil
}

func (g *SSOAdminGenerator) loadPermissionSetChildren(svc *ssoadmin.Client, instanceARN, permissionSetARN string) error {
	if err := g.loadManagedPolicyAttachments(svc, instanceARN, permissionSetARN); err != nil {
		return err
	}
	if err := g.loadCustomerManagedPolicyAttachments(svc, instanceARN, permissionSetARN); err != nil {
		return err
	}
	if err := g.loadPermissionSetInlinePolicy(svc, instanceARN, permissionSetARN); err != nil {
		return err
	}
	if err := g.loadPermissionsBoundaryAttachment(svc, instanceARN, permissionSetARN); err != nil {
		return err
	}
	return g.loadAccountAssignments(svc, instanceARN, permissionSetARN)
}

func describeSSOAdminPermissionSet(svc *ssoadmin.Client, instanceARN, permissionSetARN string) (*ssotypes.PermissionSet, error) {
	output, err := svc.DescribePermissionSet(context.TODO(), &ssoadmin.DescribePermissionSetInput{
		InstanceArn:      aws.String(instanceARN),
		PermissionSetArn: aws.String(permissionSetARN),
	})
	if err != nil {
		return nil, err
	}
	if output == nil || output.PermissionSet == nil {
		return nil, nil
	}
	return output.PermissionSet, nil
}

func (g *SSOAdminGenerator) loadInstanceAccessControlAttributes(svc *ssoadmin.Client, instanceARN string) error {
	output, err := svc.DescribeInstanceAccessControlAttributeConfiguration(context.TODO(), &ssoadmin.DescribeInstanceAccessControlAttributeConfigurationInput{
		InstanceArn: aws.String(instanceARN),
	})
	if err != nil {
		if ssoAdminInstanceAccessControlAttributesNotConfigured(err) {
			return nil
		}
		return err
	}
	if output == nil || !ssoAdminInstanceAccessControlAttributesConfigured(output.InstanceAccessControlAttributeConfiguration) {
		return nil
	}
	g.Resources = append(g.Resources, newSSOAdminInstanceAccessControlAttributesResource(instanceARN, output.InstanceAccessControlAttributeConfiguration))
	return nil
}

func (g *SSOAdminGenerator) loadManagedPolicyAttachments(svc *ssoadmin.Client, instanceARN, permissionSetARN string) error {
	p := ssoadmin.NewListManagedPoliciesInPermissionSetPaginator(svc, &ssoadmin.ListManagedPoliciesInPermissionSetInput{
		InstanceArn:      aws.String(instanceARN),
		PermissionSetArn: aws.String(permissionSetARN),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, policy := range page.AttachedManagedPolicies {
			if StringValue(policy.Arn) == "" {
				continue
			}
			g.Resources = append(g.Resources, newSSOAdminManagedPolicyAttachmentResource(instanceARN, permissionSetARN, policy))
		}
	}
	return nil
}

func (g *SSOAdminGenerator) loadCustomerManagedPolicyAttachments(svc *ssoadmin.Client, instanceARN, permissionSetARN string) error {
	p := ssoadmin.NewListCustomerManagedPolicyReferencesInPermissionSetPaginator(svc, &ssoadmin.ListCustomerManagedPolicyReferencesInPermissionSetInput{
		InstanceArn:      aws.String(instanceARN),
		PermissionSetArn: aws.String(permissionSetARN),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, policy := range page.CustomerManagedPolicyReferences {
			if StringValue(policy.Name) == "" {
				continue
			}
			g.Resources = append(g.Resources, newSSOAdminCustomerManagedPolicyAttachmentResource(instanceARN, permissionSetARN, policy))
		}
	}
	return nil
}

func (g *SSOAdminGenerator) loadPermissionSetInlinePolicy(svc *ssoadmin.Client, instanceARN, permissionSetARN string) error {
	output, err := svc.GetInlinePolicyForPermissionSet(context.TODO(), &ssoadmin.GetInlinePolicyForPermissionSetInput{
		InstanceArn:      aws.String(instanceARN),
		PermissionSetArn: aws.String(permissionSetARN),
	})
	if err != nil {
		if ssoAdminResourceNotFound(err) {
			return nil
		}
		return err
	}
	inlinePolicy := ""
	if output != nil {
		inlinePolicy = StringValue(output.InlinePolicy)
	}
	if inlinePolicy == "" {
		return nil
	}
	g.Resources = append(g.Resources, newSSOAdminPermissionSetInlinePolicyResource(instanceARN, permissionSetARN, inlinePolicy))
	return nil
}

func (g *SSOAdminGenerator) loadPermissionsBoundaryAttachment(svc *ssoadmin.Client, instanceARN, permissionSetARN string) error {
	output, err := svc.GetPermissionsBoundaryForPermissionSet(context.TODO(), &ssoadmin.GetPermissionsBoundaryForPermissionSetInput{
		InstanceArn:      aws.String(instanceARN),
		PermissionSetArn: aws.String(permissionSetARN),
	})
	if err != nil {
		if ssoAdminResourceNotFound(err) {
			return nil
		}
		return err
	}
	if output == nil || output.PermissionsBoundary == nil || !ssoAdminPermissionsBoundaryConfigured(output.PermissionsBoundary) {
		return nil
	}
	g.Resources = append(g.Resources, newSSOAdminPermissionsBoundaryAttachmentResource(instanceARN, permissionSetARN, output.PermissionsBoundary))
	return nil
}

func (g *SSOAdminGenerator) loadAccountAssignments(svc *ssoadmin.Client, instanceARN, permissionSetARN string) error {
	p := ssoadmin.NewListAccountsForProvisionedPermissionSetPaginator(svc, &ssoadmin.ListAccountsForProvisionedPermissionSetInput{
		InstanceArn:      aws.String(instanceARN),
		PermissionSetArn: aws.String(permissionSetARN),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, accountID := range page.AccountIds {
			if accountID == "" {
				continue
			}
			if err := g.loadAccountAssignmentsForAccount(svc, instanceARN, permissionSetARN, accountID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *SSOAdminGenerator) loadAccountAssignmentsForAccount(svc *ssoadmin.Client, instanceARN, permissionSetARN, accountID string) error {
	p := ssoadmin.NewListAccountAssignmentsPaginator(svc, &ssoadmin.ListAccountAssignmentsInput{
		AccountId:        aws.String(accountID),
		InstanceArn:      aws.String(instanceARN),
		PermissionSetArn: aws.String(permissionSetARN),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, assignment := range page.AccountAssignments {
			targetID := StringValue(assignment.AccountId)
			if targetID == "" {
				targetID = accountID
			}
			if !ssoAdminAccountAssignmentConfigured(targetID, assignment) {
				continue
			}
			g.Resources = append(g.Resources, newSSOAdminAccountAssignmentResource(instanceARN, permissionSetARN, targetID, assignment))
		}
	}
	return nil
}

func newSSOAdminPermissionSetResource(instanceARN string, permissionSet *ssotypes.PermissionSet) terraformutils.Resource {
	permissionSetARN := StringValue(permissionSet.PermissionSetArn)
	attributes := map[string]string{
		"arn":          permissionSetARN,
		"instance_arn": instanceARN,
	}
	if createdDate := permissionSet.CreatedDate; createdDate != nil {
		attributes["created_date"] = createdDate.Format(time.RFC3339)
	}
	if description := StringValue(permissionSet.Description); description != "" {
		attributes["description"] = description
	}
	if name := StringValue(permissionSet.Name); name != "" {
		attributes["name"] = name
	}
	if relayState := StringValue(permissionSet.RelayState); relayState != "" {
		attributes["relay_state"] = relayState
	}
	if sessionDuration := StringValue(permissionSet.SessionDuration); sessionDuration != "" {
		attributes["session_duration"] = sessionDuration
	}
	return terraformutils.NewResource(
		ssoAdminPermissionSetResourceID(permissionSetARN, instanceARN),
		ssoAdminResourceName(StringValue(permissionSet.Name), permissionSetARN, instanceARN),
		ssoAdminPermissionSetResourceType,
		"aws",
		attributes,
		ssoAdminAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newSSOAdminAccountAssignmentResource(instanceARN, permissionSetARN, targetID string, assignment ssotypes.AccountAssignment) terraformutils.Resource {
	principalID := StringValue(assignment.PrincipalId)
	principalType := string(assignment.PrincipalType)
	targetType := string(ssotypes.TargetTypeAwsAccount)
	attributes := map[string]string{
		"instance_arn":       instanceARN,
		"permission_set_arn": permissionSetARN,
		"principal_id":       principalID,
		"principal_type":     principalType,
		"target_id":          targetID,
		"target_type":        targetType,
	}
	return terraformutils.NewResource(
		ssoAdminAccountAssignmentResourceID(principalID, principalType, targetID, targetType, permissionSetARN, instanceARN),
		ssoAdminAccountAssignmentResourceName(instanceARN, permissionSetARN, principalID, principalType, targetID, targetType),
		ssoAdminAccountAssignmentResourceType,
		"aws",
		attributes,
		ssoAdminAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newSSOAdminInstanceAccessControlAttributesResource(instanceARN string, config *ssotypes.InstanceAccessControlAttributeConfiguration) terraformutils.Resource {
	accessControlAttributes := ssoAdminAccessControlAttributes(config)
	attributes := map[string]string{
		"attribute.#":  strconv.Itoa(len(accessControlAttributes)),
		"instance_arn": instanceARN,
	}
	for i, attribute := range accessControlAttributes {
		attributePrefix := fmt.Sprintf("attribute.%d", i)
		valuePrefix := attributePrefix + ".value.0"
		sourcePrefix := valuePrefix + ".source"
		attributes[attributePrefix+".key"] = attribute.key
		attributes[attributePrefix+".value.#"] = "1"
		attributes[sourcePrefix+".#"] = strconv.Itoa(len(attribute.sources))
		for j, source := range attribute.sources {
			attributes[fmt.Sprintf("%s.%d", sourcePrefix, j)] = source
		}
	}
	return terraformutils.NewResource(
		ssoAdminInstanceAccessControlAttributesResourceID(instanceARN),
		ssoAdminInstanceAccessControlAttributesResourceName(instanceARN),
		ssoAdminInstanceAccessControlAttributesResourceType,
		"aws",
		attributes,
		ssoAdminAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newSSOAdminManagedPolicyAttachmentResource(instanceARN, permissionSetARN string, policy ssotypes.AttachedManagedPolicy) terraformutils.Resource {
	policyARN := StringValue(policy.Arn)
	attributes := map[string]string{
		"instance_arn":       instanceARN,
		"managed_policy_arn": policyARN,
		"permission_set_arn": permissionSetARN,
	}
	if policyName := StringValue(policy.Name); policyName != "" {
		attributes["managed_policy_name"] = policyName
	}
	return terraformutils.NewResource(
		ssoAdminManagedPolicyAttachmentResourceID(policyARN, permissionSetARN, instanceARN),
		ssoAdminResourceName(permissionSetARN, policyARN, instanceARN),
		ssoAdminManagedPolicyAttachmentResourceType,
		"aws",
		attributes,
		ssoAdminAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newSSOAdminCustomerManagedPolicyAttachmentResource(instanceARN, permissionSetARN string, policy ssotypes.CustomerManagedPolicyReference) terraformutils.Resource {
	policyName := StringValue(policy.Name)
	policyPath := ssoAdminCustomerManagedPolicyPath(policy.Path)
	attributes := map[string]string{
		"customer_managed_policy_reference.#":      "1",
		"customer_managed_policy_reference.0.name": policyName,
		"customer_managed_policy_reference.0.path": policyPath,
		"instance_arn":       instanceARN,
		"permission_set_arn": permissionSetARN,
	}
	return terraformutils.NewResource(
		ssoAdminCustomerManagedPolicyAttachmentResourceID(policyName, policyPath, permissionSetARN, instanceARN),
		ssoAdminResourceName(permissionSetARN, policyPath, policyName, instanceARN),
		ssoAdminCustomerManagedPolicyAttachmentResourceType,
		"aws",
		attributes,
		ssoAdminAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newSSOAdminPermissionSetInlinePolicyResource(instanceARN, permissionSetARN, inlinePolicy string) terraformutils.Resource {
	return terraformutils.NewResource(
		ssoAdminPermissionSetResourceID(permissionSetARN, instanceARN),
		ssoAdminResourceName(permissionSetARN, "inline-policy", instanceARN),
		ssoAdminPermissionSetInlinePolicyResourceType,
		"aws",
		map[string]string{
			"inline_policy":      inlinePolicy,
			"instance_arn":       instanceARN,
			"permission_set_arn": permissionSetARN,
		},
		ssoAdminAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newSSOAdminPermissionsBoundaryAttachmentResource(instanceARN, permissionSetARN string, boundary *ssotypes.PermissionsBoundary) terraformutils.Resource {
	attributes := map[string]string{
		"instance_arn":           instanceARN,
		"permission_set_arn":     permissionSetARN,
		"permissions_boundary.#": "1",
	}
	if managedPolicyARN := StringValue(boundary.ManagedPolicyArn); managedPolicyARN != "" {
		attributes[ssoAdminNestedPermissionsBoundaryAttributePrefix+".managed_policy_arn"] = managedPolicyARN
	} else if policy := boundary.CustomerManagedPolicyReference; policy != nil {
		attributes[ssoAdminNestedPermissionsBoundaryAttributePrefix+".customer_managed_policy_reference.#"] = "1"
		attributes[ssoAdminNestedCustomerManagedPolicyReferenceAttribute+".name"] = StringValue(policy.Name)
		attributes[ssoAdminNestedCustomerManagedPolicyReferenceAttribute+".path"] = ssoAdminCustomerManagedPolicyPath(policy.Path)
	}
	return terraformutils.NewResource(
		ssoAdminPermissionSetResourceID(permissionSetARN, instanceARN),
		ssoAdminResourceName(permissionSetARN, "permissions-boundary", instanceARN),
		ssoAdminPermissionsBoundaryAttachmentResourceType,
		"aws",
		attributes,
		ssoAdminAllowEmptyValues,
		map[string]interface{}{},
	)
}

func ssoAdminPermissionSetResourceID(permissionSetARN, instanceARN string) string {
	return strings.Join([]string{permissionSetARN, instanceARN}, ssoAdminResourceIDSeparator)
}

func ssoAdminAccountAssignmentResourceID(principalID, principalType, targetID, targetType, permissionSetARN, instanceARN string) string {
	return strings.Join([]string{principalID, principalType, targetID, targetType, permissionSetARN, instanceARN}, ssoAdminResourceIDSeparator)
}

func ssoAdminInstanceAccessControlAttributesResourceID(instanceARN string) string {
	return instanceARN
}

func ssoAdminManagedPolicyAttachmentResourceID(managedPolicyARN, permissionSetARN, instanceARN string) string {
	return strings.Join([]string{managedPolicyARN, permissionSetARN, instanceARN}, ssoAdminResourceIDSeparator)
}

func ssoAdminCustomerManagedPolicyAttachmentResourceID(policyName, policyPath, permissionSetARN, instanceARN string) string {
	return strings.Join([]string{policyName, policyPath, permissionSetARN, instanceARN}, ssoAdminResourceIDSeparator)
}

func ssoAdminAccountAssignmentResourceName(instanceARN, permissionSetARN, principalID, principalType, targetID, targetType string) string {
	return ssoAdminResourceName(permissionSetARN, targetID, targetType, principalType, principalID, instanceARN)
}

func ssoAdminInstanceAccessControlAttributesResourceName(instanceARN string) string {
	return ssoAdminResourceName("access-control-attributes", instanceARN)
}

func ssoAdminResourceName(parts ...string) string {
	var nonEmptyParts []string
	for _, part := range parts {
		if part != "" {
			nonEmptyParts = append(nonEmptyParts, part)
		}
	}
	return strings.Join(nonEmptyParts, ssoAdminResourceNameSeparator)
}

func (g *SSOAdminGenerator) updateManagedPolicyAttachmentDependencies() {
	accountAssignmentRefsByPermissionSet := ssoAdminAccountAssignmentRefsByPermissionSet(g.Resources)
	for i := range g.Resources {
		if ssoAdminResourceType(g.Resources[i]) != ssoAdminManagedPolicyAttachmentResourceType {
			continue
		}
		permissionSetARN := ssoAdminResourceAttribute(g.Resources[i], "permission_set_arn")
		ssoAdminSetDependsOn(&g.Resources[i], accountAssignmentRefsByPermissionSet[permissionSetARN])
	}
}

func ssoAdminAccountAssignmentRefsByPermissionSet(resources []terraformutils.Resource) map[string][]string {
	refsByPermissionSet := map[string]map[string]struct{}{}
	for _, resource := range resources {
		resourceRef := ssoAdminResourceRef(resource)
		if ssoAdminResourceType(resource) != ssoAdminAccountAssignmentResourceType || resourceRef == "" {
			continue
		}
		permissionSetARN := ssoAdminResourceAttribute(resource, "permission_set_arn")
		if permissionSetARN == "" {
			continue
		}
		if refsByPermissionSet[permissionSetARN] == nil {
			refsByPermissionSet[permissionSetARN] = map[string]struct{}{}
		}
		refsByPermissionSet[permissionSetARN][resourceRef] = struct{}{}
	}

	result := map[string][]string{}
	for permissionSetARN, refs := range refsByPermissionSet {
		for ref := range refs {
			result[permissionSetARN] = append(result[permissionSetARN], ref)
		}
		sort.Strings(result[permissionSetARN])
	}
	return result
}

func ssoAdminResourceType(resource terraformutils.Resource) string {
	if resource.InstanceInfo == nil {
		return ""
	}
	return resource.InstanceInfo.Type
}

func ssoAdminResourceRef(resource terraformutils.Resource) string {
	if resource.InstanceInfo == nil {
		return ""
	}
	return resource.InstanceInfo.Id
}

func ssoAdminResourceAttribute(resource terraformutils.Resource, key string) string {
	if value, ok := resource.Item[key].(string); ok && value != "" {
		return value
	}
	if resource.InstanceState == nil {
		return ""
	}
	return resource.InstanceState.Attributes[key]
}

func ssoAdminSetDependsOn(resource *terraformutils.Resource, dependsOn []string) {
	if len(dependsOn) == 0 {
		delete(resource.AdditionalFields, "depends_on")
		if resource.Item != nil {
			delete(resource.Item, "depends_on")
		}
		return
	}
	refs := append([]string(nil), dependsOn...)
	sort.Strings(refs)
	if resource.AdditionalFields == nil {
		resource.AdditionalFields = map[string]interface{}{}
	}
	resource.AdditionalFields["depends_on"] = refs
	if resource.Item != nil {
		resource.Item["depends_on"] = refs
	}
}

func ssoAdminPermissionsBoundaryConfigured(boundary *ssotypes.PermissionsBoundary) bool {
	if boundary == nil {
		return false
	}
	if StringValue(boundary.ManagedPolicyArn) != "" {
		return true
	}
	if boundary.CustomerManagedPolicyReference == nil {
		return false
	}
	return StringValue(boundary.CustomerManagedPolicyReference.Name) != ""
}

func ssoAdminAccountAssignmentConfigured(targetID string, assignment ssotypes.AccountAssignment) bool {
	return targetID != "" && StringValue(assignment.PrincipalId) != "" && string(assignment.PrincipalType) != ""
}

type ssoAdminAccessControlAttribute struct {
	key     string
	sources []string
}

func ssoAdminInstanceAccessControlAttributesConfigured(config *ssotypes.InstanceAccessControlAttributeConfiguration) bool {
	return len(ssoAdminAccessControlAttributes(config)) > 0
}

func ssoAdminAccessControlAttributes(config *ssotypes.InstanceAccessControlAttributeConfiguration) []ssoAdminAccessControlAttribute {
	if config == nil {
		return nil
	}
	attributes := make([]ssoAdminAccessControlAttribute, 0, len(config.AccessControlAttributes))
	for _, attribute := range config.AccessControlAttributes {
		key := StringValue(attribute.Key)
		if key == "" || attribute.Value == nil {
			continue
		}
		sources := make([]string, 0, len(attribute.Value.Source))
		for _, source := range attribute.Value.Source {
			if source != "" {
				sources = append(sources, source)
			}
		}
		if len(sources) == 0 {
			continue
		}
		sort.Strings(sources)
		attributes = append(attributes, ssoAdminAccessControlAttribute{key: key, sources: sources})
	}
	sort.Slice(attributes, func(i, j int) bool {
		if attributes[i].key != attributes[j].key {
			return attributes[i].key < attributes[j].key
		}
		return strings.Join(attributes[i].sources, "\x00") < strings.Join(attributes[j].sources, "\x00")
	})
	return attributes
}

func ssoAdminInstanceAccessControlAttributesNotConfigured(err error) bool {
	return ssoAdminResourceNotFound(err)
}

func ssoAdminEscapeInstanceAccessControlAttributeSources(item map[string]interface{}) {
	attributes, ok := item["attribute"].([]interface{})
	if !ok {
		return
	}
	for _, attribute := range attributes {
		attributeMap, ok := attribute.(map[string]interface{})
		if !ok {
			continue
		}
		values, ok := attributeMap["value"].([]interface{})
		if !ok {
			continue
		}
		for _, value := range values {
			valueMap, ok := value.(map[string]interface{})
			if !ok {
				continue
			}
			ssoAdminEscapeAccessControlAttributeSourceValues(valueMap)
		}
	}
}

func ssoAdminEscapeAccessControlAttributeSourceValues(valueMap map[string]interface{}) {
	switch sources := valueMap["source"].(type) {
	case []interface{}:
		for i, source := range sources {
			sourceString, ok := source.(string)
			if !ok || sourceString == "" {
				continue
			}
			sources[i] = ssoAdminEscapeTerraformTemplateMarkers(sourceString)
		}
	case []string:
		for i, source := range sources {
			if source == "" {
				continue
			}
			sources[i] = ssoAdminEscapeTerraformTemplateMarkers(source)
		}
	}
}

func ssoAdminEscapeTerraformTemplateMarkers(value string) string {
	value = ssoAdminEscapeTerraformTemplateMarker(value, "${", '$')
	return ssoAdminEscapeTerraformTemplateMarker(value, "%{", '%')
}

func ssoAdminEscapeTerraformTemplateMarker(value, marker string, escape byte) string {
	var escaped strings.Builder
	for i := 0; i < len(value); i++ {
		if strings.HasPrefix(value[i:], marker) && (i == 0 || value[i-1] != escape) {
			escaped.WriteByte(escape)
		}
		escaped.WriteByte(value[i])
	}
	return escaped.String()
}

func ssoAdminCustomerManagedPolicyPath(path *string) string {
	if p := StringValue(path); p != "" {
		return p
	}
	return ssoAdminDefaultCustomerManagedPolicyPath
}

func ssoAdminResourceNotFound(err error) bool {
	var notFound *ssotypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
