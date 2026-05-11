// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	securityhubtypes "github.com/aws/aws-sdk-go-v2/service/securityhub/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const securityHubTestAccountID = "123456789012"

func TestSecurityHubProductSubscriptionResourceID(t *testing.T) {
	productARN := "arn:aws:securityhub:us-east-1::product/aws/guardduty"
	subscriptionARN := "arn:aws:securityhub:us-east-1:123456789012:product-subscription/aws/guardduty"
	got := securityHubProductSubscriptionResourceID(productARN, subscriptionARN)
	want := productARN + "," + subscriptionARN
	if got != want {
		t.Fatalf("securityHubProductSubscriptionResourceID() = %q, want %q", got, want)
	}
}

func TestSecurityHubProductARNAndSubscriptionSuffixes(t *testing.T) {
	tests := []struct {
		name string
		arn  string
		want string
		ok   bool
	}{
		{
			name: "subscription suffix",
			arn:  "arn:aws:securityhub:us-west-2:123456789012:product-subscription/alertlogic/althreatmanagement",
			want: "alertlogic/althreatmanagement",
			ok:   true,
		},
		{
			name: "invalid arn",
			arn:  "not-an-arn",
			ok:   false,
		},
		{
			name: "not product subscription",
			arn:  "arn:aws:securityhub:us-west-2:123456789012:action/custom/example",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := securityHubProductSubscriptionSuffix(tt.arn)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("securityHubProductSubscriptionSuffix() = (%q, %t), want (%q, %t)", got, ok, tt.want, tt.ok)
			}
		})
	}

	productSuffix, ok := securityHubProductARNSuffix("arn:aws:securityhub:us-west-2:999999999999:product/alertlogic/althreatmanagement")
	if !ok || productSuffix != "alertlogic/althreatmanagement" {
		t.Fatalf("securityHubProductARNSuffix() = (%q, %t), want (alertlogic/althreatmanagement, true)", productSuffix, ok)
	}
}

func TestSecurityHubAddProductARNsBySuffix(t *testing.T) {
	productsBySuffix := map[string]string{}
	ambiguousSuffixes := map[string]bool{}
	securityHubAddProductARNsBySuffix(productsBySuffix, ambiguousSuffixes, []securityhubtypes.Product{
		{ProductArn: aws.String("arn:aws:securityhub:us-east-1::product/aws/guardduty")},
		{ProductArn: aws.String("arn:aws:securityhub:us-east-1:999999999999:product/alertlogic/althreatmanagement")},
		{ProductArn: aws.String("not-an-arn")},
	})

	if got := productsBySuffix["aws/guardduty"]; got != "arn:aws:securityhub:us-east-1::product/aws/guardduty" {
		t.Fatalf("aws product ARN = %q", got)
	}
	if got := productsBySuffix["alertlogic/althreatmanagement"]; got != "arn:aws:securityhub:us-east-1:999999999999:product/alertlogic/althreatmanagement" {
		t.Fatalf("partner product ARN = %q", got)
	}

	securityHubAddProductARNsBySuffix(productsBySuffix, ambiguousSuffixes, []securityhubtypes.Product{
		{ProductArn: aws.String("arn:aws:securityhub:us-east-1:888888888888:product/alertlogic/althreatmanagement")},
	})
	for suffix := range ambiguousSuffixes {
		delete(productsBySuffix, suffix)
	}
	if _, ok := productsBySuffix["alertlogic/althreatmanagement"]; ok {
		t.Fatal("expected ambiguous partner product suffix to be removed")
	}
}

func TestSecurityHubActionTargetIdentifier(t *testing.T) {
	got := securityHubActionTargetIdentifier("arn:aws:securityhub:us-east-1:123456789012:action/custom/SendToChat")
	want := "SendToChat"
	if got != want {
		t.Fatalf("securityHubActionTargetIdentifier() = %q, want %q", got, want)
	}
	if got := securityHubActionTargetIdentifier("arn:aws:securityhub:us-east-1:123456789012:action/custom"); got != "" {
		t.Fatalf("securityHubActionTargetIdentifier() = %q, want empty", got)
	}
}

