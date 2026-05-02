// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	configtypes "github.com/aws/aws-sdk-go-v2/service/configservice/types"
)

func TestConfigAggregateAuthorizationID(t *testing.T) {
	got := configAggregateAuthorizationID("123456789012", "us-east-1")
	want := "123456789012:us-east-1"
	if got != want {
		t.Fatalf("configAggregateAuthorizationID() = %q, want %q", got, want)
	}
}

func TestConfigOrganizationRuleResourceType(t *testing.T) {
	tests := []struct {
		name string
		rule configtypes.OrganizationConfigRule
		want string
	}{
		{
			name: "managed rule",
			rule: configtypes.OrganizationConfigRule{
				OrganizationManagedRuleMetadata: &configtypes.OrganizationManagedRuleMetadata{},
			},
			want: "aws_config_organization_managed_rule",
		},
		{
			name: "custom rule",
			rule: configtypes.OrganizationConfigRule{
				OrganizationCustomRuleMetadata: &configtypes.OrganizationCustomRuleMetadata{},
			},
			want: "aws_config_organization_custom_rule",
		},
		{
			name: "custom policy rule",
			rule: configtypes.OrganizationConfigRule{
				OrganizationCustomPolicyRuleMetadata: &configtypes.OrganizationCustomPolicyRuleMetadataNoPolicy{},
			},
			want: "aws_config_organization_custom_policy_rule",
		},
		{
			name: "unknown rule shape",
			rule: configtypes.OrganizationConfigRule{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := configOrganizationRuleResourceType(tt.rule)
			if got != tt.want {
				t.Fatalf("configOrganizationRuleResourceType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfigRemediationConfigurationMissing(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "no such config rule",
			err:  &configtypes.NoSuchConfigRuleException{},
			want: true,
		},
		{
			name: "wrapped no such config rule",
			err:  errors.Join(errors.New("lookup failed"), &configtypes.NoSuchConfigRuleException{}),
			want: true,
		},
		{
			name: "no such remediation configuration",
			err:  &configtypes.NoSuchRemediationConfigurationException{},
			want: true,
		},
		{
			name: "wrapped no such remediation configuration",
			err:  errors.Join(errors.New("lookup failed"), &configtypes.NoSuchRemediationConfigurationException{}),
			want: true,
		},
		{
			name: "generic error",
			err:  errors.New("boom"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := configRemediationConfigurationMissing(tt.err)
			if got != tt.want {
				t.Fatalf("configRemediationConfigurationMissing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChunkStrings(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		size   int
		want   [][]string
	}{
		{
			name:   "chunks values",
			values: []string{"a", "b", "c", "d", "e"},
			size:   2,
			want:   [][]string{{"a", "b"}, {"c", "d"}, {"e"}},
		},
		{
			name:   "empty values",
			values: nil,
			size:   2,
			want:   nil,
		},
		{
			name:   "invalid size",
			values: []string{"a"},
			size:   0,
			want:   nil,
		},
		{
			name:   "negative size",
			values: []string{"a"},
			size:   -1,
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chunkStrings(tt.values, tt.size)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("chunkStrings() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestConfigOrganizationRuleResourceTypePrefersManagedShape(t *testing.T) {
	rule := configtypes.OrganizationConfigRule{
		OrganizationConfigRuleName: aws.String("example"),
		OrganizationManagedRuleMetadata: &configtypes.OrganizationManagedRuleMetadata{
			RuleIdentifier: aws.String("S3_BUCKET_VERSIONING_ENABLED"),
		},
		OrganizationCustomRuleMetadata: &configtypes.OrganizationCustomRuleMetadata{},
	}

	got := configOrganizationRuleResourceType(rule)
	want := "aws_config_organization_managed_rule"
	if got != want {
		t.Fatalf("configOrganizationRuleResourceType() = %q, want %q", got, want)
	}
}
