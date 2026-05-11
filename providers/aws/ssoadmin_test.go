// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ssotypes "github.com/aws/aws-sdk-go-v2/service/ssoadmin/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	testSSOAdminInstanceARN      = "arn:aws:sso:::instance/ssoins-1234567890abcdef"
	testSSOAdminPermissionSetARN = "arn:aws:sso:::permissionSet/ssoins-1234567890abcdef/ps-1234567890abcdef"
	testSSOAdminPolicyARN        = "arn:aws:iam::aws:policy/ReadOnlyAccess"
	testSSOAdminAccountID        = "123456789012"
	testSSOAdminPrincipalID      = "1234567890-11111111-2222-3333-4444-555555555555"
)

func TestSSOAdminResourceIDs(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "permission set",
			got:  ssoAdminPermissionSetResourceID(testSSOAdminPermissionSetARN, testSSOAdminInstanceARN),
			want: testSSOAdminPermissionSetARN + "," + testSSOAdminInstanceARN,
		},
		{
			name: "managed policy attachment",
			got:  ssoAdminManagedPolicyAttachmentResourceID(testSSOAdminPolicyARN, testSSOAdminPermissionSetARN, testSSOAdminInstanceARN),
			want: testSSOAdminPolicyARN + "," + testSSOAdminPermissionSetARN + "," + testSSOAdminInstanceARN,
		},
		{
			name: "customer managed policy attachment",
			got:  ssoAdminCustomerManagedPolicyAttachmentResourceID("Boundary", "/service/", testSSOAdminPermissionSetARN, testSSOAdminInstanceARN),
			want: "Boundary,/service/," + testSSOAdminPermissionSetARN + "," + testSSOAdminInstanceARN,
		},
		{
			name: "account assignment",
			got: ssoAdminAccountAssignmentResourceID(
				testSSOAdminPrincipalID,
				string(ssotypes.PrincipalTypeUser),
				testSSOAdminAccountID,
				string(ssotypes.TargetTypeAwsAccount),
				testSSOAdminPermissionSetARN,
				testSSOAdminInstanceARN,
			),
			want: testSSOAdminPrincipalID + ",USER," + testSSOAdminAccountID + ",AWS_ACCOUNT," + testSSOAdminPermissionSetARN + "," + testSSOAdminInstanceARN,
		},
		{
			name: "instance access control attributes",
			got:  ssoAdminInstanceAccessControlAttributesResourceID(testSSOAdminInstanceARN),
			want: testSSOAdminInstanceARN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("resource ID = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestSSOAdminInstanceAccessControlAttributesResource(t *testing.T) {
	resource := newSSOAdminInstanceAccessControlAttributesResource(
		testSSOAdminInstanceARN,
		&ssotypes.InstanceAccessControlAttributeConfiguration{
			AccessControlAttributes: []ssotypes.AccessControlAttribute{
				{
					Key: aws.String("name"),
					Value: &ssotypes.AccessControlAttributeValue{
						Source: []string{"${path:name.preferredName}", "${path:name.givenName}"},
					},
				},
				{
					Key: aws.String("last"),
					Value: &ssotypes.AccessControlAttributeValue{
						Source: []string{"${path:name.familyName}"},
					},
				},
			},
		},
	)

	if got, want := resource.InstanceState.ID, testSSOAdminInstanceARN; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceInfo.Type, ssoAdminInstanceAccessControlAttributesResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"attribute.#":                  "2",
		"attribute.0.key":              "last",
		"attribute.0.value.#":          "1",
		"attribute.0.value.0.source.#": "1",
		"attribute.0.value.0.source.0": "${path:name.familyName}",
		"attribute.1.key":              "name",
		"attribute.1.value.#":          "1",
		"attribute.1.value.0.source.#": "2",
		"attribute.1.value.0.source.0": "${path:name.givenName}",
		"attribute.1.value.0.source.1": "${path:name.preferredName}",
		"instance_arn":                 testSSOAdminInstanceARN,
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestSSOAdminAccountAssignmentResource(t *testing.T) {
	resource := newSSOAdminAccountAssignmentResource(
		testSSOAdminInstanceARN,
		testSSOAdminPermissionSetARN,
		testSSOAdminAccountID,
		ssotypes.AccountAssignment{
			AccountId:        aws.String(testSSOAdminAccountID),
			PermissionSetArn: aws.String(testSSOAdminPermissionSetARN),
			PrincipalId:      aws.String(testSSOAdminPrincipalID),
			PrincipalType:    ssotypes.PrincipalTypeUser,
		},
	)

	if got, want := resource.InstanceState.ID, testSSOAdminPrincipalID+",USER,"+testSSOAdminAccountID+",AWS_ACCOUNT,"+testSSOAdminPermissionSetARN+","+testSSOAdminInstanceARN; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceInfo.Type, ssoAdminAccountAssignmentResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"instance_arn":       testSSOAdminInstanceARN,
		"permission_set_arn": testSSOAdminPermissionSetARN,
		"principal_id":       testSSOAdminPrincipalID,
		"principal_type":     "USER",
		"target_id":          testSSOAdminAccountID,
		"target_type":        "AWS_ACCOUNT",
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestSSOAdminPermissionSetResource(t *testing.T) {
	resource := newSSOAdminPermissionSetResource(testSSOAdminInstanceARN, &ssotypes.PermissionSet{
		Description:      aws.String("Read-only access"),
		Name:             aws.String("ReadOnly"),
		PermissionSetArn: aws.String(testSSOAdminPermissionSetARN),
		RelayState:       aws.String("https://example.com/start"),
		SessionDuration:  aws.String("PT4H"),
	})

	if got, want := resource.InstanceState.ID, testSSOAdminPermissionSetARN+","+testSSOAdminInstanceARN; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceInfo.Type, ssoAdminPermissionSetResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"arn":              testSSOAdminPermissionSetARN,
		"description":      "Read-only access",
		"instance_arn":     testSSOAdminInstanceARN,
		"name":             "ReadOnly",
		"relay_state":      "https://example.com/start",
		"session_duration": "PT4H",
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
	if _, ok := attributes["permission_set_arn"]; ok {
		t.Fatalf("unexpected permission_set_arn attribute in %#v", attributes)
	}
}

func TestSSOAdminManagedPolicyAttachmentResource(t *testing.T) {
	resource := newSSOAdminManagedPolicyAttachmentResource(testSSOAdminInstanceARN, testSSOAdminPermissionSetARN, ssotypes.AttachedManagedPolicy{
		Arn:  aws.String(testSSOAdminPolicyARN),
		Name: aws.String("ReadOnlyAccess"),
	})

	if got, want := resource.InstanceState.ID, testSSOAdminPolicyARN+","+testSSOAdminPermissionSetARN+","+testSSOAdminInstanceARN; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceInfo.Type, ssoAdminManagedPolicyAttachmentResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"instance_arn":        testSSOAdminInstanceARN,
		"managed_policy_arn":  testSSOAdminPolicyARN,
		"managed_policy_name": "ReadOnlyAccess",
		"permission_set_arn":  testSSOAdminPermissionSetARN,
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
	if _, ok := resource.AdditionalFields["depends_on"]; ok {
		t.Fatalf("unexpected depends_on in %#v", resource.AdditionalFields)
	}
}

func TestSSOAdminPostConvertHookAddsManagedPolicyAttachmentDependsOnFilteredAssignments(t *testing.T) {
	managedPolicyAttachment := newSSOAdminManagedPolicyAttachmentResource(
		testSSOAdminInstanceARN,
		testSSOAdminPermissionSetARN,
		ssotypes.AttachedManagedPolicy{Arn: aws.String(testSSOAdminPolicyARN)},
	)
	managedPolicyAttachment.Item = map[string]interface{}{"permission_set_arn": testSSOAdminPermissionSetARN}
	matchingAssignment := newSSOAdminAccountAssignmentResource(
		testSSOAdminInstanceARN,
		testSSOAdminPermissionSetARN,
		testSSOAdminAccountID,
		ssotypes.AccountAssignment{
			AccountId:        aws.String(testSSOAdminAccountID),
			PermissionSetArn: aws.String(testSSOAdminPermissionSetARN),
			PrincipalId:      aws.String(testSSOAdminPrincipalID),
			PrincipalType:    ssotypes.PrincipalTypeUser,
		},
	)
	otherAssignment := newSSOAdminAccountAssignmentResource(
		testSSOAdminInstanceARN,
		testSSOAdminPermissionSetARN+"-other",
		testSSOAdminAccountID,
		ssotypes.AccountAssignment{
			AccountId:        aws.String(testSSOAdminAccountID),
			PermissionSetArn: aws.String(testSSOAdminPermissionSetARN + "-other"),
			PrincipalId:      aws.String("1234567890-aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
			PrincipalType:    ssotypes.PrincipalTypeGroup,
		},
	)
	g := &SSOAdminGenerator{}
	g.Resources = []terraformutils.Resource{managedPolicyAttachment, otherAssignment, matchingAssignment}

	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}

	dependsOn, ok := g.Resources[0].Item["depends_on"].([]string)
	if !ok {
		t.Fatalf("depends_on type = %T, want []string", g.Resources[0].Item["depends_on"])
	}
	want := []string{matchingAssignment.InstanceInfo.Id}
	if len(dependsOn) != len(want) {
		t.Fatalf("depends_on = %#v, want %#v", dependsOn, want)
	}
	for i := range want {
		if dependsOn[i] != want[i] {
			t.Fatalf("depends_on = %#v, want %#v", dependsOn, want)
		}
	}
	additionalDependsOn, ok := g.Resources[0].AdditionalFields["depends_on"].([]string)
	if !ok || len(additionalDependsOn) != 1 || additionalDependsOn[0] != want[0] {
		t.Fatalf("additional depends_on = %#v, want %#v", g.Resources[0].AdditionalFields["depends_on"], want)
	}
}

func TestSSOAdminPostConvertHookDropsDanglingManagedPolicyAttachmentDependsOn(t *testing.T) {
	managedPolicyAttachment := newSSOAdminManagedPolicyAttachmentResource(
		testSSOAdminInstanceARN,
		testSSOAdminPermissionSetARN,
		ssotypes.AttachedManagedPolicy{Arn: aws.String(testSSOAdminPolicyARN)},
	)
	managedPolicyAttachment.Item = map[string]interface{}{
		"depends_on":         []string{"aws_ssoadmin_account_assignment.tfer--filtered"},
		"permission_set_arn": testSSOAdminPermissionSetARN,
	}
	managedPolicyAttachment.AdditionalFields = map[string]interface{}{
		"depends_on": []string{"aws_ssoadmin_account_assignment.tfer--filtered"},
	}
	g := &SSOAdminGenerator{}
	g.Resources = []terraformutils.Resource{managedPolicyAttachment}

	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}

	if _, ok := g.Resources[0].Item["depends_on"]; ok {
		t.Fatalf("unexpected item depends_on = %#v", g.Resources[0].Item["depends_on"])
	}
	if _, ok := g.Resources[0].AdditionalFields["depends_on"]; ok {
		t.Fatalf("unexpected additional depends_on = %#v", g.Resources[0].AdditionalFields["depends_on"])
	}
}

func TestSSOAdminCustomerManagedPolicyAttachmentResource(t *testing.T) {
	tests := []struct {
		name       string
		policy     ssotypes.CustomerManagedPolicyReference
		policyPath string
	}{
		{
			name:       "explicit path",
			policy:     ssotypes.CustomerManagedPolicyReference{Name: aws.String("Boundary"), Path: aws.String("/service/")},
			policyPath: "/service/",
		},
		{
			name:       "default path",
			policy:     ssotypes.CustomerManagedPolicyReference{Name: aws.String("Boundary")},
			policyPath: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := newSSOAdminCustomerManagedPolicyAttachmentResource(testSSOAdminInstanceARN, testSSOAdminPermissionSetARN, tt.policy)

			if got, want := resource.InstanceState.ID, "Boundary,"+tt.policyPath+","+testSSOAdminPermissionSetARN+","+testSSOAdminInstanceARN; got != want {
				t.Fatalf("resource ID = %q, want %q", got, want)
			}
			if got, want := resource.InstanceInfo.Type, ssoAdminCustomerManagedPolicyAttachmentResourceType; got != want {
				t.Fatalf("resource type = %q, want %q", got, want)
			}
			attributes := resource.InstanceState.Attributes
			for key, want := range map[string]string{
				"customer_managed_policy_reference.#":      "1",
				"customer_managed_policy_reference.0.name": "Boundary",
				"customer_managed_policy_reference.0.path": tt.policyPath,
				"instance_arn":       testSSOAdminInstanceARN,
				"permission_set_arn": testSSOAdminPermissionSetARN,
			} {
				if got := attributes[key]; got != want {
					t.Fatalf("%s = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestSSOAdminCustomerManagedPolicyPath(t *testing.T) {
	tests := []struct {
		name string
		path *string
		want string
	}{
		{name: "nil", want: "/"},
		{name: "empty", path: aws.String(""), want: "/"},
		{name: "explicit", path: aws.String("/service/"), want: "/service/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ssoAdminCustomerManagedPolicyPath(tt.path); got != tt.want {
				t.Fatalf("ssoAdminCustomerManagedPolicyPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSSOAdminPermissionSetInlinePolicyResource(t *testing.T) {
	inlinePolicy := "{\"Version\":\"2012-10-17\",\"Statement\":[]}"
	resource := newSSOAdminPermissionSetInlinePolicyResource(testSSOAdminInstanceARN, testSSOAdminPermissionSetARN, inlinePolicy)

	if got, want := resource.InstanceState.ID, testSSOAdminPermissionSetARN+","+testSSOAdminInstanceARN; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceInfo.Type, ssoAdminPermissionSetInlinePolicyResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"inline_policy":      inlinePolicy,
		"instance_arn":       testSSOAdminInstanceARN,
		"permission_set_arn": testSSOAdminPermissionSetARN,
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestSSOAdminPostConvertHookWrapsInlinePolicy(t *testing.T) {
	inlinePolicy := "{\"Resource\":\"$" + "{aws:username}\"}"
	resource := newSSOAdminPermissionSetInlinePolicyResource(testSSOAdminInstanceARN, testSSOAdminPermissionSetARN, inlinePolicy)
	resource.Item = map[string]interface{}{"inline_policy": inlinePolicy}

	g := &SSOAdminGenerator{}
	g.Resources = append(g.Resources, resource)

	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}

	want := "<<POLICY\n{\"Resource\":\"$" + "$" + "{aws:username}\"}\nPOLICY"
	if got := g.Resources[0].Item["inline_policy"]; got != want {
		t.Fatalf("inline_policy = %q, want %q", got, want)
	}
}

func TestSSOAdminPostConvertHookEscapesInstanceAccessControlAttributeSources(t *testing.T) {
	rawGivenNameSource := "$" + "{path:name.givenName}"
	escapedGivenNameSource := "$" + "$" + "{path:name.givenName}"
	escapedFamilyNameSource := "$" + "$" + "{path:name.familyName}"
	rawDirectiveSource := "%" + "{if true}"
	escapedDirectiveSource := "%" + "%" + "{if true}"
	resource := newSSOAdminInstanceAccessControlAttributesResource(
		testSSOAdminInstanceARN,
		&ssotypes.InstanceAccessControlAttributeConfiguration{
			AccessControlAttributes: []ssotypes.AccessControlAttribute{
				{
					Key: aws.String("name"),
					Value: &ssotypes.AccessControlAttributeValue{
						Source: []string{rawGivenNameSource},
					},
				},
			},
		},
	)
	resource.Item = map[string]interface{}{
		"attribute": []interface{}{
			map[string]interface{}{
				"key": "name",
				"value": []interface{}{
					map[string]interface{}{
						"source": []interface{}{
							rawGivenNameSource,
							escapedFamilyNameSource,
							rawDirectiveSource,
						},
					},
				},
			},
		},
		"instance_arn": testSSOAdminInstanceARN,
	}
	g := &SSOAdminGenerator{}
	g.Resources = append(g.Resources, resource)

	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}

	attribute := g.Resources[0].Item["attribute"].([]interface{})[0].(map[string]interface{})
	value := attribute["value"].([]interface{})[0].(map[string]interface{})
	sources := value["source"].([]interface{})
	for i, want := range []string{
		escapedGivenNameSource,
		escapedFamilyNameSource,
		escapedDirectiveSource,
	} {
		if got := sources[i]; got != want {
			t.Fatalf("source[%d] = %q, want %q", i, got, want)
		}
	}

	data, err := terraformutils.HclPrintResource(g.Resources, map[string]interface{}{}, "hcl", true)
	if err != nil {
		t.Fatalf("HclPrintResource() error = %v", err)
	}
	output := string(data)
	for _, want := range []string{
		"\"" + escapedGivenNameSource + "\"",
		"\"" + escapedFamilyNameSource + "\"",
		"\"" + escapedDirectiveSource + "\"",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output does not contain %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "\""+rawGivenNameSource+"\"") || strings.Contains(output, "\""+rawDirectiveSource+"\"") {
		t.Fatalf("output contains unescaped Terraform template markers:\n%s", output)
	}
}

func TestSSOAdminPermissionsBoundaryAttachmentResource(t *testing.T) {
	tests := []struct {
		name       string
		boundary   *ssotypes.PermissionsBoundary
		attributes map[string]string
	}{
		{
			name: "managed policy",
			boundary: &ssotypes.PermissionsBoundary{
				ManagedPolicyArn: aws.String(testSSOAdminPolicyARN),
			},
			attributes: map[string]string{
				"permissions_boundary.0.managed_policy_arn": testSSOAdminPolicyARN,
			},
		},
		{
			name: "customer managed policy",
			boundary: &ssotypes.PermissionsBoundary{
				CustomerManagedPolicyReference: &ssotypes.CustomerManagedPolicyReference{
					Name: aws.String("Boundary"),
					Path: aws.String("/service/"),
				},
			},
			attributes: map[string]string{
				"permissions_boundary.0.customer_managed_policy_reference.#":      "1",
				"permissions_boundary.0.customer_managed_policy_reference.0.name": "Boundary",
				"permissions_boundary.0.customer_managed_policy_reference.0.path": "/service/",
			},
		},
		{
			name: "customer managed policy default path",
			boundary: &ssotypes.PermissionsBoundary{
				CustomerManagedPolicyReference: &ssotypes.CustomerManagedPolicyReference{
					Name: aws.String("Boundary"),
				},
			},
			attributes: map[string]string{
				"permissions_boundary.0.customer_managed_policy_reference.#":      "1",
				"permissions_boundary.0.customer_managed_policy_reference.0.name": "Boundary",
				"permissions_boundary.0.customer_managed_policy_reference.0.path": "/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := newSSOAdminPermissionsBoundaryAttachmentResource(testSSOAdminInstanceARN, testSSOAdminPermissionSetARN, tt.boundary)

			if got, want := resource.InstanceState.ID, testSSOAdminPermissionSetARN+","+testSSOAdminInstanceARN; got != want {
				t.Fatalf("resource ID = %q, want %q", got, want)
			}
			if got, want := resource.InstanceInfo.Type, ssoAdminPermissionsBoundaryAttachmentResourceType; got != want {
				t.Fatalf("resource type = %q, want %q", got, want)
			}
			attributes := resource.InstanceState.Attributes
			for key, want := range map[string]string{
				"instance_arn":           testSSOAdminInstanceARN,
				"permission_set_arn":     testSSOAdminPermissionSetARN,
				"permissions_boundary.#": "1",
			} {
				if got := attributes[key]; got != want {
					t.Fatalf("%s = %q, want %q", key, got, want)
				}
			}
			for key, want := range tt.attributes {
				if got := attributes[key]; got != want {
					t.Fatalf("%s = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestSSOAdminResourceNamesDoNotCollapseJoinedParts(t *testing.T) {
	left := newSSOAdminManagedPolicyAttachmentResource("instance", "a_b", ssotypes.AttachedManagedPolicy{Arn: aws.String("c")})
	right := newSSOAdminManagedPolicyAttachmentResource("instance", "a", ssotypes.AttachedManagedPolicy{Arn: aws.String("b_c")})
	if left.ResourceName == right.ResourceName {
		t.Fatalf("managed policy attachment resource names collide: %q", left.ResourceName)
	}

	left = newSSOAdminAccountAssignmentResource(
		"instance",
		"permission",
		"a_b",
		ssotypes.AccountAssignment{PrincipalId: aws.String("c"), PrincipalType: ssotypes.PrincipalTypeGroup},
	)
	right = newSSOAdminAccountAssignmentResource(
		"instance",
		"permission",
		"a",
		ssotypes.AccountAssignment{PrincipalId: aws.String("b_c"), PrincipalType: ssotypes.PrincipalTypeGroup},
	)
	if left.ResourceName == right.ResourceName {
		t.Fatalf("account assignment resource names collide: %q", left.ResourceName)
	}

	config := &ssotypes.InstanceAccessControlAttributeConfiguration{
		AccessControlAttributes: []ssotypes.AccessControlAttribute{
			{
				Key: aws.String("name"),
				Value: &ssotypes.AccessControlAttributeValue{
					Source: []string{"${path:name.givenName}"},
				},
			},
		},
	}
	left = newSSOAdminInstanceAccessControlAttributesResource("instance:a_b", config)
	right = newSSOAdminInstanceAccessControlAttributesResource("instance:a:b", config)
	if left.ResourceName == right.ResourceName {
		t.Fatalf("instance access control attributes resource names collide: %q", left.ResourceName)
	}
}

func TestSSOAdminInstanceAccessControlAttributesConfigured(t *testing.T) {
	tests := []struct {
		name   string
		config *ssotypes.InstanceAccessControlAttributeConfiguration
		want   bool
	}{
		{name: "nil", want: false},
		{name: "empty", config: &ssotypes.InstanceAccessControlAttributeConfiguration{}, want: false},
		{
			name: "configured",
			config: &ssotypes.InstanceAccessControlAttributeConfiguration{
				AccessControlAttributes: []ssotypes.AccessControlAttribute{
					{
						Key: aws.String("name"),
						Value: &ssotypes.AccessControlAttributeValue{
							Source: []string{"${path:name.givenName}"},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "missing key",
			config: &ssotypes.InstanceAccessControlAttributeConfiguration{
				AccessControlAttributes: []ssotypes.AccessControlAttribute{
					{Value: &ssotypes.AccessControlAttributeValue{Source: []string{"${path:name.givenName}"}}},
				},
			},
			want: false,
		},
		{
			name: "missing value",
			config: &ssotypes.InstanceAccessControlAttributeConfiguration{
				AccessControlAttributes: []ssotypes.AccessControlAttribute{{Key: aws.String("name")}},
			},
			want: false,
		},
		{
			name: "missing source",
			config: &ssotypes.InstanceAccessControlAttributeConfiguration{
				AccessControlAttributes: []ssotypes.AccessControlAttribute{
					{Key: aws.String("name"), Value: &ssotypes.AccessControlAttributeValue{}},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ssoAdminInstanceAccessControlAttributesConfigured(tt.config); got != tt.want {
				t.Fatalf("ssoAdminInstanceAccessControlAttributesConfigured() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestSSOAdminAccountAssignmentConfigured(t *testing.T) {
	tests := []struct {
		name       string
		targetID   string
		assignment ssotypes.AccountAssignment
		want       bool
	}{
		{
			name:     "configured",
			targetID: testSSOAdminAccountID,
			assignment: ssotypes.AccountAssignment{
				PrincipalId:   aws.String(testSSOAdminPrincipalID),
				PrincipalType: ssotypes.PrincipalTypeUser,
			},
			want: true,
		},
		{
			name:     "missing target ID",
			targetID: "",
			assignment: ssotypes.AccountAssignment{
				PrincipalId:   aws.String(testSSOAdminPrincipalID),
				PrincipalType: ssotypes.PrincipalTypeUser,
			},
		},
		{
			name:     "missing principal ID",
			targetID: testSSOAdminAccountID,
			assignment: ssotypes.AccountAssignment{
				PrincipalType: ssotypes.PrincipalTypeUser,
			},
		},
		{
			name:     "missing principal type",
			targetID: testSSOAdminAccountID,
			assignment: ssotypes.AccountAssignment{
				PrincipalId: aws.String(testSSOAdminPrincipalID),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ssoAdminAccountAssignmentConfigured(tt.targetID, tt.assignment); got != tt.want {
				t.Fatalf("ssoAdminAccountAssignmentConfigured() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestSSOAdminPermissionsBoundaryConfigured(t *testing.T) {
	tests := []struct {
		name     string
		boundary *ssotypes.PermissionsBoundary
		want     bool
	}{
		{name: "nil", want: false},
		{name: "empty", boundary: &ssotypes.PermissionsBoundary{}, want: false},
		{name: "managed policy", boundary: &ssotypes.PermissionsBoundary{ManagedPolicyArn: aws.String(testSSOAdminPolicyARN)}, want: true},
		{
			name: "customer managed policy",
			boundary: &ssotypes.PermissionsBoundary{
				CustomerManagedPolicyReference: &ssotypes.CustomerManagedPolicyReference{Name: aws.String("Boundary"), Path: aws.String("/")},
			},
			want: true,
		},
		{
			name: "customer managed policy missing path",
			boundary: &ssotypes.PermissionsBoundary{
				CustomerManagedPolicyReference: &ssotypes.CustomerManagedPolicyReference{Name: aws.String("Boundary")},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ssoAdminPermissionsBoundaryConfigured(tt.boundary); got != tt.want {
				t.Fatalf("ssoAdminPermissionsBoundaryConfigured() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestSSOAdminInstanceAccessControlAttributesNotConfigured(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "resource not found", err: &ssotypes.ResourceNotFoundException{}, want: true},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ssoAdminInstanceAccessControlAttributesNotConfigured(tt.err); got != tt.want {
				t.Fatalf("ssoAdminInstanceAccessControlAttributesNotConfigured(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func TestSSOAdminResourceNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "resource not found", err: &ssotypes.ResourceNotFoundException{}, want: true},
		{name: "wrapped resource not found", err: errors.Join(errors.New("lookup failed"), &ssotypes.ResourceNotFoundException{}), want: true},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ssoAdminResourceNotFound(tt.err); got != tt.want {
				t.Fatalf("ssoAdminResourceNotFound(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}