func TestSecurityHubStandardsSubscriptionResource(t *testing.T) {
	subscriptionARN := "arn:aws:securityhub:us-east-1:123456789012:subscription/cis-aws-foundations-benchmark/v/1.2.0"
	standardsARN := "arn:aws:securityhub:us-east-1::standards/cis-aws-foundations-benchmark/v/1.2.0"
	resource, ok := newSecurityHubStandardsSubscriptionResource(securityhubtypes.StandardsSubscription{
		StandardsArn:             aws.String(standardsARN),
		StandardsSubscriptionArn: aws.String(subscriptionARN),
	}, securityHubTestAccountID)
	if !ok {
		t.Fatal("expected standards subscription resource")
	}
	if got := resource.InstanceInfo.Type; got != securityHubStandardsSubscriptionResourceType {
		t.Fatalf("resource type = %q, want %q", got, securityHubStandardsSubscriptionResourceType)
	}
	if got := resource.InstanceState.ID; got != subscriptionARN {
		t.Fatalf("resource ID = %q, want %q", got, subscriptionARN)
	}
	if got := resource.InstanceState.Attributes["standards_arn"]; got != standardsARN {
		t.Fatalf("standards_arn = %q, want %q", got, standardsARN)
	}
	assertSecurityHubAccountDependency(t, resource)
}

func TestSecurityHubActionTargetResource(t *testing.T) {
	resource, ok := newSecurityHubActionTargetResource(securityhubtypes.ActionTarget{
		ActionTargetArn: aws.String("arn:aws:securityhub:us-east-1:123456789012:action/custom/SendToChat"),
		Description:     aws.String("Send findings to chat"),
		Name:            aws.String("SendToChat"),
	}, securityHubTestAccountID)
	if !ok {
		t.Fatal("expected action target resource")
	}
	if got := resource.InstanceInfo.Type; got != securityHubActionTargetResourceType {
		t.Fatalf("resource type = %q, want %q", got, securityHubActionTargetResourceType)
	}
	if got := resource.InstanceState.ID; got != "arn:aws:securityhub:us-east-1:123456789012:action/custom/SendToChat" {
		t.Fatalf("resource ID = %q", got)
	}
	if got := resource.InstanceState.Attributes["identifier"]; got != "SendToChat" {
		t.Fatalf("identifier = %q, want SendToChat", got)
	}
	assertSecurityHubAccountDependency(t, resource)
}

func TestSecurityHubProductSubscriptionResource(t *testing.T) {
	resource, ok := newSecurityHubProductSubscriptionResource(
		"arn:aws:securityhub:us-east-1:123456789012:product-subscription/alertlogic/althreatmanagement",
		"arn:aws:securityhub:us-east-1:999999999999:product/alertlogic/althreatmanagement",
		securityHubTestAccountID,
	)
	if !ok {
		t.Fatal("expected product subscription resource")
	}
	if got := resource.InstanceInfo.Type; got != securityHubProductSubscriptionResourceType {
		t.Fatalf("resource type = %q, want %q", got, securityHubProductSubscriptionResourceType)
	}
	wantProductARN := "arn:aws:securityhub:us-east-1:999999999999:product/alertlogic/althreatmanagement"
	if got := resource.InstanceState.Attributes["product_arn"]; got != wantProductARN {
		t.Fatalf("product_arn = %q, want %q", got, wantProductARN)
	}
	wantID := wantProductARN + ",arn:aws:securityhub:us-east-1:123456789012:product-subscription/alertlogic/althreatmanagement"
	if got := resource.InstanceState.ID; got != wantID {
		t.Fatalf("resource ID = %q, want %q", got, wantID)
	}
	assertSecurityHubAccountDependency(t, resource)
}

