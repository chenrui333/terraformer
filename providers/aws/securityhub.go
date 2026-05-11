// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"strings"

	awsarn "github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	securityhubtypes "github.com/aws/aws-sdk-go-v2/service/securityhub/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var securityhubAllowEmptyValues = []string{"tags."}

const (
	securityHubAccountResourceType                        = "aws_securityhub_account"
	securityHubActionTargetResourceType                   = "aws_securityhub_action_target"
	securityHubAutomationRuleResourceType                 = "aws_securityhub_automation_rule"
	securityHubConfigurationPolicyResourceType            = "aws_securityhub_configuration_policy"
	securityHubConfigurationPolicyAssociationResourceType = "aws_securityhub_configuration_policy_association"
	securityHubFindingAggregatorResourceType              = "aws_securityhub_finding_aggregator"
	securityHubInsightResourceType                        = "aws_securityhub_insight"
	securityHubMemberResourceType                         = "aws_securityhub_member"
	securityHubOrganizationAdminAccountResourceType       = "aws_securityhub_organization_admin_account"
	securityHubOrganizationConfigurationResourceType      = "aws_securityhub_organization_configuration"
	securityHubProductSubscriptionResourceType            = "aws_securityhub_product_subscription"
	securityHubStandardsSubscriptionResourceType          = "aws_securityhub_standards_subscription"
	securityHubProductSubscriptionIDSeparator             = ","
	securityHubProductSubscriptionResourcePrefix          = "product-subscription/"
	securityHubProductResourcePrefix                      = "product/"
	securityHubResourceNameSeparator                      = ":"
)

type SecurityhubGenerator struct {
	AWSService
}

func (g *SecurityhubGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	client := securityhub.NewFromConfig(config)

	account, err := g.getAccountNumber(config)
	if err != nil {
		return err
	}

	accountNumber := *account
	accountDisabled, err := g.addAccount(client, accountNumber)
	if accountDisabled {
		return nil
	}
	if err != nil {
		return err
	}
	if err = g.addMembers(client, accountNumber); err != nil {
		return err
	}
	if err = g.addStandardsSubscription(client, accountNumber); err != nil {
		return err
	}
	for _, loader := range []func(*SecurityhubGenerator, *securityhub.Client, string) error{
		(*SecurityhubGenerator).addActionTargets,
		(*SecurityhubGenerator).addProductSubscriptions,
		(*SecurityhubGenerator).addInsights,
		(*SecurityhubGenerator).addFindingAggregators,
		(*SecurityhubGenerator).addOrganizationAdminAccounts,
		(*SecurityhubGenerator).addOrganizationConfiguration,
		(*SecurityhubGenerator).addConfigurationPolicies,
		(*SecurityhubGenerator).addConfigurationPolicyAssociations,
		(*SecurityhubGenerator).addAutomationRules,
	} {
		if err := loader(g, client, accountNumber); err != nil {
			if securityHubOptionalResourceUnavailable(err) {
				continue
			}
			return err
		}
	}
	return nil
}

func (g *SecurityhubGenerator) addAccount(client *securityhub.Client, accountNumber string) (bool, error) {
	_, err := client.GetEnabledStandards(context.TODO(), &securityhub.GetEnabledStandardsInput{})

	if err != nil {
		errorMsg := err.Error()
		if !strings.Contains(errorMsg, "not subscribed to AWS Security Hub") {
			return false, err
		}
		return true, nil
	}
	g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
		accountNumber,
		accountNumber,
		securityHubAccountResourceType,
		"aws",
		securityhubAllowEmptyValues,
	))
	return false, nil
}

func (g *SecurityhubGenerator) addMembers(svc *securityhub.Client, accountNumber string) error {
	p := securityhub.NewListMembersPaginator(svc, &securityhub.ListMembersInput{})

	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if securityHubOptionalResourceUnavailable(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, member := range page.Members {
			id := StringValue(member.AccountId)
			if id == "" {
				continue
			}
			attributes := map[string]string{
				"account_id": id,
			}
			if member.Email != nil {
				attributes["email"] = *member.Email
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				id,
				securityHubResourceName("member", id),
				securityHubMemberResourceType,
				"aws",
				attributes,
				securityhubAllowEmptyValues,
				securityHubAccountDependency(accountNumber),
			))
		}
	}
	return nil
}

