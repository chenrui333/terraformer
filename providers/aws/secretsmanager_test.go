// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	secretstypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestSecretsManagerPostConvertHookMovesSplitPolicyOutOfSecret(t *testing.T) {
	secretARN := "arn:aws:secretsmanager:us-east-1:123456789012:secret:orders-abc123"
	secret := newSecretsManagerSecretResource(secretARN, "orders")
	secret.Item = map[string]interface{}{
		"name":   "orders",
		"policy": "{\"Version\":\"2012-10-17\"}",
	}
	policy := newSecretsManagerSecretPolicyResource(secretARN, "orders", "{\"Version\":\"2012-10-17\"}")
	policy.Item = map[string]interface{}{
		"secret_arn": secretARN,
		"policy":     "{\"Version\":\"2012-10-17\"}",
	}

	g := SecretsManagerGenerator{}
	g.Resources = []terraformutils.Resource{secret, policy}

	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}
	if _, ok := g.Resources[0].Item["policy"]; ok {
		t.Fatal("secret inline policy should be removed when split policy resource exists")
	}
	assertSecretsManagerPolicyHeredoc(t, g.Resources[1].Item["policy"])
}

func TestSecretsManagerPostConvertHookKeepsInlinePolicyWithoutSplitResource(t *testing.T) {
	secretARN := "arn:aws:secretsmanager:us-east-1:123456789012:secret:orders-abc123"
	secret := newSecretsManagerSecretResource(secretARN, "orders")
	secret.Item = map[string]interface{}{"policy": "{\"Version\":\"2012-10-17\"}"}

	g := SecretsManagerGenerator{}
	g.Resources = []terraformutils.Resource{secret}

	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}
	assertSecretsManagerPolicyHeredoc(t, g.Resources[0].Item["policy"])
}

func TestSecretsManagerSecretRotationConfigured(t *testing.T) {
	days := int64(30)
	tests := []struct {
		name   string
		secret secretstypes.SecretListEntry
		want   bool
	}{
		{name: "disabled", secret: secretstypes.SecretListEntry{RotationEnabled: aws.Bool(false), RotationRules: &secretstypes.RotationRulesType{AutomaticallyAfterDays: &days}}, want: false},
		{name: "enabled without rules", secret: secretstypes.SecretListEntry{RotationEnabled: aws.Bool(true)}, want: false},
		{name: "enabled with empty rules", secret: secretstypes.SecretListEntry{RotationEnabled: aws.Bool(true), RotationRules: &secretstypes.RotationRulesType{}}, want: false},
		{name: "enabled with days", secret: secretstypes.SecretListEntry{RotationEnabled: aws.Bool(true), RotationRules: &secretstypes.RotationRulesType{AutomaticallyAfterDays: &days}}, want: true},
		{name: "enabled with schedule", secret: secretstypes.SecretListEntry{RotationEnabled: aws.Bool(true), RotationRules: &secretstypes.RotationRulesType{ScheduleExpression: aws.String("rate(10 days)")}}, want: true},
		{name: "enabled with duration", secret: secretstypes.SecretListEntry{RotationEnabled: aws.Bool(true), RotationRules: &secretstypes.RotationRulesType{Duration: aws.String("3h")}}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := secretsManagerSecretRotationConfigured(tt.secret); got != tt.want {
				t.Fatalf("secretsManagerSecretRotationConfigured() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestSecretsManagerFilterGatesSecretAndChildDiscovery(t *testing.T) {
	secretARN := "arn:aws:secretsmanager:us-east-1:123456789012:secret:orders-abc123"
	otherARN := "arn:aws:secretsmanager:us-east-1:123456789012:secret:other-abc123"
	secret := newSecretsManagerSecretResource(secretARN, "orders")
	other := newSecretsManagerSecretResource(otherARN, "other")

	tests := []struct {
		name           string
		filters        []terraformutils.ResourceFilter
		appendSecret   bool
		appendOther    bool
		loadChildren   bool
		loadOtherChild bool
	}{
		{
			name:           "no filters imports secrets and children",
			appendSecret:   true,
			appendOther:    true,
			loadChildren:   true,
			loadOtherChild: true,
		},
		{
			name: "typed secret id filter limits secret and children",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: "secretsmanager_secret", FieldPath: "id", AcceptableValues: []string{secretARN}},
			},
			appendSecret: true,
			loadChildren: true,
		},
		{
			name: "typed child id filter does not import parent secrets",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: "secretsmanager_secret_policy", FieldPath: "id", AcceptableValues: []string{secretARN}},
			},
			loadChildren: true,
		},
		{
			name: "untyped id filter limits same-id resources",
			filters: []terraformutils.ResourceFilter{
				{FieldPath: "id", AcceptableValues: []string{secretARN}},
			},
			appendSecret: true,
			loadChildren: true,
		},
		{
			name: "typed non-id secret filter does not pre-load children",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: "secretsmanager_secret", FieldPath: "tags.env", AcceptableValues: []string{"prod"}},
			},
			appendSecret: true,
			appendOther:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := SecretsManagerGenerator{}
			g.Filter = tt.filters
			if got := g.shouldAppendSecretResource(secret); got != tt.appendSecret {
				t.Fatalf("shouldAppendSecretResource(secret) = %t, want %t", got, tt.appendSecret)
			}
			if got := g.shouldAppendSecretResource(other); got != tt.appendOther {
				t.Fatalf("shouldAppendSecretResource(other) = %t, want %t", got, tt.appendOther)
			}
			if got := g.shouldLoadSecretChildren(secret); got != tt.loadChildren {
				t.Fatalf("shouldLoadSecretChildren(secret) = %t, want %t", got, tt.loadChildren)
			}
			if got := g.shouldLoadSecretChildren(other); got != tt.loadOtherChild {
				t.Fatalf("shouldLoadSecretChildren(other) = %t, want %t", got, tt.loadOtherChild)
			}
		})
	}
}

