// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestSNSImportIDs(t *testing.T) {
	arn := "arn:aws:sns:us-east-1:123456789012:topic-a"
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "topic policy", got: snsTopicPolicyImportID(arn), want: arn},
		{name: "topic data protection policy", got: snsTopicDataProtectionPolicyImportID(arn), want: arn},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("import ID = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestNewSNSTopicPolicyResource(t *testing.T) {
	arn := "arn:aws:sns:us-east-1:123456789012:topic-a"
	resource, ok := newSNSTopicPolicyResource(arn, "topic-a", &sns.GetTopicAttributesOutput{
		Attributes: map[string]string{
			snsTopicAttributeOwner:    "123456789012",
			snsTopicAttributePolicy:   `{"Statement":[]}`,
			snsTopicAttributeTopicARN: arn,
		},
	})
	assertMessagingResource(t, resource, ok, snsTopicPolicyResourceType, arn, map[string]string{
		"arn":    arn,
		"owner":  "123456789012",
		"policy": `{"Statement":[]}`,
	})

	if _, ok := newSNSTopicPolicyResource("", "topic-a", &sns.GetTopicAttributesOutput{}); ok {
		t.Fatal("empty topic ARN should be skipped")
	}
	if _, ok := newSNSTopicPolicyResource(arn, "topic-a", &sns.GetTopicAttributesOutput{}); ok {
		t.Fatal("missing policy should be skipped")
	}
}

func TestNewSNSTopicDataProtectionPolicyResource(t *testing.T) {
	arn := "arn:aws:sns:us-east-1:123456789012:topic-a"
	resource, ok := newSNSTopicDataProtectionPolicyResource(arn, "topic-a", &sns.GetDataProtectionPolicyOutput{
		DataProtectionPolicy: aws.String(`{"Name":"protect"}`),
	})
	assertMessagingResource(t, resource, ok, snsTopicDataProtectionPolicyType, arn, map[string]string{
		"arn":    arn,
		"policy": `{"Name":"protect"}`,
	})

	if _, ok := newSNSTopicDataProtectionPolicyResource("", "topic-a", &sns.GetDataProtectionPolicyOutput{DataProtectionPolicy: aws.String(`{}`)}); ok {
		t.Fatal("empty topic ARN should be skipped")
	}
	if _, ok := newSNSTopicDataProtectionPolicyResource(arn, "topic-a", &sns.GetDataProtectionPolicyOutput{}); ok {
		t.Fatal("empty data protection policy should be skipped")
	}
}

func TestSNSPostConvertHookRemovesInlinePolicyWhenSplitPolicyExists(t *testing.T) {
	arn := "arn:aws:sns:us-east-1:123456789012:topic-a"
	topic := terraformutils.NewSimpleResource(arn, "topic-a", snsTopicResourceType, "aws", snsAllowEmptyValues)
	topic.Item = map[string]interface{}{
		"name":   "topic-a",
		"policy": `{"Condition":"${aws:username}"}`,
	}
	policy := terraformutils.NewSimpleResource(arn, "topic-policy-a", snsTopicPolicyResourceType, "aws", snsAllowEmptyValues)
	policy.Item = map[string]interface{}{
		"arn":    arn,
		"policy": `{"Condition":"${aws:username}"}`,
	}
	g := SnsGenerator{}
	g.Resources = []terraformutils.Resource{topic, policy}

	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}
	if _, ok := g.Resources[0].Item["policy"]; ok {
		t.Fatal("topic inline policy was not removed when split topic policy exists")
	}
	policyValue, ok := g.Resources[1].Item["policy"].(string)
	if !ok {
		t.Fatalf("split policy type = %T, want string", g.Resources[1].Item["policy"])
	}
	if !strings.HasPrefix(policyValue, "<<POLICY\n") || !strings.HasSuffix(policyValue, "\nPOLICY") {
		t.Fatalf("policy was not heredoc wrapped: %q", policyValue)
	}
	if !strings.Contains(policyValue, "$${aws:username}") {
		t.Fatalf("policy interpolation was not escaped: %q", policyValue)
	}
}
