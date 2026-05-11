// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
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
		if err := g.loadPermissionSets(svc, instanceARN); err != nil {
			if ssoAdminResourceNotFound(err) {
				continue
			}
			return err
		}
	}
	return nil
}

func (g *SSOAdminGenerator) PostConvertHook() error {
	for i, resource := range g.Resources {
		if resource.InstanceInfo.Type != ssoAdminPermissionSetInlinePolicyResourceType {
			continue
		}
		inlinePolicy, ok := resource.Item["inline_policy"].(string)
		if !ok || inlinePolicy == "" {
			continue
		}
		g.Resources[i].Item["inline_policy"] = fmt.Sprintf("<<POLICY\n%s\nPOLICY", g.escapeAwsInterpolation(inlinePolicy))
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
	return g.loadPermissionsBoundaryAttachment(svc, instanceARN, permissionSetARN)
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

func ssoAdminManagedPolicyAttachmentResourceID(managedPolicyARN, permissionSetARN, instanceARN string) string {
	return strings.Join([]string{managedPolicyARN, permissionSetARN, instanceARN}, ssoAdminResourceIDSeparator)
}

func ssoAdminCustomerManagedPolicyAttachmentResourceID(policyName, policyPath, permissionSetARN, instanceARN string) string {
	return strings.Join([]string{policyName, policyPath, permissionSetARN, instanceARN}, ssoAdminResourceIDSeparator)
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
