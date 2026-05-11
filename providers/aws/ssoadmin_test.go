// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ssotypes "github.com/aws/aws-sdk-go-v2/service/ssoadmin/types"
)

const (
	testSSOAdminInstanceARN      = "arn:aws:sso:::instance/ssoins-1234567890abcdef"
	testSSOAdminPermissionSetARN = "arn:aws:sso:::permissionSet/ssoins-1234567890abcdef/ps-1234567890abcdef"
	testSSOAdminPolicyARN        = "arn:aws:iam::aws:policy/ReadOnlyAccess"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("resource ID = %q, want %q", tt.got, tt.want)
			}
		})
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