func (g *SecurityhubGenerator) addStandardsSubscription(svc *securityhub.Client, accountNumber string) error {
	p := securityhub.NewGetEnabledStandardsPaginator(svc, &securityhub.GetEnabledStandardsInput{})

	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, standardsSubscription := range page.StandardsSubscriptions {
			resource, ok := newSecurityHubStandardsSubscriptionResource(standardsSubscription, accountNumber)
			if !ok {
				continue
			}
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SecurityhubGenerator) addActionTargets(svc *securityhub.Client, accountNumber string) error {
	p := securityhub.NewDescribeActionTargetsPaginator(svc, &securityhub.DescribeActionTargetsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, actionTarget := range page.ActionTargets {
			resource, ok := newSecurityHubActionTargetResource(actionTarget, accountNumber)
			if !ok {
				continue
			}
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SecurityhubGenerator) addProductSubscriptions(svc *securityhub.Client, accountNumber string) error {
	productARNBySuffix, err := securityHubProductARNsBySubscriptionSuffix(svc)
	if err != nil {
		return err
	}
	p := securityhub.NewListEnabledProductsForImportPaginator(svc, &securityhub.ListEnabledProductsForImportInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, productSubscriptionARN := range page.ProductSubscriptions {
			suffix, ok := securityHubProductSubscriptionSuffix(productSubscriptionARN)
			if !ok {
				continue
			}
			resource, ok := newSecurityHubProductSubscriptionResource(productSubscriptionARN, productARNBySuffix[suffix], accountNumber)
			if !ok {
				continue
			}
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SecurityhubGenerator) addInsights(svc *securityhub.Client, accountNumber string) error {
	p := securityhub.NewGetInsightsPaginator(svc, &securityhub.GetInsightsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, insight := range page.Insights {
			resource, ok := newSecurityHubInsightResource(insight, accountNumber)
			if !ok {
				continue
			}
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SecurityhubGenerator) addFindingAggregators(svc *securityhub.Client, accountNumber string) error {
	p := securityhub.NewListFindingAggregatorsPaginator(svc, &securityhub.ListFindingAggregatorsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, findingAggregator := range page.FindingAggregators {
			arn := StringValue(findingAggregator.FindingAggregatorArn)
			if arn == "" {
				continue
			}
			g.Resources = append(g.Resources, newSecurityHubSimpleResource(
				arn,
				securityHubResourceName("finding_aggregator", arn),
				securityHubFindingAggregatorResourceType,
				accountNumber,
			))
		}
	}
	return nil
}

func (g *SecurityhubGenerator) addOrganizationAdminAccounts(svc *securityhub.Client, accountNumber string) error {
	p := securityhub.NewListOrganizationAdminAccountsPaginator(svc, &securityhub.ListOrganizationAdminAccountsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, adminAccount := range page.AdminAccounts {
			if adminAccount.Status != securityhubtypes.AdminStatusEnabled {
				continue
			}
			accountID := StringValue(adminAccount.AccountId)
			if accountID == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				accountID,
				securityHubResourceName("organization_admin_account", accountID),
				securityHubOrganizationAdminAccountResourceType,
				"aws",
				map[string]string{"admin_account_id": accountID},
				securityhubAllowEmptyValues,
				securityHubAccountDependency(accountNumber),
			))
		}
	}
	return nil
}

func (g *SecurityhubGenerator) addOrganizationConfiguration(svc *securityhub.Client, accountNumber string) error {
	_, err := svc.DescribeOrganizationConfiguration(context.TODO(), &securityhub.DescribeOrganizationConfigurationInput{})
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, newSecurityHubSimpleResource(
		accountNumber,
		securityHubResourceName("organization_configuration", accountNumber),
		securityHubOrganizationConfigurationResourceType,
		accountNumber,
	))
	return nil
}

func (g *SecurityhubGenerator) addConfigurationPolicies(svc *securityhub.Client, accountNumber string) error {
	p := securityhub.NewListConfigurationPoliciesPaginator(svc, &securityhub.ListConfigurationPoliciesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, policy := range page.ConfigurationPolicySummaries {
			id := StringValue(policy.Id)
			if id == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				id,
				securityHubResourceName("configuration_policy", id),
				securityHubConfigurationPolicyResourceType,
				"aws",
				map[string]string{"id": id},
				securityhubAllowEmptyValues,
				securityHubAccountDependency(accountNumber),
			))
		}
	}
	return nil
}

func (g *SecurityhubGenerator) addConfigurationPolicyAssociations(svc *securityhub.Client, accountNumber string) error {
	p := securityhub.NewListConfigurationPolicyAssociationsPaginator(svc, &securityhub.ListConfigurationPolicyAssociationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, association := range page.ConfigurationPolicyAssociationSummaries {
			resource, ok := newSecurityHubConfigurationPolicyAssociationResource(association, accountNumber)
			if !ok {
				continue
			}
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SecurityhubGenerator) addAutomationRules(svc *securityhub.Client, accountNumber string) error {
	var nextToken *string
	for {
		page, err := svc.ListAutomationRules(context.TODO(), &securityhub.ListAutomationRulesInput{
			NextToken: nextToken,
		})
		if err != nil {
			return err
		}
		for _, rule := range page.AutomationRulesMetadata {
			arn := StringValue(rule.RuleArn)
			if arn == "" {
				continue
			}
			g.Resources = append(g.Resources, newSecurityHubSimpleResource(
				arn,
				securityHubResourceName("automation_rule", arn),
				securityHubAutomationRuleResourceType,
				accountNumber,
			))
		}
		nextToken = page.NextToken
		if !awsHasMorePages(nextToken) {
			break
		}
	}
	return nil
}

func newSecurityHubStandardsSubscriptionResource(standardsSubscription securityhubtypes.StandardsSubscription, accountNumber string) (terraformutils.Resource, bool) {
	id := StringValue(standardsSubscription.StandardsSubscriptionArn)
	standardsARN := StringValue(standardsSubscription.StandardsArn)
	if id == "" || standardsARN == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		id,
		securityHubResourceName("standards_subscription", id),
		securityHubStandardsSubscriptionResourceType,
		"aws",
		map[string]string{
			"standards_arn": standardsARN,
		},
		securityhubAllowEmptyValues,
		securityHubAccountDependency(accountNumber),
	), true
}

func newSecurityHubActionTargetResource(actionTarget securityhubtypes.ActionTarget, accountNumber string) (terraformutils.Resource, bool) {
	arn := StringValue(actionTarget.ActionTargetArn)
	identifier := securityHubActionTargetIdentifier(arn)
	if arn == "" || identifier == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		arn,
		securityHubResourceName("action_target", arn),
		securityHubActionTargetResourceType,
		"aws",
		map[string]string{
			"description": StringValue(actionTarget.Description),
			"identifier":  identifier,
			"name":        StringValue(actionTarget.Name),
		},
		securityhubAllowEmptyValues,
		securityHubAccountDependency(accountNumber),
	), true
}

func newSecurityHubProductSubscriptionResource(productSubscriptionARN, productARN, accountNumber string) (terraformutils.Resource, bool) {
	if productSubscriptionARN == "" || productARN == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		securityHubProductSubscriptionResourceID(productARN, productSubscriptionARN),
		securityHubResourceName("product_subscription", productSubscriptionARN),
		securityHubProductSubscriptionResourceType,
		"aws",
		map[string]string{
			"arn":         productSubscriptionARN,
			"product_arn": productARN,
		},
		securityhubAllowEmptyValues,
		securityHubAccountDependency(accountNumber),
	), true
}

func newSecurityHubInsightResource(insight securityhubtypes.Insight, accountNumber string) (terraformutils.Resource, bool) {
	arn := StringValue(insight.InsightArn)
	if arn == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		arn,
		securityHubResourceName("insight", arn),
		securityHubInsightResourceType,
		"aws",
		map[string]string{
			"group_by_attribute": StringValue(insight.GroupByAttribute),
			"name":               StringValue(insight.Name),
		},
		securityhubAllowEmptyValues,
		securityHubAccountDependency(accountNumber),
	), true
}

