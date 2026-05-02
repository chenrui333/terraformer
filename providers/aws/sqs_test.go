// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"strings"
	"testing"

	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestSqsPostConvertHookMovesSplitAttributesOutOfQueue(t *testing.T) {
	queueURL := "https://sqs.us-east-1.amazonaws.com/123456789012/orders"
	queue := terraformutils.NewSimpleResource(queueURL, "orders", "aws_sqs_queue", "aws", sqsAllowEmptyValues)
	queue.Item = map[string]interface{}{
		"policy":                     "{\"Version\":\"2012-10-17\"}",
		"redrive_policy":             "{\"deadLetterTargetArn\":\"arn:aws:sqs:us-east-1:123456789012:orders-dlq\",\"maxReceiveCount\":5}",
		"redrive_allow_policy":       "{\"redrivePermission\":\"allowAll\"}",
		"visibility_timeout_seconds": 30,
	}

	queuePolicy := terraformutils.NewResource(
		queueURL,
		"orders",
		"aws_sqs_queue_policy",
		"aws",
		map[string]string{"queue_url": queueURL, "policy": "{\"Version\":\"2012-10-17\"}"},
		sqsAllowEmptyValues,
		map[string]interface{}{},
	)
	queuePolicy.Item = map[string]interface{}{"queue_url": queueURL, "policy": "{\"Version\":\"2012-10-17\"}"}

	redrivePolicy := terraformutils.NewResource(
		queueURL,
		"orders",
		"aws_sqs_queue_redrive_policy",
		"aws",
		map[string]string{"queue_url": queueURL, "redrive_policy": "{\"deadLetterTargetArn\":\"arn:aws:sqs:us-east-1:123456789012:orders-dlq\",\"maxReceiveCount\":5}"},
		sqsAllowEmptyValues,
		map[string]interface{}{},
	)
	redrivePolicy.Item = map[string]interface{}{"queue_url": queueURL, "redrive_policy": "{\"deadLetterTargetArn\":\"arn:aws:sqs:us-east-1:123456789012:orders-dlq\",\"maxReceiveCount\":5}"}

	redriveAllowPolicy := terraformutils.NewResource(
		queueURL,
		"orders",
		"aws_sqs_queue_redrive_allow_policy",
		"aws",
		map[string]string{"queue_url": queueURL, "redrive_allow_policy": "{\"redrivePermission\":\"allowAll\"}"},
		sqsAllowEmptyValues,
		map[string]interface{}{},
	)
	redriveAllowPolicy.Item = map[string]interface{}{"queue_url": queueURL, "redrive_allow_policy": "{\"redrivePermission\":\"allowAll\"}"}

	g := SqsGenerator{}
	g.Resources = []terraformutils.Resource{queue, queuePolicy, redrivePolicy, redriveAllowPolicy}

	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}

	for _, key := range []string{"policy", "redrive_policy", "redrive_allow_policy"} {
		if _, ok := g.Resources[0].Item[key]; ok {
			t.Fatalf("queue inline %s should be removed when split resource exists", key)
		}
	}
	if _, ok := g.Resources[0].Item["visibility_timeout_seconds"]; !ok {
		t.Fatal("unrelated queue attributes should be preserved")
	}

	assertSqsHeredoc(t, g.Resources[1].Item["policy"])
	assertSqsHeredoc(t, g.Resources[2].Item["redrive_policy"])
	assertSqsHeredoc(t, g.Resources[3].Item["redrive_allow_policy"])
}

func TestSqsPostConvertHookKeepsInlinePolicyWithoutSplitResource(t *testing.T) {
	queueURL := "https://sqs.us-east-1.amazonaws.com/123456789012/orders"
	queue := terraformutils.NewSimpleResource(queueURL, "orders", "aws_sqs_queue", "aws", sqsAllowEmptyValues)
	queue.Item = map[string]interface{}{"policy": "{\"Version\":\"2012-10-17\"}"}

	g := SqsGenerator{}
	g.Resources = []terraformutils.Resource{queue}

	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}

	assertSqsHeredoc(t, g.Resources[0].Item["policy"])
}