func TestSecurityHubInsightResource(t *testing.T) {
	resource, ok := newSecurityHubInsightResource(securityhubtypes.Insight{
		GroupByAttribute: aws.String("ResourceId"),
		InsightArn:       aws.String("arn:aws:securityhub:us-east-1:123456789012:insight/123456789012/custom/example"),
		Name:             aws.String("Grouped resources"),
	}, securityHubTestAccountID)
	if !ok {
		t.Fatal("expected insight resource")
	}
	if got := resource.InstanceInfo.Type; got != securityHubInsightResourceType {
		t.Fatalf("resource type = %q, want %q", got, securityHubInsightResourceType)
	}
	if got := resource.InstanceState.Attributes["group_by_attribute"]; got != "ResourceId" {
		t.Fatalf("group_by_attribute = %q, want ResourceId", got)
	}
	assertSecurityHubAccountDependency(t, resource)
}

func TestSecurityHubConfigurationPolicyAssociationResource(t *testing.T) {
	for _, status := range []securityhubtypes.ConfigurationPolicyAssociationStatus{
		securityhubtypes.ConfigurationPolicyAssociationStatusSuccess,
		securityhubtypes.ConfigurationPolicyAssociationStatusPending,
	} {
		t.Run(string(status), func(t *testing.T) {
			resource, ok := newSecurityHubConfigurationPolicyAssociationResource(securityhubtypes.ConfigurationPolicyAssociationSummary{
				AssociationStatus:     status,
				AssociationType:       securityhubtypes.AssociationTypeApplied,
				ConfigurationPolicyId: aws.String("00000000-1111-2222-3333-444444444444"),
				TargetId:              aws.String("123456789012"),
			}, securityHubTestAccountID)
			if !ok {
				t.Fatal("expected configuration policy association resource")
			}
			if got := resource.InstanceInfo.Type; got != securityHubConfigurationPolicyAssociationResourceType {
				t.Fatalf("resource type = %q, want %q", got, securityHubConfigurationPolicyAssociationResourceType)
			}
			if got := resource.InstanceState.Attributes["policy_id"]; got != "00000000-1111-2222-3333-444444444444" {
				t.Fatalf("policy_id = %q", got)
			}
			if got := resource.InstanceState.Attributes["target_id"]; got != "123456789012" {
				t.Fatalf("target_id = %q", got)
			}
			assertSecurityHubAccountDependency(t, resource)
		})
	}
}

func TestSecurityHubEmptyIdentifierSkips(t *testing.T) {
	if _, ok := newSecurityHubStandardsSubscriptionResource(securityhubtypes.StandardsSubscription{
		StandardsArn: aws.String("arn:aws:securityhub:us-east-1::standards/cis-aws-foundations-benchmark/v/1.2.0"),
	}, securityHubTestAccountID); ok {
		t.Fatal("expected empty standards subscription ARN to skip")
	}
	if _, ok := newSecurityHubStandardsSubscriptionResource(securityhubtypes.StandardsSubscription{
		StandardsSubscriptionArn: aws.String("arn:aws:securityhub:us-east-1:123456789012:subscription/cis-aws-foundations-benchmark/v/1.2.0"),
	}, securityHubTestAccountID); ok {
		t.Fatal("expected empty standards ARN to skip")
	}
	if _, ok := newSecurityHubProductSubscriptionResource("", "arn:aws:securityhub:us-east-1::product/aws/guardduty", securityHubTestAccountID); ok {
		t.Fatal("expected empty product subscription ARN to skip")
	}
	if _, ok := newSecurityHubProductSubscriptionResource("arn:aws:securityhub:us-east-1:123456789012:product-subscription/aws/guardduty", "", securityHubTestAccountID); ok {
		t.Fatal("expected empty product ARN to skip")
	}
	if _, ok := newSecurityHubInsightResource(securityhubtypes.Insight{}, securityHubTestAccountID); ok {
		t.Fatal("expected empty insight ARN to skip")
	}
	if _, ok := newSecurityHubConfigurationPolicyAssociationResource(securityhubtypes.ConfigurationPolicyAssociationSummary{
		AssociationStatus:     securityhubtypes.ConfigurationPolicyAssociationStatusSuccess,
		AssociationType:       securityhubtypes.AssociationTypeApplied,
		ConfigurationPolicyId: aws.String("policy-id"),
	}, securityHubTestAccountID); ok {
		t.Fatal("expected missing target ID to skip")
	}
	if _, ok := newSecurityHubConfigurationPolicyAssociationResource(securityhubtypes.ConfigurationPolicyAssociationSummary{
		AssociationStatus:     securityhubtypes.ConfigurationPolicyAssociationStatusSuccess,
		AssociationType:       securityhubtypes.AssociationTypeInherited,
		ConfigurationPolicyId: aws.String("policy-id"),
		TargetId:              aws.String("123456789012"),
	}, securityHubTestAccountID); ok {
		t.Fatal("expected inherited association to skip")
	}
	if _, ok := newSecurityHubConfigurationPolicyAssociationResource(securityhubtypes.ConfigurationPolicyAssociationSummary{
		AssociationStatus:     securityhubtypes.ConfigurationPolicyAssociationStatusFailed,
		AssociationType:       securityhubtypes.AssociationTypeApplied,
		ConfigurationPolicyId: aws.String("policy-id"),
		TargetId:              aws.String("123456789012"),
	}, securityHubTestAccountID); ok {
		t.Fatal("expected failed association to skip")
	}
}

