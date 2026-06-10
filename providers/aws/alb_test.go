// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

func TestListenerSupportsCertificates(t *testing.T) {
	tests := []struct {
		name     string
		protocol types.ProtocolEnum
		want     bool
	}{
		{name: "https", protocol: types.ProtocolEnumHttps, want: true},
		{name: "tls", protocol: types.ProtocolEnumTls, want: true},
		{name: "http", protocol: types.ProtocolEnumHttp, want: false},
		{name: "tcp", protocol: types.ProtocolEnumTcp, want: false},
		{name: "udp", protocol: types.ProtocolEnumUdp, want: false},
		{name: "tcp udp", protocol: types.ProtocolEnumTcpUdp, want: false},
		{name: "geneve", protocol: types.ProtocolEnumGeneve, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := listenerSupportsCertificates(tt.protocol); got != tt.want {
				t.Fatalf("listenerSupportsCertificates(%q) = %t, want %t", tt.protocol, got, tt.want)
			}
		})
	}
}

func TestNewALBTargetGroupAttachmentResource(t *testing.T) {
	targetGroupARN := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/app/abc"
	resource, ok := newALBTargetGroupAttachmentResource(targetGroupARN, &types.TargetDescription{
		Id:               aws.String("10.0.0.12"),
		Port:             aws.Int32(8080),
		AvailabilityZone: aws.String("us-east-1a"),
	})
	if !ok {
		t.Fatal("newALBTargetGroupAttachmentResource() ok = false, want true")
	}
	if got := resource.InstanceInfo.Type; got != "aws_lb_target_group_attachment" {
		t.Fatalf("resource type = %q, want aws_lb_target_group_attachment", got)
	}
	wantID := targetGroupARN + ",10.0.0.12,8080,us-east-1a"
	if got := resource.InstanceState.ID; got != wantID {
		t.Fatalf("state ID = %q, want %q", got, wantID)
	}
	attrs := resource.InstanceState.Attributes
	if attrs["target_group_arn"] != targetGroupARN {
		t.Fatalf("target_group_arn = %q, want %q", attrs["target_group_arn"], targetGroupARN)
	}
	if attrs["target_id"] != "10.0.0.12" {
		t.Fatalf("target_id = %q, want 10.0.0.12", attrs["target_id"])
	}
	if attrs["port"] != "8080" {
		t.Fatalf("port = %q, want 8080", attrs["port"])
	}
	if attrs["availability_zone"] != "us-east-1a" {
		t.Fatalf("availability_zone = %q, want us-east-1a", attrs["availability_zone"])
	}
}

func TestNewALBTargetGroupAttachmentResourceWithAvailabilityZoneOnly(t *testing.T) {
	targetGroupARN := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/app/abc"
	resource, ok := newALBTargetGroupAttachmentResource(targetGroupARN, &types.TargetDescription{
		Id:               aws.String("i-123"),
		AvailabilityZone: aws.String("all"),
	})
	if !ok {
		t.Fatal("newALBTargetGroupAttachmentResource() ok = false, want true")
	}
	wantID := targetGroupARN + ",i-123,,all"
	if got := resource.InstanceState.ID; got != wantID {
		t.Fatalf("state ID = %q, want %q", got, wantID)
	}
}

func TestNewALBTargetGroupAttachmentResourceSkipsIncompleteTargets(t *testing.T) {
	if _, ok := newALBTargetGroupAttachmentResource("", &types.TargetDescription{Id: aws.String("i-123")}); ok {
		t.Fatal("target without target group ARN should be skipped")
	}
	if _, ok := newALBTargetGroupAttachmentResource("target-group", &types.TargetDescription{}); ok {
		t.Fatal("target without ID should be skipped")
	}
	if _, ok := newALBTargetGroupAttachmentResource("target-group", nil); ok {
		t.Fatal("nil target should be skipped")
	}
}