func TestSqsQueueAttributeConfigured(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "empty", value: "", want: false},
		{name: "json object", value: "{}", want: true},
		{name: "json policy", value: "{\"Version\":\"2012-10-17\"}", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sqsQueueAttributeConfigured(tt.value); got != tt.want {
				t.Fatalf("sqsQueueAttributeConfigured(%q) = %t, want %t", tt.value, got, tt.want)
			}
		})
	}
}

func TestSqsFilterGatesQueueAndChildDiscovery(t *testing.T) {
	queueURL := "https://sqs.us-east-1.amazonaws.com/123456789012/orders"
	otherURL := "https://sqs.us-east-1.amazonaws.com/123456789012/other"
	queue := newSqsQueueResource(queueURL, "orders")
	otherQueue := newSqsQueueResource(otherURL, "other")

	tests := []struct {
		name           string
		filters        []terraformutils.ResourceFilter
		appendQueue    bool
		appendOther    bool
		loadQueueChild bool
		loadOtherChild bool
	}{
		{
			name:           "no filters imports queue and children",
			appendQueue:    true,
			appendOther:    true,
			loadQueueChild: true,
			loadOtherChild: true,
		},
		{
			name: "typed queue id filter limits queue and children",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: "sqs_queue", FieldPath: "id", AcceptableValues: []string{queueURL}},
			},
			appendQueue:    true,
			loadQueueChild: true,
		},
		{
			name: "typed child id filter does not import parent queues",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: "sqs_queue_policy", FieldPath: "id", AcceptableValues: []string{queueURL}},
			},
			loadQueueChild: true,
		},
		{
			name: "untyped id filter limits all same-id resources",
			filters: []terraformutils.ResourceFilter{
				{FieldPath: "id", AcceptableValues: []string{queueURL}},
			},
			appendQueue:    true,
			loadQueueChild: true,
		},
		{
			name: "typed non-id queue filter does not pre-load children",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: "sqs_queue", FieldPath: "tags.env", AcceptableValues: []string{"prod"}},
			},
			appendQueue: true,
			appendOther: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := SqsGenerator{}
			g.Filter = tt.filters
			if got := g.shouldAppendQueueResource(queue); got != tt.appendQueue {
				t.Fatalf("shouldAppendQueueResource(queue) = %t, want %t", got, tt.appendQueue)
			}
			if got := g.shouldAppendQueueResource(otherQueue); got != tt.appendOther {
				t.Fatalf("shouldAppendQueueResource(other) = %t, want %t", got, tt.appendOther)
			}
			if got := g.shouldLoadQueueAttributeResources(queue); got != tt.loadQueueChild {
				t.Fatalf("shouldLoadQueueAttributeResources(queue) = %t, want %t", got, tt.loadQueueChild)
			}
			if got := g.shouldLoadQueueAttributeResources(otherQueue); got != tt.loadOtherChild {
				t.Fatalf("shouldLoadQueueAttributeResources(other) = %t, want %t", got, tt.loadOtherChild)
			}
		})
	}
}

func TestSqsQueueMissing(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "typed missing", err: &sqstypes.QueueDoesNotExist{}, want: true},
		{name: "wrapped typed missing", err: errors.Join(errors.New("wrapper"), &sqstypes.QueueDoesNotExist{}), want: true},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sqsQueueMissing(tt.err); got != tt.want {
				t.Fatalf("sqsQueueMissing(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func assertSqsHeredoc(t *testing.T, value interface{}) {
	t.Helper()

	policy, ok := value.(string)
	if !ok {
		t.Fatalf("value has type %T, want string", value)
	}
	if !strings.HasPrefix(policy, "<<POLICY\n") || !strings.HasSuffix(policy, "\nPOLICY") {
		t.Fatalf("value %q is not wrapped as a POLICY heredoc", policy)
	}
}