func TestSecurityHubAccountDependency(t *testing.T) {
	got := securityHubAccountDependency(securityHubTestAccountID)
	dependsOn, ok := got["depends_on"].([]string)
	if !ok || len(dependsOn) != 1 {
		t.Fatalf("depends_on = %#v", got["depends_on"])
	}
	want := terraformutils.NewSimpleResource(
		securityHubTestAccountID,
		securityHubTestAccountID,
		securityHubAccountResourceType,
		"aws",
		securityhubAllowEmptyValues,
	).InstanceInfo.Id
	if dependsOn[0] != want {
		t.Fatalf("depends_on[0] = %q, want %q", dependsOn[0], want)
	}
	if empty := securityHubAccountDependency(""); len(empty) != 0 {
		t.Fatalf("empty dependency = %#v, want empty", empty)
	}
}

func TestSecurityHubOptionalResourceUnavailable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "resource not found",
			err:  &securityhubtypes.ResourceNotFoundException{Message: aws.String("not found")},
			want: true,
		},
		{
			name: "generic access denied",
			err:  &securityhubtypes.AccessDeniedException{Message: aws.String("denied")},
			want: false,
		},
		{
			name: "access denied for central configuration",
			err:  &securityhubtypes.AccessDeniedException{Message: aws.String("Must be a Security Hub delegated administrator with Central Configuration enabled")},
			want: true,
		},
		{
			name: "not subscribed",
			err:  &securityhubtypes.InvalidAccessException{Message: aws.String("not subscribed to AWS Security Hub")},
			want: true,
		},
		{
			name: "central configuration",
			err:  &securityhubtypes.InvalidAccessException{Message: aws.String("Must be a Security Hub delegated administrator with Central Configuration enabled")},
			want: true,
		},
		{
			name: "other invalid access",
			err:  &securityhubtypes.InvalidAccessException{Message: aws.String("invalid request")},
			want: false,
		},
		{
			name: "other error",
			err:  errors.New("boom"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := securityHubOptionalResourceUnavailable(tt.err); got != tt.want {
				t.Fatalf("securityHubOptionalResourceUnavailable() = %t, want %t", got, tt.want)
			}
		})
	}
}

func assertSecurityHubAccountDependency(t *testing.T, resource terraformutils.Resource) {
	t.Helper()

	dependsOn, ok := resource.AdditionalFields["depends_on"].([]string)
	if !ok || len(dependsOn) != 1 {
		t.Fatalf("depends_on = %#v", resource.AdditionalFields["depends_on"])
	}
	want := securityHubAccountResourceRef(securityHubTestAccountID)
	if dependsOn[0] != want {
		t.Fatalf("depends_on[0] = %q, want %q", dependsOn[0], want)
	}
}