func newSecurityHubConfigurationPolicyAssociationResource(association securityhubtypes.ConfigurationPolicyAssociationSummary, accountNumber string) (terraformutils.Resource, bool) {
	targetID := StringValue(association.TargetId)
	policyID := StringValue(association.ConfigurationPolicyId)
	if targetID == "" || policyID == "" ||
		association.AssociationType != securityhubtypes.AssociationTypeApplied ||
		!securityHubConfigurationPolicyAssociationImportable(association.AssociationStatus) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		targetID,
		securityHubResourceName("configuration_policy_association", targetID),
		securityHubConfigurationPolicyAssociationResourceType,
		"aws",
		map[string]string{
			"policy_id": policyID,
			"target_id": targetID,
		},
		securityhubAllowEmptyValues,
		securityHubAccountDependency(accountNumber),
	), true
}

func securityHubConfigurationPolicyAssociationImportable(status securityhubtypes.ConfigurationPolicyAssociationStatus) bool {
	return status == securityhubtypes.ConfigurationPolicyAssociationStatusSuccess ||
		status == securityhubtypes.ConfigurationPolicyAssociationStatusPending
}

func newSecurityHubSimpleResource(id, name, resourceType, accountNumber string) terraformutils.Resource {
	return terraformutils.NewResource(
		id,
		name,
		resourceType,
		"aws",
		map[string]string{},
		securityhubAllowEmptyValues,
		securityHubAccountDependency(accountNumber),
	)
}