func TestSecretsManagerFilterGatesChildAppend(t *testing.T) {
	secretARN := "arn:aws:secretsmanager:us-east-1:123456789012:secret:orders-abc123"
	policy := newSecretsManagerSecretPolicyResource(secretARN, "orders", "{\"Version\":\"2012-10-17\"}")
	rotation := newSecretsManagerSecretRotationResource(secretARN, "orders")

	tests := []struct {
		name           string
		filters        []terraformutils.ResourceFilter
		appendPolicy   bool
		appendRotation bool
	}{
		{name: "no typed child filter appends all configured children", appendPolicy: true, appendRotation: true},
		{
			name: "typed policy id filter appends only policy",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: "secretsmanager_secret_policy", FieldPath: "id", AcceptableValues: []string{secretARN}},
			},
			appendPolicy: true,
		},
		{
			name: "typed rotation id filter appends only rotation",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: "secretsmanager_secret_rotation", FieldPath: "id", AcceptableValues: []string{secretARN}},
			},
			appendRotation: true,
		},
		{
			name: "typed child id filter with different secret skips all siblings",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: "secretsmanager_secret_policy", FieldPath: "id", AcceptableValues: []string{"arn:aws:secretsmanager:us-east-1:123456789012:secret:other-abc123"}},
			},
		},
		{
			name: "typed child non-id filter allows requested child type",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: "secretsmanager_secret_policy", FieldPath: "policy", AcceptableValues: []string{"{\"Version\":\"2012-10-17\"}"}},
			},
			appendPolicy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := SecretsManagerGenerator{}
			g.Filter = tt.filters
			if got := g.shouldAppendSecretChildResource("secretsmanager_secret_policy", policy); got != tt.appendPolicy {
				t.Fatalf("shouldAppendSecretChildResource(policy) = %t, want %t", got, tt.appendPolicy)
			}
			if got := g.shouldAppendSecretChildResource("secretsmanager_secret_rotation", rotation); got != tt.appendRotation {
				t.Fatalf("shouldAppendSecretChildResource(rotation) = %t, want %t", got, tt.appendRotation)
			}
		})
	}
}

func TestSecretsManagerResourceMissing(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "typed not found", err: &secretstypes.ResourceNotFoundException{}, want: true},
		{name: "wrapped not found", err: errors.Join(errors.New("wrapper"), &secretstypes.ResourceNotFoundException{}), want: true},
		{name: "marked for deletion", err: &secretstypes.InvalidRequestException{Message: aws.String("You can't perform this operation on the secret because it was marked for deletion")}, want: true},
		{name: "generic invalid request", err: &secretstypes.InvalidRequestException{Message: aws.String("other invalid request")}, want: false},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := secretsManagerResourceMissing(tt.err); got != tt.want {
				t.Fatalf("secretsManagerResourceMissing(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func assertSecretsManagerPolicyHeredoc(t *testing.T, value interface{}) {
	t.Helper()

	policy, ok := value.(string)
	if !ok {
		t.Fatalf("value has type %T, want string", value)
	}
	if !strings.HasPrefix(policy, "<<POLICY\n") || !strings.HasSuffix(policy, "\nPOLICY") {
		t.Fatalf("value %q is not wrapped as a POLICY heredoc", policy)
	}
}
