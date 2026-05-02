// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"fmt"
	"testing"

	kinesistypes "github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestKinesisConsumerImportable(t *testing.T) {
	tests := []struct {
		name     string
		consumer kinesistypes.Consumer
		want     bool
	}{
		{name: "active", consumer: kinesistypes.Consumer{ConsumerStatus: kinesistypes.ConsumerStatusActive}, want: true},
		{name: "creating", consumer: kinesistypes.Consumer{ConsumerStatus: kinesistypes.ConsumerStatusCreating}, want: false},
		{name: "deleting", consumer: kinesistypes.Consumer{ConsumerStatus: kinesistypes.ConsumerStatusDeleting}, want: false},
		{name: "empty", consumer: kinesistypes.Consumer{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := kinesisConsumerImportable(tt.consumer)
			if got != tt.want {
				t.Fatalf("kinesisConsumerImportable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKinesisResourceName(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "filters empty parts", parts: []string{"", "orders", "", "policy"}, want: "orders/policy"},
		{name: "preserves segment boundaries", parts: []string{"orders", "stream", "policy"}, want: "orders/stream/policy"},
		{name: "fallback", parts: nil, want: "kinesis_resource"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := kinesisResourceName(tt.parts...)
			if got != tt.want {
				t.Fatalf("kinesisResourceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestKinesisShouldLoadStreamChildrenHonorsStreamIDFilters(t *testing.T) {
	tests := []struct {
		name    string
		filters []terraformutils.ResourceFilter
		stream  string
		want    bool
	}{
		{name: "no filters", stream: "orders", want: true},
		{
			name: "matching stream id filter",
			filters: []terraformutils.ResourceFilter{{
				ServiceName:      "kinesis_stream",
				FieldPath:        "id",
				AcceptableValues: []string{"orders"},
			}},
			stream: "orders",
			want:   true,
		},
		{
			name: "nonmatching stream id filter",
			filters: []terraformutils.ResourceFilter{{
				ServiceName:      "kinesis_stream",
				FieldPath:        "id",
				AcceptableValues: []string{"orders"},
			}},
			stream: "payments",
			want:   false,
		},
		{
			name: "child id filter keeps discovery despite nonmatching stream filter",
			filters: []terraformutils.ResourceFilter{
				{
					ServiceName:      "kinesis_stream",
					FieldPath:        "id",
					AcceptableValues: []string{"orders"},
				},
				{
					ServiceName:      "kinesis_stream_consumer",
					FieldPath:        "id",
					AcceptableValues: []string{"arn:aws:kinesis:us-east-1:123456789012:stream/payments/consumer/app:1"},
				},
			},
			stream: "payments",
			want:   true,
		},
		{
			name: "resource policy filter keeps discovery despite nonmatching stream filter",
			filters: []terraformutils.ResourceFilter{
				{
					ServiceName:      "kinesis_stream",
					FieldPath:        "id",
					AcceptableValues: []string{"orders"},
				},
				{
					ServiceName:      "kinesis_resource_policy",
					FieldPath:        "id",
					AcceptableValues: []string{"arn:aws:kinesis:us-east-1:123456789012:stream/payments"},
				},
			},
			stream: "payments",
			want:   true,
		},
		{
			name: "child filter does not suppress stream discovery",
			filters: []terraformutils.ResourceFilter{{
				ServiceName:      "kinesis_stream_consumer",
				FieldPath:        "id",
				AcceptableValues: []string{"arn:aws:kinesis:us-east-1:123456789012:stream/orders/consumer/app:1"},
			}},
			stream: "orders",
			want:   true,
		},
		{
			name: "untyped child id filter reaches cleanup",
			filters: []terraformutils.ResourceFilter{{
				FieldPath:        "id",
				AcceptableValues: []string{"arn:aws:kinesis:us-east-1:123456789012:stream/orders/consumer/app:1"},
			}},
			stream: "orders",
			want:   true,
		},
		{
			name: "non-id stream filter is handled by post-refresh cleanup",
			filters: []terraformutils.ResourceFilter{{
				ServiceName:      "kinesis_stream",
				FieldPath:        "tags.env",
				AcceptableValues: []string{"prod"},
			}},
			stream: "orders",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &KinesisGenerator{}
			g.Filter = tt.filters
			got := g.shouldLoadStreamChildren(tt.stream)
			if got != tt.want {
				t.Fatalf("shouldLoadStreamChildren() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKinesisResourceMissing(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "typed resource not found", err: &kinesistypes.ResourceNotFoundException{}, want: true},
		{name: "wrapped resource not found", err: fmt.Errorf("wrapped: %w", &kinesistypes.ResourceNotFoundException{}), want: true},
		{name: "api resource not found code", err: &smithy.GenericAPIError{Code: "ResourceNotFoundException"}, want: true},
		{name: "other", err: errors.New("boom"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := kinesisResourceMissing(tt.err)
			if got != tt.want {
				t.Fatalf("kinesisResourceMissing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKinesisPostConvertHookWrapsResourcePolicy(t *testing.T) {
	resource := terraformutils.NewSimpleResource(
		"arn:aws:kinesis:us-east-1:123456789012:stream/orders",
		"orders/policy",
		"aws_kinesis_resource_policy",
		"aws",
		kinesisAllowEmptyValues,
	)
	resource.Item = map[string]interface{}{"policy": "{\"Resource\":\"$" + "{aws:kinesis}\"}"}
	g := &KinesisGenerator{}
	g.Resources = []terraformutils.Resource{resource}

	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}

	want := "<<POLICY\n{\"Resource\":\"$" + "$" + "{aws:kinesis}\"}\nPOLICY"
	if got := g.Resources[0].Item["policy"]; got != want {
		t.Fatalf("policy = %q, want %q", got, want)
	}
}