func securityHubAccountDependency(accountNumber string) map[string]interface{} {
	if accountNumber == "" {
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		"depends_on": []string{securityHubAccountResourceRef(accountNumber)},
	}
}

func securityHubAccountResourceRef(accountNumber string) string {
	return securityHubAccountResourceType + "." + terraformutils.TfSanitize(accountNumber)
}

func securityHubActionTargetIdentifier(actionTargetARN string) string {
	parts := strings.Split(actionTargetARN, "/")
	if len(parts) != 3 {
		return ""
	}
	return parts[2]
}

func securityHubProductSubscriptionResourceID(productARN, productSubscriptionARN string) string {
	return strings.Join([]string{productARN, productSubscriptionARN}, securityHubProductSubscriptionIDSeparator)
}

func securityHubProductARNsBySubscriptionSuffix(svc *securityhub.Client) (map[string]string, error) {
	p := securityhub.NewDescribeProductsPaginator(svc, &securityhub.DescribeProductsInput{})
	productsBySuffix := map[string]string{}
	ambiguousSuffixes := map[string]bool{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		securityHubAddProductARNsBySuffix(productsBySuffix, ambiguousSuffixes, page.Products)
	}
	for suffix := range ambiguousSuffixes {
		delete(productsBySuffix, suffix)
	}
	return productsBySuffix, nil
}

func securityHubAddProductARNsBySuffix(productsBySuffix map[string]string, ambiguousSuffixes map[string]bool, products []securityhubtypes.Product) {
	for _, product := range products {
		productARN := StringValue(product.ProductArn)
		suffix, ok := securityHubProductARNSuffix(productARN)
		if !ok {
			continue
		}
		if existing, exists := productsBySuffix[suffix]; exists && existing != productARN {
			ambiguousSuffixes[suffix] = true
			continue
		}
		productsBySuffix[suffix] = productARN
	}
}

func securityHubProductARNSuffix(productARN string) (string, bool) {
	return securityHubARNResourceSuffix(productARN, securityHubProductResourcePrefix)
}

func securityHubProductSubscriptionSuffix(productSubscriptionARN string) (string, bool) {
	return securityHubARNResourceSuffix(productSubscriptionARN, securityHubProductSubscriptionResourcePrefix)
}

func securityHubARNResourceSuffix(value, prefix string) (string, bool) {
	parsed, err := awsarn.Parse(value)
	if err != nil {
		return "", false
	}
	if !strings.HasPrefix(parsed.Resource, prefix) {
		return "", false
	}
	suffix := strings.TrimPrefix(parsed.Resource, prefix)
	if suffix == "" {
		return "", false
	}
	return suffix, true
}

func securityHubResourceName(parts ...string) string {
	return strings.Join(parts, securityHubResourceNameSeparator)
}

func securityHubOptionalResourceUnavailable(err error) bool {
	if err == nil {
		return false
	}
	var resourceNotFound *securityhubtypes.ResourceNotFoundException
	if errors.As(err, &resourceNotFound) {
		return true
	}
	var accessDenied *securityhubtypes.AccessDeniedException
	if errors.As(err, &accessDenied) {
		return securityHubSkippableAccessMessage(accessDenied.ErrorMessage())
	}
	var invalidAccess *securityhubtypes.InvalidAccessException
	if errors.As(err, &invalidAccess) {
		return securityHubSkippableAccessMessage(invalidAccess.ErrorMessage())
	}
	return false
}

func securityHubSkippableAccessMessage(message string) bool {
	message = strings.ToLower(message)
	return strings.Contains(message, "not subscribed to aws security hub") ||
		strings.Contains(message, "no such resource found") ||
		strings.Contains(message, "not a member of an organization") ||
		strings.Contains(message, "delegated administrator") ||
		strings.Contains(message, "central configuration")
}
