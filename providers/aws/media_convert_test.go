// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/mediaconvert"
	mediaconverttypes "github.com/aws/aws-sdk-go-v2/service/mediaconvert/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewMediaConvertQueueResource(t *testing.T) {
	resource, ok := newMediaConvertQueueResource(mediaconverttypes.Queue{
		Name: aws.String("transcode"),
		Type: mediaconverttypes.Type("CUSTOM"),
	})
	assertMediaConvertResource(t, resource, ok, "transcode", mediaConvertResourceName("queue", "transcode"), mediaConvertQueueResourceType)
	assertMediaConvertAttribute(t, resource, "name", "transcode")

	if _, ok := newMediaConvertQueueResource(mediaconverttypes.Queue{}); ok {
		t.Fatal("queue with empty name should be skipped")
	}
}

func TestMediaConvertQueueImportable(t *testing.T) {
	tests := []struct {
		name  string
		queue mediaconverttypes.Queue
		want  bool
	}{
		{name: "custom", queue: mediaconverttypes.Queue{Type: mediaconverttypes.Type("CUSTOM")}, want: true},
		{name: "empty type", queue: mediaconverttypes.Queue{}, want: true},
		{name: "system", queue: mediaconverttypes.Queue{Type: mediaconverttypes.TypeSystem}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mediaConvertQueueImportable(tt.queue); got != tt.want {
				t.Fatalf("mediaConvertQueueImportable(%#v) = %t, want %t", tt.queue, got, tt.want)
			}
		})
	}
}

func TestMediaConvertQueueImportID(t *testing.T) {
	if got, want := mediaConvertQueueImportID("transcode"), "transcode"; got != want {
		t.Fatalf("MediaConvert queue import ID = %q, want %q", got, want)
	}
}

func TestMediaConvertResourceNamesPreserveSegmentBoundaries(t *testing.T) {
	left := terraformutils.TfSanitize(mediaConvertResourceName("queue", "a/b_c"))
	right := terraformutils.TfSanitize(mediaConvertResourceName("que", "ue/a_b_c"))
	if left == right {
		t.Fatalf("MediaConvert resource names collide: %q", left)
	}
}

func TestMediaConvertQueueNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "not found", err: &mediaconverttypes.NotFoundException{}, want: true},
		{name: "wrapped not found", err: errors.Join(errors.New("lookup failed"), &mediaconverttypes.NotFoundException{}), want: true},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mediaConvertQueueNotFound(tt.err); got != tt.want {
				t.Fatalf("mediaConvertQueueNotFound(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func TestMediaConvertQueueEndpointUnavailable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "not found", err: &mediaconverttypes.NotFoundException{}, want: false},
		{name: "customer endpoint bad request", err: &mediaconverttypes.BadRequestException{Message: aws.String("You must use the customer-specific endpoint")}, want: true},
		{name: "account endpoint bad request", err: &mediaconverttypes.BadRequestException{Message: aws.String("account endpoint is not available")}, want: true},
		{name: "wrapped endpoint bad request", err: errors.Join(errors.New("list queues failed"), &mediaconverttypes.BadRequestException{Message: aws.String("use the account-specific endpoint")}), want: true},
		{name: "other bad request", err: &mediaconverttypes.BadRequestException{Message: aws.String("invalid maxResults")}, want: false},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mediaConvertQueueEndpointUnavailable(tt.err); got != tt.want {
				t.Fatalf("mediaConvertQueueEndpointUnavailable(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func TestMediaConvertAccountEndpoint(t *testing.T) {
	client := &mediaConvertDescribeEndpointsClient{
		outputs: []*mediaconvert.DescribeEndpointsOutput{
			{
				Endpoints: []mediaconverttypes.Endpoint{{}},
				NextToken: aws.String("next"),
			},
			{
				Endpoints: []mediaconverttypes.Endpoint{{Url: aws.String("https://abcd.mediaconvert.us-east-1.amazonaws.com")}},
			},
		},
	}
	endpoint, err := mediaConvertAccountEndpoint(context.Background(), client)
	if err != nil {
		t.Fatalf("mediaConvertAccountEndpoint returned error: %v", err)
	}
	if got, want := endpoint, "https://abcd.mediaconvert.us-east-1.amazonaws.com"; got != want {
		t.Fatalf("mediaConvert account endpoint = %q, want %q", got, want)
	}
	if got, want := len(client.inputs), 2; got != want {
		t.Fatalf("DescribeEndpoints calls = %d, want %d", got, want)
	}
	for _, input := range client.inputs {
		if got, want := input.Mode, mediaconverttypes.DescribeEndpointsModeGetOnly; got != want {
			t.Fatalf("DescribeEndpoints mode = %q, want %q", got, want)
		}
	}
}

func TestMediaConvertAccountEndpointEmpty(t *testing.T) {
	client := &mediaConvertDescribeEndpointsClient{
		outputs: []*mediaconvert.DescribeEndpointsOutput{{}},
	}
	endpoint, err := mediaConvertAccountEndpoint(context.Background(), client)
	if err != nil {
		t.Fatalf("mediaConvertAccountEndpoint returned error: %v", err)
	}
	if endpoint != "" {
		t.Fatalf("MediaConvert account endpoint = %q, want empty", endpoint)
	}
}

func TestMediaConvertAccountEndpointError(t *testing.T) {
	wantErr := errors.New("describe endpoints failed")
	client := &mediaConvertDescribeEndpointsClient{errs: []error{wantErr}}
	endpoint, err := mediaConvertAccountEndpoint(context.Background(), client)
	if !errors.Is(err, wantErr) {
		t.Fatalf("mediaConvertAccountEndpoint error = %v, want %v", err, wantErr)
	}
	if endpoint != "" {
		t.Fatalf("MediaConvert account endpoint = %q, want empty", endpoint)
	}
}

type mediaConvertDescribeEndpointsClient struct {
	outputs []*mediaconvert.DescribeEndpointsOutput
	errs    []error
	inputs  []*mediaconvert.DescribeEndpointsInput
	calls   int
}

func (c *mediaConvertDescribeEndpointsClient) DescribeEndpoints(_ context.Context, input *mediaconvert.DescribeEndpointsInput, _ ...func(*mediaconvert.Options)) (*mediaconvert.DescribeEndpointsOutput, error) {
	c.inputs = append(c.inputs, input)
	call := c.calls
	c.calls++
	if call < len(c.errs) && c.errs[call] != nil {
		return nil, c.errs[call]
	}
	if call < len(c.outputs) {
		return c.outputs[call], nil
	}
	return &mediaconvert.DescribeEndpointsOutput{}, nil
}

func assertMediaConvertResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantName, wantType string) {
	t.Helper()
	if !ok {
		t.Fatal("resource was skipped")
	}
	if got := resource.InstanceState.ID; got != wantID {
		t.Fatalf("resource ID = %q, want %q", got, wantID)
	}
	if got := resource.ResourceName; got != terraformutils.TfSanitize(wantName) {
		t.Fatalf("resource name = %q, want %q", got, terraformutils.TfSanitize(wantName))
	}
	if got := resource.InstanceInfo.Type; got != wantType {
		t.Fatalf("resource type = %q, want %q", got, wantType)
	}
}

func assertMediaConvertAttribute(t *testing.T, resource terraformutils.Resource, key, want string) {
	t.Helper()
	if got := resource.InstanceState.Attributes[key]; got != want {
		t.Fatalf("resource attribute %q = %q, want %q", key, got, want)
	}
}
